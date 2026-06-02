package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"matrix/api/internal/infra/lark"
	"matrix/api/pkg/log"
)

// ── 接口（service 包"接口定义在使用方"原则）────────────────────────────────────

// endpointSyncRepo EndpointSyncService 所需的图能力，与 syncEndpointRepo 区分：
// 这里是终端管理"独立同步"专用，签名相同但语义上属于另一条同步链路。
type endpointSyncRepo interface {
	BulkUpsert(ctx context.Context, batch []lark.Device) error
	BulkLinkLogin(ctx context.Context, batch []lark.Device) error
	ListAllFeishuDeviceIDs(ctx context.Context) (map[string]struct{}, error)
	DeleteByFeishuDeviceIDs(ctx context.Context, ids []string) (int, error)
}

// endpointSyncFetcher 从飞书拉终端清单。
type endpointSyncFetcher interface {
	FetchAllDevices(ctx context.Context) ([]lark.Device, error)
}

// ── 进度状态 ─────────────────────────────────────────────────────────────────

// EndpointSyncPhase 终端同步阶段标识。done/failed 为终态。
type EndpointSyncPhase string

const (
	EndpointPhaseIdle         EndpointSyncPhase = "idle"
	EndpointPhaseFetchDevices EndpointSyncPhase = "fetch_devices"
	EndpointPhaseWriteDevices EndpointSyncPhase = "write_devices"
	EndpointPhaseLinkLogins   EndpointSyncPhase = "link_logins"
	EndpointPhaseFinalizing   EndpointSyncPhase = "finalizing"
	EndpointPhaseDone         EndpointSyncPhase = "done"
	EndpointPhaseFailed       EndpointSyncPhase = "failed"
)

