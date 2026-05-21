package neo4j_repo

import (
	"context"
	"sync"

	"matrix/api/domain"
)

// UserGraphRepo 图中 User 节点的内存存根，生产环境替换为 Neo4j 实现。
type UserGraphRepo struct {
	mu    sync.RWMutex
	users map[string]*domain.User // feishuID → User
}

func NewUserGraphRepo() *UserGraphRepo {
	r := &UserGraphRepo{users: make(map[string]*domain.User)}
	r.users["ou_feishu_001"] = &domain.User{ID: "usr-1", Name: "张三", FeishuID: "ou_feishu_001"}
	r.users["ou_feishu_002"] = &domain.User{ID: "usr-2", Name: "李四", FeishuID: "ou_feishu_002"}
	return r
}

// FindByFeishuID 按飞书 ID 查找图中的 User 节点。nil 表示图里还没有该节点。
func (r *UserGraphRepo) FindByFeishuID(_ context.Context, feishuID string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.users[feishuID], nil
}
