package neo4j_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/domain"
	"matrix/api/internal/infra/lark"
)

// Neo4jDepartmentRepo Department 节点的 Neo4j 实现：写入 + 图特性查询。
// 与 Neo4jUserGraphRepo 风格一致（struct + driver/dbName），便于 service 层统一注入。
type Neo4jDepartmentRepo struct {
	driver neo4j.Driver
	dbName string
}

func NewNeo4jDepartmentRepo(driver neo4j.Driver, dbName string) *Neo4jDepartmentRepo {
	return &Neo4jDepartmentRepo{driver: driver, dbName: dbName}
}

// ── 写入（供 sync 使用） ──────────────────────────────────────────────────────

// BulkUpsert 批量 MERGE 部门节点。一次 cypher 写入一批，相比逐条调用降低 ~100x 网络往返。
// batch 长度建议 200~500；上层切片传入。空切片直接返回 nil。
func (r *Neo4jDepartmentRepo) BulkUpsert(ctx context.Context, batch []lark.Department) error {
	if len(batch) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([]map[string]any, 0, len(batch))
	for _, d := range batch {
		if d.DepartmentID == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"department_id":  d.DepartmentID,
			"name":           d.Name,
			"parent_id":      d.ParentID,
			"leader_user_id": d.LeaderUserID,
			"member_count":   int64(d.MemberCount),
		})
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 MERGE (dept:Department {department_id: row.department_id})
		 SET dept.name = row.name,
		     dept.parent_id = row.parent_id,
		     dept.leader_user_id = row.leader_user_id,
		     dept.member_count = row.member_count,
		     dept.updated_at = $now`,
		map[string]any{"rows": rows, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// BulkLinkParents 批量建立 PARENT_OF 关系。parentID 为 "0"/"" 视为顶级，跳过。
func (r *Neo4jDepartmentRepo) BulkLinkParents(ctx context.Context, batch []lark.Department) error {
	if len(batch) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(batch))
	for _, d := range batch {
		if d.DepartmentID == "" || d.ParentID == "" || d.ParentID == "0" {
			continue
		}
		rows = append(rows, map[string]any{
			"child":  d.DepartmentID,
			"parent": d.ParentID,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 MATCH (p:Department {department_id: row.parent})
		 MATCH (c:Department {department_id: row.child})
		 MERGE (p)-[:PARENT_OF]->(c)`,
		map[string]any{"rows": rows},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// BulkLinkMembers 批量重建 User→Department 的 MEMBER_OF 边。先按用户清空旧边，再按 deptIDs 全量重建，保证与飞书严格一致。
