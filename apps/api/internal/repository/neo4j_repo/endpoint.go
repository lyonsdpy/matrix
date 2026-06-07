package neo4j_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/domain"
	"matrix/api/internal/infra/lark"
)

// Neo4jEndpointRepo Endpoint(终端) 节点的 Neo4j 仓库。
// 业务唯一键：feishu_device_id（= 飞书 device_record_id），与 User.feishu_id 同模式。
// 与 User 的关联通过两条独立边维持：
//   - (Endpoint)-[:CURRENT_LOGIN]->(User)  ← 飞书 current_user_id，当前登录用户
//   - (Endpoint)-[:LATEST_LOGIN]->(User)   ← 飞书 latest_user_id，最近一次登录用户
// 语义都是"登录"而非"归属"——归属由后续资产管理系统单独维护。
type Neo4jEndpointRepo struct {
	driver neo4j.Driver
	dbName string
}

func NewNeo4jEndpointRepo(driver neo4j.Driver, dbName string) *Neo4jEndpointRepo {
	return &Neo4jEndpointRepo{driver: driver, dbName: dbName}
}

// BulkUpsert 批量 MERGE Endpoint 节点。id 派生自 feishu_device_id，保证幂等。
// 写入飞书所有可靠字段（硬件标识、合规、MDM 等），status 统一置 ACTIVE（资产管理后续接管）。
func (r *Neo4jEndpointRepo) BulkUpsert(ctx context.Context, batch []lark.Device) error {
	if len(batch) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([]map[string]any, 0, len(batch))
	for _, d := range batch {
		if d.DeviceID == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"feishu_device_id":   d.DeviceID,
			"id":                 uuid.NewSHA1(uuid.Nil, []byte(d.DeviceID)).String(),
			"name":               d.DeviceName,
			"type":               string(terminalTypeToEndpoint(d.Platform)),
			"os":                 string(deviceSystemToOS(d.OSCode)),
			"status":             string(domain.EndpointStatusActive),
			"platform_code":      d.Platform,
			"serial_number":      d.SerialNumber,
			"disk_serial_number": d.DiskSerialNumber,
			"board_uuid":         d.BoardUUID,
			"mac_address":        d.MACAddress,
			"model":              d.Model,
			"os_code":            d.OSCode,
			"version":            d.Version,
			"ownership":          d.Ownership,
			"trust_level":        d.TrustLevel,
			"certification":      d.Certification,
			"is_managed":         d.IsManaged,
			"mdm_device_id":      d.MDMDeviceID,
			"mdm_provider":       d.MDMProvider,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 MERGE (e:Endpoint {feishu_device_id: row.feishu_device_id})
		 ON CREATE SET e.id = row.id, e.created_at = $now
		 SET e.name = row.name,
		     e.type = row.type,
		     e.os = row.os,
		     e.status = row.status,
		     e.platform_code = row.platform_code,
		     e.serial_number = row.serial_number,
		     e.disk_serial_number = row.disk_serial_number,
		     e.board_uuid = row.board_uuid,
		     e.mac_address = row.mac_address,
		     e.model = row.model,
		     e.os_code = row.os_code,
		     e.version = row.version,
		     e.ownership = row.ownership,
		     e.trust_level = row.trust_level,
		     e.certification = row.certification,
		     e.is_managed = row.is_managed,
		     e.mdm_device_id = row.mdm_device_id,
		     e.mdm_provider = row.mdm_provider,
		     e.updated_at = $now`,
		map[string]any{"rows": rows, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// BulkLinkLogin 批量重建 Endpoint→User 的两种登录边。
// 每个设备先清掉旧的 CURRENT_LOGIN / LATEST_LOGIN（含历史遗留的 LAST_LOGGED_IN_BY），
// 再根据 row.current_user_id / row.latest_user_id 分别建新边。
// 找不到对应 User 节点的 row 静默跳过（飞书 device 引用但通讯录未同步的边缘用户）。
func (r *Neo4jEndpointRepo) BulkLinkLogin(ctx context.Context, batch []lark.Device) error {
	if len(batch) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(batch))
	for _, d := range batch {
		if d.DeviceID == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"feishu_device_id": d.DeviceID,
			"current_user_id":  d.CurrentUserID, // 可空，cypher 内过滤
			"latest_user_id":   d.LatestUserID,  // 可空，cypher 内过滤
		})
	}
	if len(rows) == 0 {
		return nil
	}
	// 三步独立 cypher：清旧边 → 建 CURRENT_LOGIN → 建 LATEST_LOGIN。
	// 改成三次而非一次的原因：嵌套 FOREACH + pattern comprehension 在不同 Neo4j 版本上
	// 兼容性差（用户线上版本报 SyntaxError）。CALL subquery 又限制更严，干脆拆成三条
	// 直白的 UNWIND，每条都很容易看懂；总耗时仍是 O(批大小)。
	if _, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 MATCH (e:Endpoint {feishu_device_id: row.feishu_device_id})
		 OPTIONAL MATCH (e)-[old:CURRENT_LOGIN|LATEST_LOGIN|LAST_LOGGED_IN_BY]->(:User)
		 DELETE old`,
		map[string]any{"rows": rows},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	); err != nil {
		return fmt.Errorf("endpoint repo: clear login edges: %w", err)
	}
	if _, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 WITH row WHERE row.current_user_id IS NOT NULL AND row.current_user_id <> ""
		 MATCH (e:Endpoint {feishu_device_id: row.feishu_device_id})
		 MATCH (cu:User {user_id: row.current_user_id})
		 MERGE (e)-[:CURRENT_LOGIN]->(cu)`,
		map[string]any{"rows": rows},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	); err != nil {
		return fmt.Errorf("endpoint repo: link current login: %w", err)
	}
	if _, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 WITH row WHERE row.latest_user_id IS NOT NULL AND row.latest_user_id <> ""
		 MATCH (e:Endpoint {feishu_device_id: row.feishu_device_id})
		 MATCH (lu:User {user_id: row.latest_user_id})
		 MERGE (e)-[:LATEST_LOGIN]->(lu)`,
		map[string]any{"rows": rows},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	); err != nil {
		return fmt.Errorf("endpoint repo: link latest login: %w", err)
	}
	return nil
}

