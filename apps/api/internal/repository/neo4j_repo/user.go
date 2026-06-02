package neo4j_repo

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"matrix/api/domain"
	"matrix/api/internal/infra/lark"
)

// UserGraphRepo 图中 User 节点的内存存根，生产环境替换为 Neo4j 实现。
type UserGraphRepo struct {
	mu    sync.RWMutex
	users map[string]*domain.User // feishuID → User
}

func NewUserGraphRepo() *UserGraphRepo {
	return &UserGraphRepo{users: make(map[string]*domain.User)}
}

// FindByFeishuID 按飞书 ID 查找图中的 User 节点。nil 表示图里还没有该节点。
func (r *UserGraphRepo) FindByFeishuID(_ context.Context, feishuID string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.users[feishuID], nil
}

func (r *UserGraphRepo) GetUser(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user %s not found", id)
}

func (r *UserGraphRepo) ListUsers(_ context.Context, first int, _ string) ([]*domain.User, bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		all = append(all, u)
	}
	if len(all) <= first {
		return all, false, "", nil
	}
	return all[:first], true, all[first-1].ID, nil
}

func (r *UserGraphRepo) CreateUser(_ context.Context, name, feishuID string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := &domain.User{ID: uuid.New().String(), Name: name, FeishuID: feishuID}
	r.users[feishuID] = u
	return u, nil
}

// GetSyncedUserDetail 内存存根：返回 nil（仅本地无 Neo4j 时使用）。
func (r *UserGraphRepo) GetSyncedUserDetail(_ context.Context, _ string) (*domain.UserDetail, error) {
	return nil, nil
}

// GetLeaders 内存存根：返回空列表。
func (r *UserGraphRepo) GetLeaders(_ context.Context, _ string, _ int) ([]*domain.SyncedUser, error) {
	return nil, nil
}

// GetManagedDepartments 内存存根：返回空列表。
func (r *UserGraphRepo) GetManagedDepartments(_ context.Context, _ string) ([]*domain.DepartmentRef, error) {
	return nil, nil
}

// SearchUsersByKeyword 内存存根：返回空列表。
func (r *UserGraphRepo) SearchUsersByKeyword(_ context.Context, _ string, _ int) ([]*domain.SyncedUser, error) {
	return nil, nil
}

// BulkUpsertSyncedUsers 内存存根：noop。
func (r *UserGraphRepo) BulkUpsertSyncedUsers(_ context.Context, _ []lark.User) error {
	return nil
}

// ListAllFeishuIDs 内存存根：从内存 map 取所有 feishu_id。
func (r *UserGraphRepo) ListAllFeishuIDs(_ context.Context) (map[string]struct{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set := make(map[string]struct{}, len(r.users))
	for k := range r.users {
		set[k] = struct{}{}
	}
	return set, nil
}

// DeleteByFeishuIDs 内存存根：从内存 map 删除并返回删除数。
func (r *UserGraphRepo) DeleteByFeishuIDs(_ context.Context, ids []string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleted := 0
	for _, id := range ids {
		if _, ok := r.users[id]; ok {
			delete(r.users, id)
			deleted++
		}
	}
	return deleted, nil
}

// SearchSyncedUsers 内存存根降级实现：仅按 name 匹配，email/mobile 等字段为空。
// 本地无 Neo4j 时使用，生产走 Neo4jUserGraphRepo 返回完整字段。
func (r *UserGraphRepo) SearchSyncedUsers(_ context.Context, search, cursor string, limit int) ([]*domain.SyncedUser, bool, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*domain.SyncedUser
	for _, u := range r.users {
		if search != "" && !strings.Contains(strings.ToLower(u.Name), strings.ToLower(search)) {
			continue
		}
		if cursor != "" && u.FeishuID <= cursor {
			continue
		}
		all = append(all, &domain.SyncedUser{ID: u.ID, Name: u.Name, FeishuID: u.FeishuID})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].FeishuID < all[j].FeishuID })
	hasNext := len(all) > limit
	if hasNext {
		all = all[:limit]
	}
	var endCursor string
	if len(all) > 0 {
		endCursor = all[len(all)-1].FeishuID
	}
	return all, hasNext, endCursor, nil
}

