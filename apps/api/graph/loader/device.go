package loader

import (
	"context"
	"matrix/api/domain"

	"github.com/vikstrous/dataloadgen"
)

// DeviceBatchProvider 是 device DataLoader 依赖的批量查询接口。
// DeviceRepo（内存）和 Neo4jRepo（图数据库）均可实现它。
type DeviceBatchProvider interface {
	BatchIPsByDeviceIDs(ctx context.Context, ids []string, limit int) (map[string][]*domain.IPv4Addr, error)
	BatchConnectionsByDeviceIDs(ctx context.Context, ids []string, limit int) (map[string][]*domain.DeviceLink, error)
}

// DeviceIPsKey 是 IpsByDeviceID loader 的查询 key。
// 携带 Limit 是因为不同查询场景需要的 IP 数量不同，DataLoader 会取同批次中最大的 Limit。
type DeviceIPsKey struct {
	DeviceID string
	Limit    int
}

// DeviceLinksKey 是 DevLinksByDeviceID loader 的查询 key。
type DeviceLinksKey struct {
	DeviceID string
	Limit    int
}

// newDeviceIPsLoader 创建按设备批量加载 IP 的 DataLoader。
func newDeviceIPsLoader(repo DeviceBatchProvider) *dataloadgen.Loader[DeviceIPsKey, []*domain.IPv4Addr] {
	return dataloadgen.NewLoader(
		func(ctx context.Context, keys []DeviceIPsKey) ([][]*domain.IPv4Addr, []error) {
			ids, limit := collectDeviceIPKeys(keys)
			m, err := repo.BatchIPsByDeviceIDs(ctx, ids, limit)
			return alignDeviceIPs(keys, m), sameErr(len(keys), err)
		},
	)
}

// newDeviceLinksLoader 创建按设备批量加载连接关系的 DataLoader。
func newDeviceLinksLoader(repo DeviceBatchProvider) *dataloadgen.Loader[DeviceLinksKey, []*domain.DeviceLink] {
	return dataloadgen.NewLoader(
		func(ctx context.Context, keys []DeviceLinksKey) ([][]*domain.DeviceLink, []error) {
			ids, limit := collectDeviceLinkKeys(keys)
			m, err := repo.BatchConnectionsByDeviceIDs(ctx, ids, limit)
			return alignDeviceLinks(keys, m), sameErr(len(keys), err)
		},
	)
}

// collectDeviceIPKeys 从一批 key 里去重提取 deviceID 列表，并取最大 Limit。
func collectDeviceIPKeys(keys []DeviceIPsKey) ([]string, int) {
	ids := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	limit := 20
	for _, key := range keys {
		if key.Limit > limit {
			limit = key.Limit
		}
		if _, ok := seen[key.DeviceID]; ok {
			continue
		}
		seen[key.DeviceID] = struct{}{}
		ids = append(ids, key.DeviceID)
	}
	return ids, limit
}

func collectDeviceLinkKeys(keys []DeviceLinksKey) ([]string, int) {
	ids := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	limit := 20
	for _, key := range keys {
		if key.Limit > limit {
			limit = key.Limit
		}
		if _, ok := seen[key.DeviceID]; ok {
			continue
		}
		seen[key.DeviceID] = struct{}{}
		ids = append(ids, key.DeviceID)
	}
	return ids, limit
}

// alignDeviceIPs 按原始 key 顺序重排批量查询结果。
// DataLoader 要求返回结果的顺序和 keys 顺序严格对应。
func alignDeviceIPs(keys []DeviceIPsKey, m map[string][]*domain.IPv4Addr) [][]*domain.IPv4Addr {
	out := make([][]*domain.IPv4Addr, len(keys))
	for i, key := range keys {
		out[i] = m[key.DeviceID]
	}
	return out
}

func alignDeviceLinks(keys []DeviceLinksKey, m map[string][]*domain.DeviceLink) [][]*domain.DeviceLink {
	out := make([][]*domain.DeviceLink, len(keys))
	for i, key := range keys {
		out[i] = m[key.DeviceID]
	}
	return out
}

// sameErr 把单个 error 展开成长度为 n 的 error slice，供 DataLoader 格式使用。
func sameErr(n int, err error) []error {
	errs := make([]error, n)
	if err != nil {
		for i := range errs {
			errs[i] = err
		}
	}
	return errs
}