// ListAllFeishuDeviceIDs 返回所有 Endpoint 的 feishu_device_id 集合，供 sync diff。
func (r *Neo4jEndpointRepo) ListAllFeishuDeviceIDs(ctx context.Context) (map[string]struct{}, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (e:Endpoint) RETURN e.feishu_device_id AS id`,
		nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, fmt.Errorf("endpoint repo: list all feishu ids: %w", err)
	}
	set := make(map[string]struct{}, len(result.Records))
	for _, rec := range result.Records {
		if v, ok := rec.Get("id"); ok {
			if s, ok := v.(string); ok && s != "" {
				set[s] = struct{}{}
			}
		}
	}
	return set, nil
}

// DeleteByFeishuDeviceIDs 批量 DETACH DELETE Endpoint，返回删除数。
func (r *Neo4jEndpointRepo) DeleteByFeishuDeviceIDs(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $ids AS id
		 MATCH (e:Endpoint {feishu_device_id: id})
		 DETACH DELETE e
		 RETURN count(e) AS deleted`,
		map[string]any{"ids": ids},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return 0, fmt.Errorf("endpoint repo: delete by ids: %w", err)
	}
	if len(result.Records) == 0 {
		return 0, nil
	}
	if v, ok := result.Records[0].Get("deleted"); ok {
		if n, ok := v.(int64); ok {
			return int(n), nil
		}
	}
	return 0, nil
}