// ── Neo4j 实现（生产用） ───────────────────────────────────────────────────

// BulkUpsertSyncedUsers 批量 MERGE User 节点。一次 cypher 写入一批用户，相比逐条调用降低 ~100x 网络往返。
// 以 feishu_id 为业务唯一键；id 派生自 open_id，保证同一用户 id 稳定。空切片直接返回 nil。
func (r *Neo4jUserGraphRepo) BulkUpsertSyncedUsers(ctx context.Context, batch []lark.User) error {
	if len(batch) == 0 {
		return nil
	}
	now := time.Now().UTC()
	rows := make([]map[string]any, 0, len(batch))
	for _, u := range batch {
		if u.OpenID == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"feishu_id":      u.OpenID,
			"id":             uuid.NewSHA1(uuid.Nil, []byte(u.OpenID)).String(),
			"name":           u.Name,
			"user_id":        u.UserID,
			"email":          u.Email,
			"mobile":         u.Mobile,
			"status":         int64(u.Status),
			"department_ids": u.DepartmentIDs,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $rows AS row
		 MERGE (u:User {feishu_id: row.feishu_id})
		 ON CREATE SET u.id = row.id, u.created_at = $now
		 SET u.name = row.name,
		     u.user_id = row.user_id,
		     u.email = row.email,
		     u.mobile = row.mobile,
		     u.status = row.status,
		     u.department_ids = row.department_ids,
		     u.updated_at = $now`,
		map[string]any{"rows": rows, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	return err
}

// ListAllFeishuIDs 返回所有 User 的 feishu_id 集合，供 sync diff 计算 deleted。
func (r *Neo4jUserGraphRepo) ListAllFeishuIDs(ctx context.Context) (map[string]struct{}, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User) RETURN u.feishu_id AS id`,
		nil,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, fmt.Errorf("user repo: list all feishu ids: %w", err)
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

// DeleteByFeishuIDs 批量 DETACH DELETE 用户节点，返回实际删除数。
func (r *Neo4jUserGraphRepo) DeleteByFeishuIDs(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`UNWIND $ids AS id
		 MATCH (u:User {feishu_id: id})
		 DETACH DELETE u
		 RETURN count(u) AS deleted`,
		map[string]any{"ids": ids},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return 0, fmt.Errorf("user repo: delete by feishu ids: %w", err)
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


// Neo4jUserGraphRepo 图中 User 节点的 Neo4j 实现，接口与内存存根一致，可直接切换。
// User 节点以 feishu_id（飞书 open_id）为业务唯一键，与扫码登录、employees.external_id 对齐。
type Neo4jUserGraphRepo struct {
	driver neo4j.Driver
	dbName string
}

func NewNeo4jUserGraphRepo(driver neo4j.Driver, dbName string) *Neo4jUserGraphRepo {
	return &Neo4jUserGraphRepo{driver: driver, dbName: dbName}
}

func (r *Neo4jUserGraphRepo) FindByFeishuID(ctx context.Context, feishuID string) (*domain.User, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {feishu_id: $feishuID}) RETURN u`,
		map[string]any{"feishuID": feishuID},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, nil // 图里还没有该节点
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "u")
	return nodeToUser(node), nil
}

func (r *Neo4jUserGraphRepo) GetUser(ctx context.Context, id string) (*domain.User, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {id: $id}) RETURN u`,
		map[string]any{"id": id},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	if len(result.Records) == 0 {
		return nil, fmt.Errorf("user %s not found", id)
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "u")
	return nodeToUser(node), nil
}

func (r *Neo4jUserGraphRepo) ListUsers(ctx context.Context, first int, after string) ([]*domain.User, bool, string, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User)
		 WHERE ($after = "" OR u.feishu_id > $after)
		 RETURN u ORDER BY u.feishu_id LIMIT $limit`,
		map[string]any{"after": after, "limit": first + 1},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, false, "", err
	}
	var users []*domain.User
	for _, record := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](record, "u")
		users = append(users, nodeToUser(node))
	}
	hasNext := len(users) > first
	if hasNext {
		users = users[:first]
	}
	var endCursor string
	if len(users) > 0 {
		endCursor = users[len(users)-1].FeishuID
	}
	return users, hasNext, endCursor, nil
}

// CreateUser 以 feishu_id 为键 MERGE，幂等：同一 open_id 重复调用不会产生重复节点。
// id 由 open_id 派生（SHA1），保证同一飞书用户的节点 id 稳定。
func (r *Neo4jUserGraphRepo) CreateUser(ctx context.Context, name, feishuID string) (*domain.User, error) {
	id := uuid.NewSHA1(uuid.Nil, []byte(feishuID)).String()
	now := time.Now().UTC()
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MERGE (u:User {feishu_id: $feishuID})
		 ON CREATE SET u.id = $id, u.name = $name, u.created_at = $now, u.updated_at = $now
		 ON MATCH SET u.name = $name, u.updated_at = $now
		 RETURN u`,
		map[string]any{"feishuID": feishuID, "id": id, "name": name, "now": now},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, err
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "u")
	return nodeToUser(node), nil
}

