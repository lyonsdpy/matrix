package repository

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"matrix/api/internal/model"
)

// TaskRepository 任务数据访问接口。
// 当前为内存实现，可按需替换为 Postgres 实现。
type TaskRepository interface {
	List() ([]*model.Task, error)
	Get(id string) (*model.Task, error)
	Create(task *model.Task) (*model.Task, error)
	Update(task *model.Task) (*model.Task, error)
	Delete(id string) error
}

type taskRepo struct {
	mu    sync.RWMutex
	tasks map[string]*model.Task
}

func newTaskRepo() TaskRepository {
	return &taskRepo{tasks: make(map[string]*model.Task)}
}

func (r *taskRepo) List() ([]*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*model.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		cp := *t
		result = append(result, &cp)
	}
	return result, nil
}

func (r *taskRepo) Get(id string) (*model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}
	cp := *t
	return &cp, nil
}

func (r *taskRepo) Create(task *model.Task) (*model.Task, error) {
	now := time.Now()
	task.ID = uuid.New().String()
	task.Status = "idle"
	task.CreatedAt = now
	task.UpdatedAt = now
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[task.ID] = task
	cp := *task
	return &cp, nil
}

func (r *taskRepo) Update(task *model.Task) (*model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[task.ID]; !ok {
		return nil, fmt.Errorf("task %q not found", task.ID)
	}
	task.UpdatedAt = time.Now()
	r.tasks[task.ID] = task
	cp := *task
	return &cp, nil
}

func (r *taskRepo) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[id]; !ok {
		return fmt.Errorf("task %q not found", id)
	}
	delete(r.tasks, id)
	return nil
}