// Search 终端管理列表查询：按设备名/序列号模糊 + 关联用户名/邮箱筛选 + 类型/系统精确筛选 + 分页（offset/limit）。
// 排序按 feishu_device_id 升序，保证分页稳定。limit/offset 由 service 层归一化。
// typeFilter / osFilter 为精确匹配（值是 EndpointType / EndpointOS 常量字符串）；空字符串视为不筛。
// userQ 非空时要求设备至少有一个关联 User(current 或 latest) 命中关键字。
// 拆成两条 cypher（count + page）便于前端做"第 N 页"跳转：单条 collect 会把全部命中节点一次性
// 驻留在内存里，对几千级别的列表也没必要，count 走全表扫描的代价远低于把所有节点收集到列表里。
func (r *Neo4jEndpointRepo) Search(ctx context.Context, q, userQ, typeFilter, osFilter string, offset, limit int) ([]*domain.SyncedEndpoint, int64, error) {
	params := map[string]any{
		"q": q, "userQ": userQ,
		"typeFilter": typeFilter, "osFilter": osFilter,
	}
	countRes, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (e:Endpoint)
		 WHERE ($q = "" OR toLower(e.name) CONTAINS toLower($q)
		                 OR toLower(coalesce(e.serial_number, "")) CONTAINS toLower($q))
		   AND ($typeFilter = "" OR e.type = $typeFilter)
		   AND ($osFilter   = "" OR e.os   = $osFilter)
		 OPTIONAL MATCH (e)-[:CURRENT_LOGIN]->(cu:User)
		 OPTIONAL MATCH (e)-[:LATEST_LOGIN]->(lu:User)
		 WITH e, cu, lu
		 WHERE $userQ = ""
		    OR toLower(coalesce(cu.name, ""))  CONTAINS toLower($userQ)
		    OR toLower(coalesce(cu.email, "")) CONTAINS toLower($userQ)
		    OR toLower(coalesce(lu.name, ""))  CONTAINS toLower($userQ)
		    OR toLower(coalesce(lu.email, "")) CONTAINS toLower($userQ)
		 RETURN count(DISTINCT e) AS total`,
		params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("endpoint repo: count: %w", err)
	}
	var total int64
	if len(countRes.Records) > 0 {
		if v, ok := countRes.Records[0].Get("total"); ok {
			if n, ok := v.(int64); ok {
				total = n
			}
		}
	}
	if total == 0 || int64(offset) >= total {
		return []*domain.SyncedEndpoint{}, total, nil
	}

	pageParams := map[string]any{
		"q": q, "userQ": userQ,
		"typeFilter": typeFilter, "osFilter": osFilter,
		"offset": offset, "limit": limit,
	}
	pageRes, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (e:Endpoint)
		 WHERE ($q = "" OR toLower(e.name) CONTAINS toLower($q)
		                 OR toLower(coalesce(e.serial_number, "")) CONTAINS toLower($q))
		   AND ($typeFilter = "" OR e.type = $typeFilter)
		   AND ($osFilter   = "" OR e.os   = $osFilter)
		 OPTIONAL MATCH (e)-[:CURRENT_LOGIN]->(cu:User)
		 OPTIONAL MATCH (e)-[:LATEST_LOGIN]->(lu:User)
		 WITH e, cu, lu
		 WHERE $userQ = ""
		    OR toLower(coalesce(cu.name, ""))  CONTAINS toLower($userQ)
		    OR toLower(coalesce(cu.email, "")) CONTAINS toLower($userQ)
		    OR toLower(coalesce(lu.name, ""))  CONTAINS toLower($userQ)
		    OR toLower(coalesce(lu.email, "")) CONTAINS toLower($userQ)
		 RETURN e, cu, lu ORDER BY e.feishu_device_id SKIP $offset LIMIT $limit`,
		pageParams,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("endpoint repo: search: %w", err)
	}
	endpoints := make([]*domain.SyncedEndpoint, 0, len(pageRes.Records))
	for _, rec := range pageRes.Records {
		eNode, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "e")
		ep := nodeToSyncedEndpoint(eNode)
		if cuVal, ok := rec.Get("cu"); ok && cuVal != nil {
			if cuNode, ok := cuVal.(neo4j.Node); ok {
				ep.CurrentUser = nodeToSyncedUser(cuNode)
			}
		}
		if luVal, ok := rec.Get("lu"); ok && luVal != nil {
			if luNode, ok := luVal.(neo4j.Node); ok {
				ep.LatestUser = nodeToSyncedUser(luNode)
			}
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, total, nil
}

