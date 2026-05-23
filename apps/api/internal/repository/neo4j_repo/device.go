package neo4j_repo

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

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
	return &DeviceRepo{
		devices: make(map[string]*domain.Device),
		ips:     make(map[string][]*domain.IPv4Addr),
		links:   make(map[string][]*domain.DeviceLink),
	}
}

func (r *DeviceRepo) ListDevices(_ context.Context, first int, after string) ([]*domain.Device, bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.devices))
	for id, d := range r.devices {
		if !d.IsDeleted() {
			ids = append(ids, id)
		}
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
	if !ok || d.IsDeleted() {
		return nil, fmt.Errorf("device %s not found", id)
	}
	return d, nil
}

func (r *DeviceRepo) CreateDevice(_ context.Context, name, deviceType, mip string) (*domain.Device, error) {
	now := time.Now()
	d := &domain.Device{
		ID:   uuid.New().String(),
		Name: name, Type: deviceType, MIP: mip,
		TemporalMeta: domain.TemporalMeta{
			Version:   1,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	r.mu.Lock()
	r.devices[d.ID] = d
	r.ips[d.ID] = nil
	r.links[d.ID] = nil
	r.mu.Unlock()
	return d, nil
}

func (r *DeviceRepo) UpdateDevice(_ context.Context, id string, name, deviceType, mip *string) (*domain.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok || d.IsDeleted() {
		return nil, fmt.Errorf("device %s not found", id)
	}
	if name != nil {
		d.Name = *name
	}
	if deviceType != nil {
		d.Type = *deviceType
	}
	if mip != nil {
		d.MIP = *mip
	}
	d.UpdatedAt = time.Now()
	d.Version++
	return d, nil
}

func (r *DeviceRepo) DeleteDevice(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[id]
	if !ok || d.IsDeleted() {
		return false, nil
	}
	// 软删除：仅标记 DeletedAt，保留历史记录
	now := time.Now()
	d.DeletedAt = &now
	d.UpdatedAt = now
	d.Version++
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

func (r *DeviceRepo) CreateConnection(_ context.Context, fromID, toID string) (*domain.DeviceLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	from, ok := r.devices[fromID]
	if !ok || from.IsDeleted() {
		return nil, fmt.Errorf("device %s not found", fromID)
	}
	to, ok := r.devices[toID]
	if !ok || to.IsDeleted() {
		return nil, fmt.Errorf("device %s not found", toID)
	}
	link := &domain.DeviceLink{
		Target: to,
		Relation: &domain.GraphRelation{
			ID:     uuid.New().String(),
			FromID: fromID,
			ToID:   toID,
			Type:   "CONNECTED_TO",
		},
	}
	r.links[fromID] = append(r.links[fromID], link)
	return link, nil
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

type Neo4jDeviceRepo struct {
	driver neo4j.Driver
	dbName string
}

func NewNeo4jDeviceRepo(driver neo4j.Driver, dbName string) *Neo4jDeviceRepo {
	return &Neo4jDeviceRepo{driver: driver, dbName: dbName}
}

func (r *Neo4jDeviceRepo) GetDevice(ctx context.Context, id string) (*domain.Device, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Device {id: $id}) WHERE d.deleted_at IS NULL RETURN d`,
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, fmt.Errorf("device %s not found", id)
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "d")
	return nodeToDevice(node), nil
}

func (r *Neo4jDeviceRepo) ListDevices(ctx context.Context, first int, after string) ([]*domain.Device, bool, string, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Device)
		 WHERE d.deleted_at IS NULL AND ($after = "" OR d.id > $after)
		 RETURN d ORDER BY d.id LIMIT $limit`,
		map[string]any{"after": after, "limit": first + 1},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
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

func (r *Neo4jDeviceRepo) CreateDevice(ctx context.Context, name, deviceType, mip string) (*domain.Device, error) {
	now := time.Now().UTC()
	id := uuid.New().String()
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`CREATE (d:Device {
			id: $id, name: $name, type: $type, mip: $mip,
			version: 1, create_by: '', update_by: '',
			created_at: $now, updated_at: $now
		}) RETURN d`,
		map[string]any{"id": id, "name": name, "type": deviceType, "mip": mip, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "d")
	return nodeToDevice(node), nil
}

func (r *Neo4jDeviceRepo) UpdateDevice(ctx context.Context, id string, name, deviceType, mip *string) (*domain.Device, error) {
	if name == nil && deviceType == nil && mip == nil {
		return r.GetDevice(ctx, id)
	}
	props := map[string]any{}
	if name != nil {
		props["name"] = *name
	}
	if deviceType != nil {
		props["type"] = *deviceType
	}
	if mip != nil {
		props["mip"] = *mip
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Device {id: $id}) WHERE d.deleted_at IS NULL
		 SET d += $props, d.updated_at = $now, d.version = d.version + 1
		 RETURN d`,
		map[string]any{"id": id, "props": props, "now": time.Now().UTC()},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, fmt.Errorf("device %s not found", id)
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "d")
	return nodeToDevice(node), nil
}

func (r *Neo4jDeviceRepo) DeleteDevice(ctx context.Context, id string) (bool, error) {
	now := time.Now().UTC()
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Device {id: $id}) WHERE d.deleted_at IS NULL
		 SET d.deleted_at = $now, d.updated_at = $now, d.version = d.version + 1
		 RETURN count(d) > 0 AS deleted`,
		map[string]any{"id": id, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return false, err
	}
	if len(result.Records) == 0 {
		return false, nil
	}
	deleted, _, _ := neo4j.GetRecordValue[bool](result.Records[0], "deleted")
	return deleted, nil
}

func (r *Neo4jDeviceRepo) BatchIPsByDeviceIDs(ctx context.Context, deviceIDs []string, limit int) (map[string][]*domain.IPv4Addr, error) {
	// limit<=0 表示不限数量，Cypher 的 [0..0] 会返回空切片，用大数替代
	cyLimit := limit
	if cyLimit <= 0 {
		cyLimit = 100000
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Device)-[:HAS_IP]->(ip:IPAddress) WHERE d.id IN $deviceIDs RETURN d.id AS deviceID, collect(ip)[0..$limit] AS ips`,
		map[string]any{"deviceIDs": deviceIDs, "limit": cyLimit},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
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

func (r *Neo4jDeviceRepo) CreateConnection(ctx context.Context, fromID, toID string) (*domain.DeviceLink, error) {
	relID := uuid.NewSHA1(uuid.Nil, []byte(fromID+"-"+toID)).String()
	now := time.Now().UTC()
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (a:Device {id: $fromID}), (b:Device {id: $toID})
		 WHERE a.deleted_at IS NULL AND b.deleted_at IS NULL
		 MERGE (a)-[r:CONNECTED_TO {id: $relID}]->(b)
		 ON CREATE SET r.created_at = $now
		 RETURN b, r`,
		map[string]any{"fromID": fromID, "toID": toID, "relID": relID, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, fmt.Errorf("device %s or %s not found", fromID, toID)
	}
	targetNode, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "b")
	rel, _, _ := neo4j.GetRecordValue[neo4j.Relationship](result.Records[0], "r")
	return &domain.DeviceLink{
		Target: nodeToDevice(targetNode),
		Relation: &domain.GraphRelation{
			ID:     rel.ElementId,
			FromID: fromID,
			ToID:   toID,
			Type:   rel.Type,
			Props:  rel.Props,
		},
	}, nil
}

func (r *Neo4jDeviceRepo) BatchConnectionsByDeviceIDs(ctx context.Context, deviceIDs []string, limit int) (map[string][]*domain.DeviceLink, error) {
	// limit<=0 表示不限数量，Cypher 的 [0..0] 会返回空切片，用大数替代
	cyLimit := limit
	if cyLimit <= 0 {
		cyLimit = 100000
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Device)-[rel:CONNECTED_TO]->(target:Device) WHERE d.id IN $deviceIDs RETURN d.id AS deviceID, collect({target: target, rel: rel})[0..$limit] AS links`,
		map[string]any{"deviceIDs": deviceIDs, "limit": cyLimit},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
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
			target := nodeToDevice(item["target"].(neo4j.Node))
			out[deviceID] = append(out[deviceID], &domain.DeviceLink{
				Target: target,
				Relation: &domain.GraphRelation{
					ID:     rel.ElementId,
					FromID: deviceID,
					ToID:   target.ID,
					Type:   rel.Type,
					Props:  rel.Props,
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
	getTime := func(key string) *time.Time {
		if v, ok := node.Props[key]; ok {
			switch t := v.(type) {
			case time.Time:
				return &t
			case neo4j.LocalDateTime:
				tt := t.Time()
				return &tt
			}
		}
		return nil
	}
	getInt := func(key string) int {
		if v, ok := node.Props[key]; ok {
			if n, ok := v.(int64); ok {
				return int(n)
			}
		}
		return 0
	}

	meta := domain.TemporalMeta{
		Version:  getInt("version"),
		CreateBy: get("create_by"),
		UpdateBy: get("update_by"),
	}
	if t := getTime("created_at"); t != nil {
		meta.CreatedAt = *t
	}
	if t := getTime("updated_at"); t != nil {
		meta.UpdatedAt = *t
	}
	meta.DeletedAt = getTime("deleted_at")
	meta.ValidFrom = getTime("valid_from")
	meta.ValidTo = getTime("valid_to")

	return &domain.Device{
		ID:           get("id"),
		Name:         get("name"),
		Type:         get("type"),
		MIP:          get("mip"),
		LoginUser:    get("login_user"),
		LoginMethod:  get("login_method"),
		LoginPasswd:  get("login_passwd"),
		TemporalMeta: meta,
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