// EndpointSyncProgress 终端同步任务的可观测快照，HTTP 接口直接 JSON 输出。
// 与 SyncProgress 结构对称但字段集合不同（仅有 endpoints 一项）。
type EndpointSyncProgress struct {
	JobID      string            `json:"job_id"`
	Phase      EndpointSyncPhase `json:"phase"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt *time.Time        `json:"finished_at,omitempty"`
	DurationMs int64             `json:"duration_ms"`

	Done  int `json:"done"`
	Total int `json:"total"`

	Endpoints SyncCounts `json:"endpoints"`

	Error string `json:"error,omitempty"`
}

// ── Service ───────────────────────────────────────────────────────────────

// EndpointSyncService 飞书设备同步服务：独立的 running 锁与进度，与 SyncService 完全解耦。
//
// 设计与 SyncService 对齐：单实例 job 守卫（atomic.Bool）；进度写内存（progress + progressMu）；
// 用独立 background ctx 跑 goroutine，HTTP 请求 ctx 结束不影响任务执行。
//
// 顺序：拉设备 → 写设备 → 建登录边 → 收尾(diff 删除)；任一阶段失败即终止整任务并置 PhaseFailed。
// 建登录边按 row.user_id 匹配 (User {user_id: ...})；通讯录未同步的边缘用户会在 cypher 内静默跳过。
type EndpointSyncService struct {
	fetcher endpointSyncFetcher
	repo    endpointSyncRepo

	jobMu   sync.Mutex
	running atomic.Bool

	progressMu sync.RWMutex
	progress   *EndpointSyncProgress
}

// NewEndpointSyncService 构造。deviceFetcher 为 nil 时 Start 直接返回错误。
func NewEndpointSyncService(fetcher endpointSyncFetcher, repo endpointSyncRepo) *EndpointSyncService {
	return &EndpointSyncService{fetcher: fetcher, repo: repo}
}

// Start 触发一次设备同步。若有 job 在跑返回 (现 jobID, false, nil)。
func (s *EndpointSyncService) Start(_ context.Context) (jobID string, started bool, err error) {
	if s.fetcher == nil {
		return "", false, fmt.Errorf("endpoint sync: lark device fetcher 未配置（缺少 lark.app_id/app_secret）")
	}
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if s.running.Load() {
		s.progressMu.RLock()
		cur := s.progress
		s.progressMu.RUnlock()
		if cur != nil {
			return cur.JobID, false, nil
		}
		return "", false, nil
	}
	jobID = uuid.New().String()
	now := time.Now()
	s.progressMu.Lock()
	s.progress = &EndpointSyncProgress{
		JobID:     jobID,
		Phase:     EndpointPhaseFetchDevices,
		StartedAt: now,
	}
	s.progressMu.Unlock()
	s.running.Store(true)

	runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	go func() {
		defer cancel()
		defer s.running.Store(false)
		s.runSync(runCtx, jobID)
	}()
	return jobID, true, nil
}

// Snapshot 返回当前/最近 job 的进度快照（值拷贝）。
func (s *EndpointSyncService) Snapshot() EndpointSyncProgress {
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()
	if s.progress == nil {
		return EndpointSyncProgress{Phase: EndpointPhaseIdle}
	}
	p := *s.progress
	if p.FinishedAt == nil {
		p.DurationMs = time.Since(p.StartedAt).Milliseconds()
	}
	return p
}

// ── 内部执行 ───────────────────────────────────────────────────────────────

const endpointSyncBatchSize = 500

func (s *EndpointSyncService) runSync(ctx context.Context, jobID string) {
	defer func() {
		if rec := recover(); rec != nil {
			s.finishFailure(fmt.Errorf("endpoint sync panic: %v", rec))
		}
	}()

	log.Logger.Infof("endpoint sync: job %s started", jobID)

	// 0) 拍快照供 diff
	existingIDs, err := s.repo.ListAllFeishuDeviceIDs(ctx)
	if err != nil {
		s.finishFailure(fmt.Errorf("snapshot endpoint ids: %w", err))
		return
	}

	// 1) 拉设备
	s.setPhase(EndpointPhaseFetchDevices, 0, 0)
	devs, err := s.fetcher.FetchAllDevices(ctx)
	if err != nil {
		s.finishFailure(fmt.Errorf("fetch devices: %w", err))
		return
	}
	devices := filterDevicesWithID(devs)

	// 2) 写设备节点
	s.setPhase(EndpointPhaseWriteDevices, 0, len(devices))
	for chunk := range chunkSlice(devices, endpointSyncBatchSize) {
		if err := s.repo.BulkUpsert(ctx, chunk); err != nil {
			s.finishFailure(fmt.Errorf("bulk upsert endpoints: %w", err))
			return
		}
		s.advance(len(chunk))
	}

	// 3) 重建登录边（CURRENT_LOGIN / LATEST_LOGIN）
	s.setPhase(EndpointPhaseLinkLogins, 0, len(devices))
	for chunk := range chunkSlice(devices, endpointSyncBatchSize) {
		if err := s.repo.BulkLinkLogin(ctx, chunk); err != nil {
			s.finishFailure(fmt.Errorf("bulk link login: %w", err))
			return
		}
		s.advance(len(chunk))
	}

	// 4) 收尾：diff 出 deleted
	s.setPhase(EndpointPhaseFinalizing, 0, 0)
	fetched := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		fetched[d.DeviceID] = struct{}{}
	}
	deleted := diffIDs(existingIDs, fetched)
	delN, err := s.repo.DeleteByFeishuDeviceIDs(ctx, deleted)
	if err != nil {
		s.finishFailure(fmt.Errorf("delete obsolete endpoints: %w", err))
		return
	}

	// 5) 汇总
	counts := SyncCounts{Total: len(devices), Deleted: delN}
	for _, d := range devices {
		if _, ok := existingIDs[d.DeviceID]; ok {
			counts.Updated++
		} else {
			counts.Created++
		}
	}
	s.finishSuccess(counts)
	log.Logger.Infof("endpoint sync: job %s done — endpoints(+%d ~%d -%d)",
		jobID, counts.Created, counts.Updated, counts.Deleted,
	)
}

// ── 进度更新辅助（与 SyncService 平行） ──────────────────────────────────────

func (s *EndpointSyncService) setPhase(p EndpointSyncPhase, done, total int) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	s.progress.Phase = p
	s.progress.Done = done
	s.progress.Total = total
}

func (s *EndpointSyncService) advance(n int) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	s.progress.Done += n
}

func (s *EndpointSyncService) finishSuccess(counts SyncCounts) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	now := time.Now()
	s.progress.Phase = EndpointPhaseDone
	s.progress.FinishedAt = &now
	s.progress.DurationMs = now.Sub(s.progress.StartedAt).Milliseconds()
	s.progress.Endpoints = counts
}

func (s *EndpointSyncService) finishFailure(err error) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	now := time.Now()
	s.progress.Phase = EndpointPhaseFailed
	s.progress.FinishedAt = &now
	s.progress.DurationMs = now.Sub(s.progress.StartedAt).Milliseconds()
	s.progress.Error = err.Error()
	log.Logger.Errorf("endpoint sync: job %s failed: %v", s.progress.JobID, err)
}
