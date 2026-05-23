package neo4j_repo

import (
	"context"
	"fmt"
	"sync"

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
	id := fmt.Sprintf("user-%d", len(r.users)+1)
	u := &domain.User{ID: id, Name: name, FeishuID: feishuID}
	r.users[feishuID] = u
	return u, nil
}
