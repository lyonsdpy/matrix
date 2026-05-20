package loader

import (
	"context"
	"sync"
	"time"

	"matrix/api/domain"
	"matrix/api/internal/repository"
)

type contextKey string

// Key 用于在 Gin context 中存取 Loaders，避免与其他 key 冲突
const Key contextKey = "gql_loaders"

// Loaders 是每个 HTTP 请求独立创建的 DataLoader 集合。
// 通过 GinMiddleware 注入 context，resolver 通过 ForContext 取出使用。
type Loaders struct {
	DeviceIPs *IPBatchLoader
}

// New 为当前请求创建新的 Loaders 实例
func New(repo *repository.DeviceRepo) *Loaders {
	return &Loaders{
		DeviceIPs: newIPBatchLoader(repo),
	}
}

// ForContext 从 context 取出当前请求的 Loaders
func ForContext(ctx context.Context) *Loaders {
	return ctx.Value(Key).(*Loaders)
}

// ─────────────────────────────────────────────────
// IPBatchLoader：DataLoader 核心实现
//
// 问题：GraphQL 查询 100 台设备的 ips 字段时，
//        naïve 实现会触发 100 次独立 DB 查询（N+1 问题）。
//
// 解法：在 1ms 批量窗口内收集所有 Load(deviceID) 调用，
//        合并为一次 BatchListIPsByDeviceIDs 调用（相当于 SQL IN 查询）。
//
// 时序图（查询 3 台设备的 ips）：
//   resolver-A calls Load("dev-001")  ─┐
//   resolver-B calls Load("dev-002")  ─┤→ 1ms 后 dispatch
//   resolver-C calls Load("dev-003")  ─┘      └→ BatchListIPs(["dev-001","dev-002","dev-003"])
//                                              └→ 结果分发给各自的 channel
// ─────────────────────────────────────────────────
type IPBatchLoader struct {
	repo *repository.DeviceRepo

	mu      sync.Mutex
	pending map[string][]chan ipResult // deviceID -> 等待该结果的 channel 列表
	timer   *time.Timer               // 批量窗口定时器，第一次 Load 时创建
}

type ipResult struct {
	ips []*domain.IPv4Addr
	err error
}

func newIPBatchLoader(repo *repository.DeviceRepo) *IPBatchLoader {
	return &IPBatchLoader{
		repo:    repo,
		pending: make(map[string][]chan ipResult),
	}
}

// Load 注册一个 deviceID 进入当前批次，阻塞直到批量结果返回。
// gqlgen 会并发调用多个字段的 resolver，所有 Load 调用在 1ms 内积累后统一执行。
func (l *IPBatchLoader) Load(ctx context.Context, deviceID string) ([]*domain.IPv4Addr, error) {
	ch := make(chan ipResult, 1)

	l.mu.Lock()
	l.pending[deviceID] = append(l.pending[deviceID], ch)
	if l.timer == nil {
		// 1ms 后执行批量查询：权衡点是 1ms 额外延迟 vs 消除 N 次独立查询
		l.timer = time.AfterFunc(time.Millisecond, func() { l.dispatch(ctx) })
	}
	l.mu.Unlock()

	r := <-ch
	return r.ips, r.err
}

// dispatch 在批量窗口结束时执行，将积累的 ID 合并为一次 repo 调用
func (l *IPBatchLoader) dispatch(ctx context.Context) {
	l.mu.Lock()
	pending := l.pending
	l.pending = make(map[string][]chan ipResult) // 重置，为下一批次做准备
	l.timer = nil
	l.mu.Unlock()

	ids := make([]string, 0, len(pending))
	for id := range pending {
		ids = append(ids, id)
	}

	// 一次批量查询替代 N 次独立查询
	ipsByDevice, err := l.repo.BatchListIPsByDeviceIDs(ctx, ids)

	// 将结果分发给所有等待的 caller
	for deviceID, waiters := range pending {
		var r ipResult
		if err != nil {
			r.err = err
		} else {
			r.ips = ipsByDevice[deviceID]
		}
		for _, ch := range waiters {
			ch <- r
		}
	}
}