func (r *Neo4jDepartmentRepo) BulkLinkMembers(ctx context.Context, batch []lark.User) error {
	if len(batch) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(batch))
	for _, u := range batch {
		if u.OpenID == "" || len(u.DepartmentIDs) == 0 {
			continue
		}
		rows = append(rows, map[string]any{
			"feishu_id": u.OpenID,
			"dept_ids":  u.DepartmentIDs,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	// 注意：先 OPTIONAL MATCH 删边再 UNWIND 建边，需要在 WITH 段隔离，避免删完整张图的副作用
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 MATCH (u:User {feishu_id: row.feishu_id})
		 OPTIONAL MATCH (u)-[old:MEMBER_OF]->(:Department)
		 DELETE old
		 WITH u, row
		 UNWIND row.dept_ids AS deptID
		 MATCH (d:Department {department_id: deptID})
		 MERGE (u)-[:MEMBER_OF]->(d)`,
		map[string]any{"rows": rows},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// ListAllIDs 返回所有 Department 的 department_id 集合，供 sync diff 计算 deleted。
func (r *Neo4jDepartmentRepo) ListAllIDs(ctx context.Context) (map[string]struct{}, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Department) RETURN d.department_id AS id`,
		nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, fmt.Errorf("department repo: list all ids: %w", err)
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

// DeleteByIDs 批量 DETACH DELETE 部门节点。飞书已不存在的部门用于回收图。
func (r *Neo4jDepartmentRepo) DeleteByIDs(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $ids AS id
		 MATCH (d:Department {department_id: id})
		 DETACH DELETE d
		 RETURN count(d) AS deleted`,
		map[string]any{"ids": ids},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return 0, fmt.Errorf("department repo: delete by ids: %w", err)
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

// Upsert 以 department_id 为键 MERGE Department 节点，幂等写入。
func (r *Neo4jDepartmentRepo) Upsert(ctx context.Context, d lark.Department) error {
	now := time.Now().UTC()
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MERGE (dept:Department {department_id: $deptID})
		 SET dept.name = $name,
		     dept.parent_id = $parentID,
		     dept.leader_user_id = $leaderUID,
		     dept.member_count = $memberCount,
		     dept.updated_at = $now`,
		map[string]any{
			"deptID":      d.DepartmentID,
			"name":        d.Name,
			"parentID":    d.ParentID,
			"leaderUID":   d.LeaderUserID,
			"memberCount": int64(d.MemberCount),
			"now":         now,
		},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// LinkParent 建立 (parent:Department)-[:PARENT_OF]->(child:Department) 关系。
// parentID 为 "0" 或空时表示根部门，不建立关系（顶级部门以"无入边"标识）。
func (r *Neo4jDepartmentRepo) LinkParent(ctx context.Context, deptID, parentID string) error {
	if parentID == "" || parentID == "0" {
		return nil
	}
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (parent:Department {department_id: $parentID})
		 MATCH (child:Department {department_id: $deptID})
		 MERGE (parent)-[:PARENT_OF]->(child)`,
		map[string]any{"parentID": parentID, "deptID": deptID},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// LinkUser 为 User 节点与其所属 Department 节点建立 MEMBER_OF 关系。
// 先删除旧 MEMBER_OF 边再批量重建，保证与飞书数据严格一致。
func (r *Neo4jDepartmentRepo) LinkUser(ctx context.Context, userFeishuID string, deptIDs []string) error {
	if len(deptIDs) == 0 {
		return nil
	}
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {feishu_id: $feishuID})
		 OPTIONAL MATCH (u)-[r:MEMBER_OF]->(:Department)
		 DELETE r
		 WITH DISTINCT u
		 UNWIND $deptIDs AS deptID
		 MATCH (d:Department {department_id: deptID})
		 MERGE (u)-[:MEMBER_OF]->(d)`,
		map[string]any{"feishuID": userFeishuID, "deptIDs": deptIDs},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// ── 查询（供通讯录读路径使用） ──────────────────────────────────────────────

// ListChildren 返回直接子部门，带 has_children 标志（前端树懒加载用）。
// parentID 为 "" 或 "0" 时返回顶级部门（无 PARENT_OF 入边的节点）。
func (r *Neo4jDepartmentRepo) ListChildren(ctx context.Context, parentID string) ([]*domain.DepartmentNode, error) {
	var cypher string
	params := map[string]any{}
	if parentID == "" || parentID == "0" {
		// 顶级：无 PARENT_OF 入边
		cypher = `MATCH (c:Department)
			WHERE NOT (c)<-[:PARENT_OF]-(:Department)
			RETURN c, EXISTS { MATCH (c)-[:PARENT_OF]->(:Department) } AS hasChildren
			ORDER BY c.name`
	} else {
		cypher = `MATCH (p:Department {department_id: $parentID})-[:PARENT_OF]->(c:Department)
			RETURN c, EXISTS { MATCH (c)-[:PARENT_OF]->(:Department) } AS hasChildren
			ORDER BY c.name`
		params["parentID"] = parentID
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver, cypher, params,
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("department repo: list children: %w", err)
	}
	nodes := make([]*domain.DepartmentNode, 0, len(result.Records))
	for _, rec := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "c")
		hasCh := false
		if v, ok := rec.Get("hasChildren"); ok {
			if b, ok := v.(bool); ok {
				hasCh = b
			}
		}
		nodes = append(nodes, nodeToDeptNode(node, hasCh))
	}
	return nodes, nil
}

// GetDetail 一次性取部门详情：基本信息 + 路径 + 直接子部门 + 直属成员（前 limit 个）+ 递归人数。
// 充分利用图：路径走 PARENT_OF*，递归人数走 PARENT_OF*+MEMBER_OF。
func (r *Neo4jDepartmentRepo) GetDetail(ctx context.Context, deptID string, memberLimit int) (*domain.DepartmentDetail, error) {
	// 1) 部门基本信息（确认存在）
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Department {department_id: $id}) RETURN d`,
		map[string]any{"id": deptID},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("department repo: get detail: %w", err)
	}
	if len(result.Records) == 0 {
		return nil, nil
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "d")
	detail := &domain.DepartmentDetail{
		DepartmentID: nodeStr(node, "department_id"),
		Name:         nodeStr(node, "name"),
		ParentID:     nodeStr(node, "parent_id"),
		LeaderUserID: nodeStr(node, "leader_user_id"),
		MemberCount:  nodeIntField(node, "member_count"),
	}

	// 2) 路径（从顶级到当前）
	path, err := r.GetPath(ctx, deptID)
	if err != nil {
		return nil, err
	}
	detail.Path = path

	// 3) 直接子部门
	children, err := r.ListChildren(ctx, deptID)
	if err != nil {
		return nil, err
	}
	detail.Children = children

	// 4) 递归成员数（部门 + 所有子孙部门的去重成员）
	count, err := r.RecursiveMemberCount(ctx, deptID)
	if err != nil {
		return nil, err
	}
	detail.RecursiveMemberCount = count

	// 5) 直属成员前 N 个
	members, err := r.ListDirectMembers(ctx, deptID, memberLimit)
	if err != nil {
		return nil, err
	}
	detail.DirectMembers = members

	return detail, nil
}

// GetPath 返回从顶级部门到指定部门的路径（含自身），顶级以"无 PARENT_OF 入边"识别。
func (r *Neo4jDepartmentRepo) GetPath(ctx context.Context, deptID string) ([]*domain.DepartmentRef, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH path = (top:Department)-[:PARENT_OF*0..]->(d:Department {department_id: $id})
		 WHERE NOT (top)<-[:PARENT_OF]-(:Department)
		 RETURN [n IN nodes(path) | {department_id: n.department_id, name: n.name}] AS chain
		 LIMIT 1`,
		map[string]any{"id": deptID},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("department repo: get path: %w", err)
	}
	if len(result.Records) == 0 {
		return nil, nil
	}
	return parsePathChain(result.Records[0]), nil
}

// RecursiveMemberCount 部门 + 所有子孙部门下去重用户数。
func (r *Neo4jDepartmentRepo) RecursiveMemberCount(ctx context.Context, deptID string) (int, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Department {department_id: $id})-[:PARENT_OF*0..]->(:Department)<-[:MEMBER_OF]-(u:User)
		 RETURN count(DISTINCT u) AS total`,
		map[string]any{"id": deptID},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return 0, fmt.Errorf("department repo: recursive member count: %w", err)
	}
	if len(result.Records) == 0 {
		return 0, nil
	}
	if v, ok := result.Records[0].Get("total"); ok {
		if n, ok := v.(int64); ok {
			return int(n), nil
		}
	}
	return 0, nil
}

// ListDirectMembers 部门直属成员（不含子部门），按姓名排序，前 limit 个。
func (r *Neo4jDepartmentRepo) ListDirectMembers(ctx context.Context, deptID string, limit int) ([]*domain.SyncedUser, error) {
	if limit <= 0 {
		limit = 50
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Department {department_id: $id})<-[:MEMBER_OF]-(u:User)
		 RETURN u ORDER BY u.name LIMIT $limit`,
		map[string]any{"id": deptID, "limit": limit},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("department repo: list direct members: %w", err)
	}
	users := make([]*domain.SyncedUser, 0, len(result.Records))
	for _, rec := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "u")
		users = append(users, nodeToSyncedUser(node))
	}
	return users, nil
}

