package repository

import (
	"context"
	"fmt"
	"matrix/api/domain"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// DeviceRepo 是设备的内存仓库，演示原子化数据访问。
// 生产环境替换为 SQL/NoSQL 实现，接口不变。
type DeviceRepo struct {
	mu      sync.RWMutex
	devices map[string]*domain.Device
	ips     map[string][]*domain.IPv4Addr
	links   map[string][]*domain.DeviceLink // deviceID -> 出向连接列表
}

func newDeviceRepo() *DeviceRepo {
	return &DeviceRepo{
		devices: make(map[string]*domain.Device),
		ips:     make(map[string][]*domain.IPv4Addr),
		links:   make(map[string][]*domain.DeviceLink),
	}
}

// ListDevices 支持游标分页，after 为上一页末尾设备的 ID。
func (r *DeviceRepo) ListDevices(_ context.Context, first int, after string) ([]*domain.Device, bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.devices))
	for id := range r.devices {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start := 0
	if after != "" {
		for i, id := range ids {
			if id == after {
				start = i + 1
				break
			}
		}
	}
	ids = ids[start:]

	hasNext := len(ids) > first
	if hasNext {
		ids = ids[:first]
	}

	out := make([]*domain.Device, 0, len(ids))
	for _, id := range ids {
		out = append(out, r.devices[id])
	}

	var endCursor string
	if len(out) > 0 {
		endCursor = out[len(out)-1].ID
	}
	return out, hasNext, endCursor, nil
}

func (r *DeviceRepo) GetDevice(_ context.Context, id string) (*domain.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.devices[id]
	if !ok {
		return nil, fmt.Errorf("device %s not found", id)
	}
	return d, nil
}

func (r *DeviceRepo) CreateDevice(_ context.Context, name, deviceType, mip string) (*domain.Device, error) {
	d := &domain.Device{
		ID:   uuid.New().String(),
		Name: name,
		Type: deviceType,
		MIP:  mip,
	}
	r.mu.Lock()
	r.devices[d.ID] = d
	r.ips[d.ID] = nil
	r.links[d.ID] = nil
	r.mu.Unlock()
	return d, nil
}

func (r *DeviceRepo) DeleteDevice(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.devices[id]; !ok {
		return false, nil
	}
	delete(r.devices, id)
	delete(r.ips, id)
	delete(r.links, id)
	return true, nil
}

// BatchIPsByDeviceIDs 是 DataLoader 的批量 IP 查询入口。
// 一次调用替代 N 次独立查询，消除 N+1。limit=0 表示不限数量。
func (r *DeviceRepo) BatchIPsByDeviceIDs(_ context.Context, ids []string, limit int) (map[string][]*domain.IPv4Addr, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string][]*domain.IPv4Addr, len(ids))
	for _, id := range ids {
		ips := r.ips[id]
		if limit > 0 && len(ips) > limit {
			ips = ips[:limit]
		}
		result[id] = ips
	}
	return result, nil
}

// BatchConnectionsByDeviceIDs 是 DataLoader 的批量连接查询入口。limit=0 表示不限数量。
func (r *DeviceRepo) BatchConnectionsByDeviceIDs(_ context.Context, ids []string, limit int) (map[string][]*domain.DeviceLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string][]*domain.DeviceLink, len(ids))
	for _, id := range ids {
		links := r.links[id]
		if limit > 0 && len(links) > limit {
			links = links[:limit]
		}
		result[id] = links
	}
	return result, nil
}
