package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"matrix/api/internal/collector"
	"matrix/api/internal/model"
	"matrix/api/internal/repository"
	"matrix/api/pkg/log"
)

// TaskService 任务管理业务逻辑接口
type TaskService interface {
	ListTasks() ([]*model.Task, error)
	GetTask(id string) (*model.Task, error)
	CreateTask(req *model.CreateTaskRequest) (*model.Task, error)
	UpdateTask(id string, req *model.UpdateTaskRequest) (*model.Task, error)
	DeleteTask(id string) error
	EnableTask(id string) (*model.Task, error)
	DisableTask(id string) (*model.Task, error)
	// RunNow 立即触发一次采集，不等待下一个周期
	RunNow(id string) error
	// Stop 停止后台调度器，优雅关闭时调用
	Stop()
}

type taskSvc struct {
	repo   repository.TaskRepository
	stopCh chan struct{}
}

func newTaskSvc(repo repository.TaskRepository) TaskService {
	svc := &taskSvc{repo: repo, stopCh: make(chan struct{})}
	go svc.scheduler()
	return svc
}

func (s *taskSvc) ListTasks() ([]*model.Task, error) {
	return s.repo.List()
}

func (s *taskSvc) GetTask(id string) (*model.Task, error) {
	return s.repo.Get(id)
}

func (s *taskSvc) CreateTask(req *model.CreateTaskRequest) (*model.Task, error) {
	// 校验采集器类型是否注册，避免创建后无法执行
	if _, err := collector.Get(req.CollectorType); err != nil {
		return nil, fmt.Errorf("无效采集器类型: %w", err)
	}
	next := time.Now().Add(time.Duration(req.IntervalSecs) * time.Second)
	task := &model.Task{
		Name:          req.Name,
		CollectorType: req.CollectorType,
		Config:        req.Config,
		Enabled:       true,
		IntervalSecs:  req.IntervalSecs,
		NextRunAt:     &next,
	}
	return s.repo.Create(task)
}

func (s *taskSvc) UpdateTask(id string, req *model.UpdateTaskRequest) (*model.Task, error) {
	task, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Config != nil {
		task.Config = req.Config
	}
	if req.IntervalSecs != nil {
		task.IntervalSecs = *req.IntervalSecs
		next := time.Now().Add(time.Duration(*req.IntervalSecs) * time.Second)
		task.NextRunAt = &next
	}
	return s.repo.Update(task)
}

func (s *taskSvc) DeleteTask(id string) error {
	return s.repo.Delete(id)
}

func (s *taskSvc) EnableTask(id string) (*model.Task, error) {
	task, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	task.Enabled = true
	task.Status = "idle"
	next := time.Now().Add(time.Duration(task.IntervalSecs) * time.Second)
	task.NextRunAt = &next
	return s.repo.Update(task)
}

func (s *taskSvc) DisableTask(id string) (*model.Task, error) {
	task, err := s.repo.Get(id)
	if err != nil {
		return nil, err
	}
	task.Enabled = false
	task.Status = "idle"
	return s.repo.Update(task)
}

func (s *taskSvc) RunNow(id string) error {
	task, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	// 立即标记 running，避免调度器同时触发
	task.Status = "running"
	if _, err = s.repo.Update(task); err != nil {
		return err
	}
	go s.execute(task)
	return nil
}

func (s *taskSvc) Stop() {
	close(s.stopCh)
}

// scheduler 后台调度循环，每秒检查一次到期任务，直到 Stop() 被调用。
func (s *taskSvc) scheduler() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.tick()
		case <-s.stopCh:
			return
		}
	}
}

func (s *taskSvc) tick() {
	tasks, _ := s.repo.List()
	now := time.Now()
	for _, task := range tasks {
		if !task.Enabled || task.Status == "running" {
			continue
		}
		if task.NextRunAt != nil && now.Before(*task.NextRunAt) {
			continue
		}
		// 同步标记 running，防止下一个 tick 重复触发
		task.Status = "running"
		if _, err := s.repo.Update(task); err != nil {
			continue
		}
		go s.execute(task)
	}
}

func (s *taskSvc) execute(task *model.Task) {
	c, err := collector.Get(task.CollectorType)
	if err != nil {
		s.finalize(task.ID, "error", nil, nil, "", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := c.Collect(ctx, task.Config)
	now := time.Now()
	next := now.Add(time.Duration(task.IntervalSecs) * time.Second)

	if err != nil {
		log.Logger.Warnf("task %q collect error: %v", task.Name, err)
		s.finalize(task.ID, "error", &now, &next, "", err.Error())
		return
	}

	resultJSON, _ := json.Marshal(result)
	log.Logger.Infof("task %q collect done: %s", task.Name, result.Summary)
	s.finalize(task.ID, "idle", &now, &next, string(resultJSON), "")
}

func (s *taskSvc) finalize(id, status string, lastRun, nextRun *time.Time, result, errMsg string) {
	task, err := s.repo.Get(id)
	if err != nil {
		return // 任务在执行期间被删除，忽略
	}
	task.Status = status
	task.LastResult = result
	task.LastError = errMsg
	if lastRun != nil {
		task.LastRunAt = lastRun
	}
	if nextRun != nil {
		task.NextRunAt = nextRun
	}
	s.repo.Update(task) //nolint:errcheck
}
