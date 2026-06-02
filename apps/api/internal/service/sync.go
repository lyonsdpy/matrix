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

// syncDeptRepo SyncService 所需的部门图能力。
type syncDeptRepo interface {
	BulkUpsert(ctx context.Context, batch []lark.Department) error
	BulkLinkParents(ctx context.Context, batch []lark.Department) error
	BulkLinkMembers(ctx context.Context, batch []lark.User) error
	ListAllIDs(ctx context.Context) (map[string]struct{}, error)
	DeleteByIDs(ctx context.Context, ids []string) (int, error)
}

// syncUserRepo SyncService 所需的用户图能力。
type syncUserRepo interface {
	BulkUpsertSyncedUsers(ctx context.Context, batch []lark.User) error
	ListAllFeishuIDs(ctx context.Context) (map[string]struct{}, error)
	DeleteByFeishuIDs(ctx context.Context, ids []string) (int, error)
}

// syncContactFetcher 从飞书拉部门 / 用户的能力。
type syncContactFetcher interface {
	FetchAllDepartments(ctx context.Context) ([]lark.Department, error)
	FetchUsersInDepartments(ctx context.Context, deptIDs []string, opts lark.FetchUsersOptions) ([]lark.User, error)
}

// ── 进度状态（前端轮询返回的就是 Snapshot）───────────────────────────────────

// SyncPhase 通讯录同步阶段标识。done/failed 为终态。
type SyncPhase string

const (
	PhaseIdle             SyncPhase = "idle"
	PhaseFetchDepartments SyncPhase = "fetch_departments"
	PhaseWriteDepartments SyncPhase = "write_departments"
	PhaseFetchUsers       SyncPhase = "fetch_users"
	PhaseWriteUsers       SyncPhase = "write_users"
	PhaseFinalizing       SyncPhase = "finalizing"
	PhaseDone             SyncPhase = "done"
	PhaseFailed           SyncPhase = "failed"
)

// SyncCounts 单个集合的 created/updated/deleted/total 计数。
type SyncCounts struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
	Total   int `json:"total"` // 飞书拉到的总数
}

