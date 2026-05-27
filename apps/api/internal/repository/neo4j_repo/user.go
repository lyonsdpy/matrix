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
func (r *Neo4jUserGraphRepo) SearchSyncedUsers(ctx context.Context, search, cursor string, limit int) ([]*domain.SyncedUser, bool, string, error) {
	result, err := neo4j.ExecuteQuery(ctx, r.driver,
		`MATCH (u:User)
		 WHERE ($search = "" OR toLower(u.name) CONTAINS toLower($search) OR toLower(u.email) CONTAINS toLower($search))
		   AND ($cursor = "" OR u.feishu_id > $cursor)
		 RETURN u ORDER BY u.feishu_id LIMIT $limit`,
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
		users = append(users, nodeToSyncedUser(node))
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
