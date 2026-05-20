package repository

import (
	"context"
	"fmt"
	"matrix/api/domain"
	"sync"

	"github.com/google/uuid"
)

// DeviceRepo 是设备的内存仓库，演示原子化数据访问。
// 生产环境替换为 SQL/NoSQL 实现，接口不变。
type DeviceRepo struct {
	mu      sync.RWMutex
	devices map[string]*domain.Device
	ips     map[string][]*domain.IPv4Addr // deviceID -> IP 列表
}

func newDeviceRepo() *DeviceRepo {
	return &DeviceRepo{
		devices: make(map[string]*domain.Device),
		ips:     make(map[string][]*domain.IPv4Addr),
	}
}

func (r *DeviceRepo) ListDevices(_ context.Context) ([]*domain.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Device, 0, len(r.devices))
	for _, d := range r.devices {
		out = append(out, d)
	}
	return out, nil
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
	return true, nil
}

// BatchListIPsByDeviceIDs 是 DataLoader 的批量查询入口。
// 一次调用用 IN 查询替代 N 次独立查询，是消除 N+1 的关键。
// 返回 map 而非 slice，让 DataLoader 按 key 分发结果。
func (r *DeviceRepo) BatchListIPsByDeviceIDs(_ context.Context, ids []string) (map[string][]*domain.IPv4Addr, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string][]*domain.IPv4Addr, len(ids))
	for _, id := range ids {
		result[id] = r.ips[id] // 没有数据时返回 nil slice，不报错
	}
	return result, nil
}