// SearchSyncedUsers 按 name/email 模糊匹配（不区分大小写），以 feishu_id 游标分页。
// 通过 OPTIONAL MATCH MEMBER_OF 边一次性取到所属部门中文名，无需额外查询。
func (r *Neo4jUserGraphRepo) SearchSyncedUsers(ctx context.Context, search, cursor string, limit int) ([]*domain.SyncedUser, bool, string, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User)
		 WHERE ($search = "" OR toLower(u.name) CONTAINS toLower($search) OR toLower(u.email) CONTAINS toLower($search))
		   AND ($cursor = "" OR u.feishu_id > $cursor)
		 OPTIONAL MATCH (u)-[:MEMBER_OF]->(d:Department)
		 WITH u, collect(d.name) AS dept_names
		 RETURN u, dept_names ORDER BY u.feishu_id LIMIT $limit`,
		map[string]any{"search": search, "cursor": cursor, "limit": limit + 1},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(r.dbName),
	)
	if err != nil {
		return nil, false, "", err
	}
	var users []*domain.SyncedUser
	for _, record := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](record, "u")
		su := nodeToSyncedUser(node)
		// 从关系边收集部门中文名
		if namesVal, ok := record.Get("dept_names"); ok {
			if arr, ok := namesVal.([]any); ok {
				for _, n := range arr {
					if s, ok := n.(string); ok && s != "" {
						su.DepartmentNames = append(su.DepartmentNames, s)
					}
				}
			}
		}
		users = append(users, su)
	}
	hasNext := len(users) > limit
	if hasNext {
		users = users[:limit]
	}
	var endCursor string
	if len(users) > 0 {
		endCursor = users[len(users)-1].FeishuID
	}
	return users, hasNext, endCursor, nil
}

// GetSyncedUserDetail 按 open_id 取用户详情：基本字段 + 所属部门(每个带从顶级到当前的完整路径)。
// 用图一次查完：MEMBER_OF 拿所属部门，PARENT_OF* 拿每个部门的祖先链。
func (r *Neo4jUserGraphRepo) GetSyncedUserDetail(ctx context.Context, openID string) (*domain.UserDetail, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {feishu_id: $openID})
		 OPTIONAL MATCH (u)-[:MEMBER_OF]->(d:Department)
		 OPTIONAL MATCH path = (top:Department)-[:PARENT_OF*0..]->(d)
		 WHERE NOT (top)<-[:PARENT_OF]-(:Department)
		 WITH u, d, [n IN nodes(path) | {department_id: n.department_id, name: n.name}] AS chain
		 RETURN u,
		        collect(CASE WHEN d IS NULL THEN NULL ELSE {department_id: d.department_id, name: d.name, path: chain} END) AS departments`,
		map[string]any{"openID": openID},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("user repo: get detail: %w", err)
	}
	if len(result.Records) == 0 {
		return nil, nil
	}
	node, _, _ := neo4j.GetRecordValue[neo4j.Node](result.Records[0], "u")
	su := nodeToSyncedUser(node)

	detail := &domain.UserDetail{SyncedUser: su}
	if v, ok := result.Records[0].Get("departments"); ok {
		if arr, ok := v.([]any); ok {
			for _, e := range arr {
				m, ok := e.(map[string]any)
				if !ok {
					continue
				}
				ud := &domain.UserDepartment{}
				if s, ok := m["department_id"].(string); ok {
					ud.DepartmentID = s
				}
				if s, ok := m["name"].(string); ok {
					ud.Name = s
				}
				if pathArr, ok := m["path"].([]any); ok {
					for _, pe := range pathArr {
						pm, ok := pe.(map[string]any)
						if !ok {
							continue
						}
						ref := &domain.DepartmentRef{}
						if s, ok := pm["department_id"].(string); ok {
							ref.DepartmentID = s
						}
						if s, ok := pm["name"].(string); ok {
							ref.Name = s
						}
						ud.Path = append(ud.Path, ref)
					}
				}
				detail.Departments = append(detail.Departments, ud)
				// 同时填充 SyncedUser.DepartmentNames，保持与列表接口字段一致
				if ud.Name != "" {
					su.DepartmentNames = append(su.DepartmentNames, ud.Name)
				}
			}
		}
	}
	return detail, nil
}

