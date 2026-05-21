package neo4j_repo

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/domain"
)

// ── 内存存根（开发/测试用，待接入 Neo4j 后替换） ───────────────────────────

// DeviceRepo 设备的内存存根，实现与 Neo4jDeviceRepo 相同的接口。
// 生产环境使用 Neo4jDeviceRepo，两者接口一致，可直接切换。
type DeviceRepo struct {
	mu      sync.RWMutex
	devices map[string]*domain.Device
	ips     map[string][]*domain.IPv4Addr
	links   map[string][]*domain.DeviceLink
}

func NewDeviceRepo() *DeviceRepo {
	r := &DeviceRepo{
		devices: make(map[string]*domain.Device),
		ips:     make(map[string][]*domain.IPv4Addr),
		links:   make(map[string][]*domain.DeviceLink),
	}
	d1 := &domain.Device{ID: "dev-1", Name: "core-switch", Type: "switch", MIP: "10.0.0.1"}
	d2 := &domain.Device{ID: "dev-2", Name: "router-a", Type: "router", MIP: "10.0.0.2"}
	d3 := &domain.Device{ID: "dev-3", Name: "firewall", Type: "firewall", MIP: "10.0.0.3"}
	for _, d := range []*domain.Device{d1, d2, d3} {
		r.devices[d.ID] = d
		r.ips[d.ID] = nil
		r.links[d.ID] = nil
	}
	r.links["dev-1"] = []*domain.DeviceLink{
		{Target: d2, Relation: &domain.GraphRelation{ID: "rel-1", FromID: "dev-1", ToID: "dev-2", Type: "CONNECTED_TO"}},
		{Target: d3, Relation: &domain.GraphRelation{ID: "rel-2", FromID: "dev-1", ToID: "dev-3", Type: "CONNECTED_TO"}},
	}
	return r
}

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
	d := &domain.Device{ID: uuid.New().String(), Name: name, Type: deviceType, MIP: mip}
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

// ── Neo4j 实现（生产用） ───────────────────────────────────────────────────

// Neo4jDeviceRepo 是 DeviceRepo 的 Neo4j 生产实现。
// 使用时在 repository.New() 中替换掉 DeviceRepo。
type Neo4jDeviceRepo struct {
	Driver neo4j.Driver
	DBName string
}

func NewNeo4jDeviceRepo(driver neo4j.Driver, dbName string) *Neo4jDeviceRepo {
	return &Neo4jDeviceRepo{Driver: driver, DBName: dbName}
}

func (r *Neo4jDeviceRepo) FindDeviceByID(ctx context.Context, id string) (*domain.Device, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.Driver,
		`MATCH (d:Device {id: $id}) RETURN d`,
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.DBName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, nil
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "d")
	return nodeToDevice(node), nil
}

func (r *Neo4jDeviceRepo) FindDevices(ctx context.Context, first int, after string) ([]*domain.Device, bool, string, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.Driver,
		`MATCH (d:Device) WHERE $after = "" OR d.id > $after RETURN d ORDER BY d.id LIMIT $limit`,
		map[string]any{"after": after, "limit": first + 1},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.DBName),
	)
	if err != nil {
		return nil, false, "", err
	}
	var devices []*domain.Device
	for _, record := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](record, "d")
		devices = append(devices, nodeToDevice(node))
	}
	hasNext := len(devices) > first
	if hasNext {
		devices = devices[:first]
	}
	var endCursor string
	if len(devices) > 0 {
		endCursor = devices[len(devices)-1].ID
	}
	return devices, hasNext, endCursor, nil
}

func (r *Neo4jDeviceRepo) BatchIPsByDeviceIDs(ctx context.Context, deviceIDs []string, limit int) (map[string][]*domain.IPv4Addr, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.Driver,
		`MATCH (d:Device)-[:HAS_IP]->(ip:IPAddress) WHERE d.id IN $deviceIDs RETURN d.id AS deviceID, collect(ip)[0..$limit] AS ips`,
		map[string]any{"deviceIDs": deviceIDs, "limit": limit},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.DBName),
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]*domain.IPv4Addr, len(deviceIDs))
	for _, id := range deviceIDs {
		out[id] = []*domain.IPv4Addr{}
	}
	for _, record := range result.Records {
		devID, _, _ := neo4j.GetRecordValue[string](record, "deviceID")
		values, _, _ := neo4j.GetRecordValue[[]any](record, "ips")
		for _, v := range values {
			out[devID] = append(out[devID], nodeToIPAddress(v.(neo4j.Node)))
		}
	}
	return out, nil
}

func (r *Neo4jDeviceRepo) BatchConnectionsByDeviceIDs(ctx context.Context, deviceIDs []string, limit int) (map[string][]*domain.DeviceLink, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.Driver,
		`MATCH (d:Device)-[rel:CONNECTED_TO]->(target:Device) WHERE d.id IN $deviceIDs RETURN d.id AS deviceID, collect({target: target, rel: rel})[0..$limit] AS links`,
		map[string]any{"deviceIDs": deviceIDs, "limit": limit},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.DBName),
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]*domain.DeviceLink, len(deviceIDs))
	for _, id := range deviceIDs {
		out[id] = []*domain.DeviceLink{}
	}
	for _, record := range result.Records {
		deviceID, _, _ := neo4j.GetRecordValue[string](record, "deviceID")
		values, _, _ := neo4j.GetRecordValue[[]any](record, "links")
		for _, v := range values {
			item := v.(map[string]any)
			rel := item["rel"].(neo4j.Relationship)
			out[deviceID] = append(out[deviceID], &domain.DeviceLink{
				Target: nodeToDevice(item["target"].(neo4j.Node)),
				Relation: &domain.GraphRelation{
					ID:    rel.ElementId,
					Type:  rel.Type,
					Props: rel.Props,
				},
			})
		}
	}
	return out, nil
}

func nodeToDevice(node neo4j.Node) *domain.Device {
	get := func(key string) string {
		if v, ok := node.Props[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	return &domain.Device{
		ID:          get("id"),
		Name:        get("name"),
		Type:        get("type"),
		MIP:         get("mip"),
		LoginUser:   get("login_user"),
		LoginMethod: get("login_method"),
		LoginPasswd: get("login_passwd"),
	}
}

func nodeToIPAddress(node neo4j.Node) *domain.IPv4Addr {
	get := func(key string) string {
		if v, ok := node.Props[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	getUint64 := func(key string) uint64 {
		if v, ok := node.Props[key]; ok {
			if n, ok := v.(int64); ok {
				return uint64(n)
			}
		}
		return 0
	}
	return &domain.IPv4Addr{
		ID:        get("id"),
		IP:        get("address"),
		Mask:      get("mask"),
		Cidr:      get("cidr"),
		StartAddr: getUint64("start_addr"),
		EndAddr:   getUint64("end_addr"),
	}
}