// SearchByName 按部门名模糊匹配（不区分大小写），返回轻量节点列表。
func (r *Neo4jDepartmentRepo) SearchByName(ctx context.Context, q string, limit int) ([]*domain.DepartmentNode, error) {
	if limit <= 0 {
		limit = 20
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (d:Department) WHERE toLower(d.name) CONTAINS toLower($q)
		 RETURN d, EXISTS { MATCH (d)-[:PARENT_OF]->(:Department) } AS hasChildren
		 ORDER BY d.name LIMIT $limit`,
		map[string]any{"q": q, "limit": limit},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("department repo: search: %w", err)
	}
	nodes := make([]*domain.DepartmentNode, 0, len(result.Records))
	for _, rec := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "d")
		hasCh := false
		if v, ok := rec.Get("hasChildren"); ok {
			if b, ok := v.(bool); ok {
				hasCh = b
			}
		}
		nodes = append(nodes, nodeToDeptNode(node, hasCh))
	}
	return nodes, nil
}

// ── 节点解析辅助 ──────────────────────────────────────────────────────────────

func nodeToDeptNode(node neo4j.Node, hasChildren bool) *domain.DepartmentNode {
	return &domain.DepartmentNode{
		DepartmentID: nodeStr(node, "department_id"),
		Name:         nodeStr(node, "name"),
		ParentID:     nodeStr(node, "parent_id"),
		MemberCount:  nodeIntField(node, "member_count"),
		HasChildren:  hasChildren,
	}
}

// nodeIntField 从节点属性安全取 int，缺失或类型不符返回 0。
func nodeIntField(node neo4j.Node, key string) int {
	if v, ok := node.Props[key]; ok {
		if n, ok := v.(int64); ok {
			return int(n)
		}
	}
	return 0
}

// ── 无 Neo4j 时的降级 stub（与 UserGraphRepo 风格一致） ──────────────────────

// StubDepartmentRepo 本地无 Neo4j 时使用，所有读方法返回空，写方法 no-op。
type StubDepartmentRepo struct{}

func NewStubDepartmentRepo() *StubDepartmentRepo { return &StubDepartmentRepo{} }

func (StubDepartmentRepo) Upsert(context.Context, lark.Department) error      { return nil }
func (StubDepartmentRepo) LinkParent(context.Context, string, string) error   { return nil }
func (StubDepartmentRepo) LinkUser(context.Context, string, []string) error   { return nil }
func (StubDepartmentRepo) ListChildren(context.Context, string) ([]*domain.DepartmentNode, error) {
	return nil, nil
}
func (StubDepartmentRepo) GetDetail(context.Context, string, int) (*domain.DepartmentDetail, error) {
	return nil, nil
}
func (StubDepartmentRepo) GetPath(context.Context, string) ([]*domain.DepartmentRef, error) {
	return nil, nil
}
func (StubDepartmentRepo) RecursiveMemberCount(context.Context, string) (int, error) {
	return 0, nil
}
func (StubDepartmentRepo) ListDirectMembers(context.Context, string, int) ([]*domain.SyncedUser, error) {
	return nil, nil
}
func (StubDepartmentRepo) SearchByName(context.Context, string, int) ([]*domain.DepartmentNode, error) {
	return nil, nil
}

// ── sync stub：no-op，本地无 Neo4j 不能真做同步 ──
func (StubDepartmentRepo) BulkUpsert(context.Context, []lark.Department) error      { return nil }
func (StubDepartmentRepo) BulkLinkParents(context.Context, []lark.Department) error { return nil }
func (StubDepartmentRepo) BulkLinkMembers(context.Context, []lark.User) error       { return nil }
func (StubDepartmentRepo) ListAllIDs(context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (StubDepartmentRepo) DeleteByIDs(context.Context, []string) (int, error) { return 0, nil }

// parsePathChain 从查询记录的 chain 字段提取 DepartmentRef 列表。
// chain 由 cypher [n IN nodes(path) | {...}] 构造，元素是 map[string]any。
func parsePathChain(rec *neo4j.Record) []*domain.DepartmentRef {
	v, ok := rec.Get("chain")
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	refs := make([]*domain.DepartmentRef, 0, len(arr))
	for _, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		ref := &domain.DepartmentRef{}
		if s, ok := m["department_id"].(string); ok {
			ref.DepartmentID = s
		}
		if s, ok := m["name"].(string); ok {
			ref.Name = s
		}
		refs = append(refs, ref)
	}
	return refs
}