// GetLeaders 上级领导：对用户所属的每个部门，沿 PARENT_OF 链从自身向上查找，
// 取链上首个 leader_user_id 不是本人 user_id 的部门，其 leader 即为该所属链的上级。
// 多部门可能合并出多名上级，按 user_id 去重，姓名排序。limit<=0 时默认 12。
// 通过一次 Cypher 完成：reverse(nodes(path)) 让数组从自身向上排列，head([... WHERE ...]) 取首个候选。
func (r *Neo4jUserGraphRepo) GetLeaders(ctx context.Context, openID string, limit int) ([]*domain.SyncedUser, error) {
	if limit <= 0 {
		limit = 12
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {feishu_id: $openID})-[:MEMBER_OF]->(d:Department)
		 MATCH path = (top:Department)-[:PARENT_OF*0..]->(d)
		 WHERE NOT (top)<-[:PARENT_OF]-(:Department)
		 WITH u, head([n IN reverse(nodes(path))
		               WHERE n.leader_user_id IS NOT NULL
		                 AND n.leader_user_id <> ""
		                 AND n.leader_user_id <> u.user_id]) AS leaderDept
		 WHERE leaderDept IS NOT NULL
		 MATCH (leader:User {user_id: leaderDept.leader_user_id})
		 RETURN DISTINCT leader ORDER BY leader.name LIMIT $limit`,
		map[string]any{"openID": openID, "limit": limit},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("user repo: get leaders: %w", err)
	}
	leaders := make([]*domain.SyncedUser, 0, len(result.Records))
	for _, rec := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "leader")
		leaders = append(leaders, nodeToSyncedUser(node))
	}
	return leaders, nil
}

// GetManagedDepartments 管理部门：所有 leader_user_id == 本人 user_id 的部门，按名称排序。
// 用户的 user_id 来自 User 节点；user_id 缺失时返回空（飞书租户内的用户标识缺失视为无法匹配）。
func (r *Neo4jUserGraphRepo) GetManagedDepartments(ctx context.Context, openID string) ([]*domain.DepartmentRef, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User {feishu_id: $openID})
		 WHERE u.user_id IS NOT NULL AND u.user_id <> ""
		 MATCH (d:Department {leader_user_id: u.user_id})
		 RETURN d.department_id AS department_id, d.name AS name ORDER BY d.name`,
		map[string]any{"openID": openID},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("user repo: get managed departments: %w", err)
	}
	refs := make([]*domain.DepartmentRef, 0, len(result.Records))
	for _, rec := range result.Records {
		ref := &domain.DepartmentRef{}
		if v, ok := rec.Get("department_id"); ok {
			if s, ok := v.(string); ok {
				ref.DepartmentID = s
			}
		}
		if v, ok := rec.Get("name"); ok {
			if s, ok := v.(string); ok {
				ref.Name = s
			}
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// SearchUsersByKeyword 按 name/email 模糊匹配，返回轻量列表（含部门名）。供联合搜索使用。
func (r *Neo4jUserGraphRepo) SearchUsersByKeyword(ctx context.Context, q string, limit int) ([]*domain.SyncedUser, error) {
	if limit <= 0 {
		limit = 20
	}
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User)
		 WHERE toLower(u.name) CONTAINS toLower($q) OR toLower(u.email) CONTAINS toLower($q)
		 OPTIONAL MATCH (u)-[:MEMBER_OF]->(d:Department)
		 WITH u, collect(d.name) AS dept_names
		 RETURN u, dept_names ORDER BY u.name LIMIT $limit`,
		map[string]any{"q": q, "limit": limit},
		neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(r.dbName))
	if err != nil {
		return nil, fmt.Errorf("user repo: search: %w", err)
	}
	users := make([]*domain.SyncedUser, 0, len(result.Records))
	for _, rec := range result.Records {
		node, _, _ := neo4j.GetRecordValue[neo4j.Node](rec, "u")
		su := nodeToSyncedUser(node)
		if v, ok := rec.Get("dept_names"); ok {
			if arr, ok := v.([]any); ok {
				for _, e := range arr {
					if s, ok := e.(string); ok && s != "" {
						su.DepartmentNames = append(su.DepartmentNames, s)
					}
				}
			}
		}
		users = append(users, su)
	}
	return users, nil
}

// nodeToUser 从 Neo4j 节点提取 domain.User 三字段，节点上的其余飞书属性（email/mobile 等）不映射。
func nodeToUser(node neo4j.Node) *domain.User {
	return &domain.User{
		ID:       nodeStr(node, "id"),
		Name:     nodeStr(node, "name"),
		FeishuID: nodeStr(node, "feishu_id"),
	}
}

// nodeToSyncedUser 从 Neo4j 节点提取完整飞书字段。
func nodeToSyncedUser(node neo4j.Node) *domain.SyncedUser {
	var deptIDs []string
	if v, ok := node.Props["department_ids"]; ok {
		if arr, ok := v.([]any); ok {
			for _, e := range arr {
				if s, ok := e.(string); ok {
					deptIDs = append(deptIDs, s)
				}
			}
		}
	}
	status := 0
	if v, ok := node.Props["status"]; ok {
		if n, ok := v.(int64); ok {
			status = int(n)
		}
	}
	return &domain.SyncedUser{
		ID:            nodeStr(node, "id"),
		Name:          nodeStr(node, "name"),
		FeishuID:      nodeStr(node, "feishu_id"),
		UserID:        nodeStr(node, "user_id"),
		Email:         nodeStr(node, "email"),
		Mobile:        nodeStr(node, "mobile"),
		Status:        status,
		DepartmentIDs: deptIDs,
	}
}

// nodeStr 从节点属性安全取字符串，缺失或类型不符返回空串。
func nodeStr(node neo4j.Node, key string) string {
	if v, ok := node.Props[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