// GetDetail 终端详情：按 id 取节点 + 两个关联 User。id 为节点 id（SHA1(feishu_device_id)）。
func (r *Neo4jEndpointRepo) GetDetail(ctx context.Context, id string) (*domain.SyncedEndpoint, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (e:Endpoint {id: $id})
		 OPTIONAL MATCH (e)-[:CURRENT_LOGIN]->(cu:User)
		 OPTIONAL MATCH (e)-[:LATEST_LOGIN]->(lu:User)
		 RETURN e, cu, lu`,
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, fmt.Errorf("endpoint repo: get detail: %w", err)
	}
	if len(result.Records) == 0 {
		return nil, nil
	}
	rec := result.Records[0]
	eNode, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "e")
	ep := nodeToSyncedEndpoint(eNode)
	if cuVal, ok := rec.Get("cu"); ok && cuVal != nil {
		if cuNode, ok := cuVal.(neo4j.Node); ok {
			ep.CurrentUser = nodeToSyncedUser(cuNode)
		}
	}
	if luVal, ok := rec.Get("lu"); ok && luVal != nil {
		if luNode, ok := luVal.(neo4j.Node); ok {
			ep.LatestUser = nodeToSyncedUser(luNode)
		}
	}
	return ep, nil
}

// nodeToSyncedEndpoint 从 Neo4j Endpoint 节点提取 domain.SyncedEndpoint（不含关联 User）。
func nodeToSyncedEndpoint(node neo4j.Node) *domain.SyncedEndpoint {
	managed := false
	if v, ok := node.Props["is_managed"]; ok {
		if b, ok := v.(bool); ok {
			managed = b
		}
	}
	return &domain.SyncedEndpoint{
		ID:               nodeStr(node, "id"),
		FeishuDeviceID:   nodeStr(node, "feishu_device_id"),
		Name:             nodeStr(node, "name"),
		Type:             domain.EndpointType(nodeStr(node, "type")),
		OS:               domain.EndpointOS(nodeStr(node, "os")),
		Status:           domain.EndpointStatus(nodeStr(node, "status")),
		PlatformCode:     nodeStr(node, "platform_code"),
		SerialNumber:     nodeStr(node, "serial_number"),
		DiskSerialNumber: nodeStr(node, "disk_serial_number"),
		BoardUUID:        nodeStr(node, "board_uuid"),
		MACAddress:       nodeStr(node, "mac_address"),
		Model:            nodeStr(node, "model"),
		OSCode:           nodeStr(node, "os_code"),
		Version:          nodeStr(node, "version"),
		Ownership:        nodeStr(node, "ownership"),
		TrustLevel:       nodeStr(node, "trust_level"),
		Certification:    nodeStr(node, "certification"),
		IsManaged:        managed,
		MDMDeviceID:      nodeStr(node, "mdm_device_id"),
		MDMProvider:      nodeStr(node, "mdm_provider"),
	}
}

// terminalTypeToEndpoint 飞书 device_terminal_type 编号 → 物理形态（domain.EndpointType）。
// 飞书 SDK 真实定义：0=未知 / 1=移动端 / 2=桌面端，本模块对齐保留三态。
func terminalTypeToEndpoint(code string) domain.EndpointType {
	switch code {
	case "1":
		return domain.EndpointTypeMobile
	case "2":
		return domain.EndpointTypeDesktop
	default:
		return domain.EndpointTypeUnknown
	}
}

// deviceSystemToOS 飞书 device_system 编号 → 操作系统（domain.EndpointOS）。
// 飞书 SDK 真实定义：1=Windows / 2=macOS / 3=Linux / 4=Android / 5=iOS / 6=OpenHarmony。
// 注意：iOS 与 Android 的编号与一般直觉相反，以 SDK 为准。
func deviceSystemToOS(code string) domain.EndpointOS {
	switch code {
	case "1":
		return domain.EndpointOSWindows
	case "2":
		return domain.EndpointOSMacOS
	case "3":
		return domain.EndpointOSLinux
	case "4":
		return domain.EndpointOSAndroid
	case "5":
		return domain.EndpointOSIOS
	case "6":
		return domain.EndpointOSHarmonyOS
	default:
		return domain.EndpointOSOther
	}
}

// ── Stub（无 Neo4j 时降级） ───────────────────────────────────────────────────

type StubEndpointRepo struct{}

func NewStubEndpointRepo() *StubEndpointRepo { return &StubEndpointRepo{} }

func (StubEndpointRepo) BulkUpsert(context.Context, []lark.Device) error    { return nil }
func (StubEndpointRepo) BulkLinkLogin(context.Context, []lark.Device) error { return nil }
func (StubEndpointRepo) ListAllFeishuDeviceIDs(context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (StubEndpointRepo) DeleteByFeishuDeviceIDs(context.Context, []string) (int, error) {
	return 0, nil
}
func (StubEndpointRepo) Search(context.Context, string, string, string, string, int, int) ([]*domain.SyncedEndpoint, int64, error) {
	return []*domain.SyncedEndpoint{}, 0, nil
}
func (StubEndpointRepo) GetDetail(context.Context, string) (*domain.SyncedEndpoint, error) {
	return nil, nil
}