// SyncProgress 通讯录同步任务的可观测快照，HTTP 接口直接 JSON 输出。
type SyncProgress struct {
	JobID      string     `json:"job_id"`
	Phase      SyncPhase  `json:"phase"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs int64      `json:"duration_ms"`

	// 进度计数：根据 phase 不同含义不同（部门阶段或用户阶段的 done/total）
	Done  int `json:"done"`
	Total int `json:"total"`

	// 完成后的差异统计
	Departments SyncCounts `json:"departments"`
	Users       SyncCounts `json:"users"`

	// 失败信息
	Error string `json:"error,omitempty"`
}

// ── Service ───────────────────────────────────────────────────────────────

// SyncService 飞书通讯录同步服务：触发一次同步、查询进度、保留最近结果。
// 仅负责部门 + 用户；设备同步由 EndpointSyncService 独立管理。
//
// 设计：同一时刻只允许一个 job（用 jobMu 守护）；进度写在内存（progress + progressMu），
// 完成后保留一段时间方便前端拿结果；不需要持久化任务表——同步任务幂等，重跑成本低。
type SyncService struct {
	fetcher  syncContactFetcher
	userRepo syncUserRepo
	deptRepo syncDeptRepo

	jobMu   sync.Mutex // 跨"开始一次新 job"的串行化
	running atomic.Bool

	progressMu sync.RWMutex
	progress   *SyncProgress // 当前/最近一次 job 的进度
}

// NewSyncService 构造。contactFetcher 为 nil 时 Start 直接返回错误。
func NewSyncService(
	contactFetcher syncContactFetcher,
	userRepo syncUserRepo,
	deptRepo syncDeptRepo,
) *SyncService {
	return &SyncService{
		fetcher:  contactFetcher,
		userRepo: userRepo,
		deptRepo: deptRepo,
	}
}

// Start 触发一次同步：若有 job 在跑返回 (现 jobID, false, nil)，否则启动 goroutine 跑并返回 (新 jobID, true, nil)。
// fetcher 未配置返回错误。
func (s *SyncService) Start(ctx context.Context) (jobID string, started bool, err error) {
	if s.fetcher == nil {
		return "", false, fmt.Errorf("sync: lark fetcher 未配置（缺少 lark.app_id/app_secret）")
	}
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if s.running.Load() {
		// 已在跑，把当前 jobID 告诉调用方
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
	s.progress = &SyncProgress{
		JobID:     jobID,
		Phase:     PhaseFetchDepartments,
		StartedAt: now,
	}
	s.progressMu.Unlock()
	s.running.Store(true)

	// 不复用调用方 ctx：HTTP 请求结束后 ctx 会被取消，但 sync 要继续跑。
	// 这里用独立 background ctx，但保留超时（20 分钟）兜底。
	runCtx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	go func() {
		defer cancel()
		defer s.running.Store(false)
		s.runSync(runCtx, jobID)
	}()
	return jobID, true, nil
}

// Snapshot 返回当前/最近 job 的进度快照（值拷贝，调用方安全使用）。
// 没有任何 job 跑过时返回 PhaseIdle 状态。
func (s *SyncService) Snapshot() SyncProgress {
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()
	if s.progress == nil {
		return SyncProgress{Phase: PhaseIdle}
	}
	p := *s.progress
	if p.FinishedAt == nil {
		p.DurationMs = time.Since(p.StartedAt).Milliseconds()
	}
	return p
}

// ── 内部执行 ───────────────────────────────────────────────────────────────

const (
	syncDeptBatchSize = 500
	syncUserBatchSize = 500
)

func (s *SyncService) runSync(ctx context.Context, jobID string) {
	defer func() {
		// goroutine 内 panic 不能让进程挂掉；记到进度里
		if rec := recover(); rec != nil {
			s.finishFailure(fmt.Errorf("sync panic: %v", rec))
		}
	}()

	log.Logger.Infof("sync: job %s started", jobID)

	// 0) 先拍快照：原 ID 集合用于事后 diff
	existingDeptIDs, err := s.deptRepo.ListAllIDs(ctx)
	if err != nil {
		s.finishFailure(fmt.Errorf("snapshot dept ids: %w", err))
		return
	}
	existingUserIDs, err := s.userRepo.ListAllFeishuIDs(ctx)
	if err != nil {
		s.finishFailure(fmt.Errorf("snapshot user feishu_ids: %w", err))
		return
	}

	// 1) 拉部门
	s.setPhase(PhaseFetchDepartments, 0, 0)
	depts, err := s.fetcher.FetchAllDepartments(ctx)
	if err != nil {
		s.finishFailure(fmt.Errorf("fetch departments: %w", err))
		return
	}

	// 2) 写部门（批量 upsert + 批量建 PARENT_OF）
	s.setPhase(PhaseWriteDepartments, 0, len(depts))
	for chunk := range chunkSlice(depts, syncDeptBatchSize) {
		if err := s.deptRepo.BulkUpsert(ctx, chunk); err != nil {
			s.finishFailure(fmt.Errorf("bulk upsert departments: %w", err))
			return
		}
		s.advance(len(chunk))
	}
	if err := s.deptRepo.BulkLinkParents(ctx, depts); err != nil {
		s.finishFailure(fmt.Errorf("bulk link parents: %w", err))
		return
	}

	// 3) 拉用户（并发，按部门 ID 列表逐个拉）
	deptIDs := make([]string, 0, len(depts))
	for _, d := range depts {
		if d.DepartmentID != "" {
			deptIDs = append(deptIDs, d.DepartmentID)
		}
	}
	s.setPhase(PhaseFetchUsers, 0, len(deptIDs))
	users, err := s.fetcher.FetchUsersInDepartments(ctx, deptIDs, lark.FetchUsersOptions{
		OnProgress: func(done, total int) { s.setPhaseCounts(done, total) },
	})
	if err != nil {
		s.finishFailure(fmt.Errorf("fetch users: %w", err))
		return
	}

	// 过滤掉缺 OpenID 的脏数据，避免后续 MERGE 写空键
	users = filterUsersWithOpenID(users)

	// 4) 写用户
	s.setPhase(PhaseWriteUsers, 0, len(users))
	for chunk := range chunkSlice(users, syncUserBatchSize) {
		if err := s.userRepo.BulkUpsertSyncedUsers(ctx, chunk); err != nil {
			s.finishFailure(fmt.Errorf("bulk upsert users: %w", err))
			return
		}
		if err := s.deptRepo.BulkLinkMembers(ctx, chunk); err != nil {
			s.finishFailure(fmt.Errorf("bulk link members: %w", err))
			return
		}
		s.advance(len(chunk))
	}

	// 5) 收尾：diff 出 deleted，DETACH DELETE
	s.setPhase(PhaseFinalizing, 0, 0)
	fetchedDept := make(map[string]struct{}, len(depts))
	for _, d := range depts {
		if d.DepartmentID != "" {
			fetchedDept[d.DepartmentID] = struct{}{}
		}
	}
	fetchedUser := make(map[string]struct{}, len(users))
	for _, u := range users {
		fetchedUser[u.OpenID] = struct{}{}
	}

	deptDeleted := diffIDs(existingDeptIDs, fetchedDept)
	userDeleted := diffIDs(existingUserIDs, fetchedUser)

	delDept, err := s.deptRepo.DeleteByIDs(ctx, deptDeleted)
	if err != nil {
		s.finishFailure(fmt.Errorf("delete obsolete departments: %w", err))
		return
	}
	delUser, err := s.userRepo.DeleteByFeishuIDs(ctx, userDeleted)
	if err != nil {
		s.finishFailure(fmt.Errorf("delete obsolete users: %w", err))
		return
	}

	// 6) 完成：汇总差异统计
	deptCounts := SyncCounts{Total: len(depts), Deleted: delDept}
	userCounts := SyncCounts{Total: len(users), Deleted: delUser}
	for _, d := range depts {
		if d.DepartmentID == "" {
			continue
		}
		if _, ok := existingDeptIDs[d.DepartmentID]; ok {
			deptCounts.Updated++
		} else {
			deptCounts.Created++
		}
	}
	for _, u := range users {
		if _, ok := existingUserIDs[u.OpenID]; ok {
			userCounts.Updated++
		} else {
			userCounts.Created++
		}
	}

	s.finishSuccess(deptCounts, userCounts)
	log.Logger.Infof("sync: job %s done — depts(+%d ~%d -%d) users(+%d ~%d -%d)",
		jobID,
		deptCounts.Created, deptCounts.Updated, deptCounts.Deleted,
		userCounts.Created, userCounts.Updated, userCounts.Deleted,
	)
}

// ── 进度更新辅助 ─────────────────────────────────────────────────────────────

func (s *SyncService) setPhase(p SyncPhase, done, total int) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	s.progress.Phase = p
	s.progress.Done = done
	s.progress.Total = total
}

func (s *SyncService) setPhaseCounts(done, total int) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	s.progress.Done = done
	s.progress.Total = total
}

func (s *SyncService) advance(n int) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	s.progress.Done += n
}

func (s *SyncService) finishSuccess(dept, user SyncCounts) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	now := time.Now()
	s.progress.Phase = PhaseDone
	s.progress.FinishedAt = &now
	s.progress.DurationMs = now.Sub(s.progress.StartedAt).Milliseconds()
	s.progress.Departments = dept
	s.progress.Users = user
}

func (s *SyncService) finishFailure(err error) {
	s.progressMu.Lock()
	defer s.progressMu.Unlock()
	if s.progress == nil {
		return
	}
	now := time.Now()
	s.progress.Phase = PhaseFailed
	s.progress.FinishedAt = &now
	s.progress.DurationMs = now.Sub(s.progress.StartedAt).Milliseconds()
	s.progress.Error = err.Error()
	log.Logger.Errorf("sync: job %s failed: %v", s.progress.JobID, err)
}

// ── 工具函数 ─────────────────────────────────────────────────────────────────

// chunkSlice 把切片切成固定大小的 chunk，用 iterator 形式避免一次性拷贝。
func chunkSlice[T any](s []T, size int) <-chan []T {
	ch := make(chan []T)
	go func() {
		defer close(ch)
		for i := 0; i < len(s); i += size {
			end := i + size
			if end > len(s) {
				end = len(s)
			}
			ch <- s[i:end]
		}
	}()
	return ch
}

// diffIDs 计算 a 中存在但 b 中不存在的键，结果稳定（map 迭代顺序不保证，但同步流程不依赖顺序）。
func diffIDs(a, b map[string]struct{}) []string {
	out := make([]string, 0)
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

func filterUsersWithOpenID(users []lark.User) []lark.User {
	out := users[:0]
	for _, u := range users {
		if u.OpenID != "" {
			out = append(out, u)
		}
	}
	return out
}

func filterDevicesWithID(devs []lark.Device) []lark.Device {
	out := devs[:0]
	for _, d := range devs {
		if d.DeviceID != "" {
			out = append(out, d)
		}
	}
	return out
}
