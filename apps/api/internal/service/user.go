package service

import (
	"context"
	"fmt"

	"matrix/api/domain"
)

// syncedUserRepo 定义用户管理查询所需的图层能力（接口定义在使用方）。
type syncedUserRepo interface {
	SearchSyncedUsers(ctx context.Context, search, cursor string, limit int) ([]*domain.SyncedUser, bool, string, error)
}

// UserService 提供已同步飞书用户的查询能力，供用户管理使用。
type UserService struct {
	graphRepo syncedUserRepo
}

func NewUserService(graph syncedUserRepo) *UserService {
	return &UserService{graphRepo: graph}
}

// SyncedUserList 用户列表查询结果，含游标分页信息。
type SyncedUserList struct {
	Users     []*domain.SyncedUser `json:"users"`
	HasNext   bool                 `json:"has_next"`
	EndCursor string               `json:"end_cursor"`
}

const (
	defaultUserPageSize = 20
	maxUserPageSize     = 100
)

// ListSynced 按搜索词和游标分页查询已同步用户。limit 越界时归一到合理范围。
func (s *UserService) ListSynced(ctx context.Context, search, cursor string, limit int) (*SyncedUserList, error) {
	if limit <= 0 {
		limit = defaultUserPageSize
	}
	if limit > maxUserPageSize {
		limit = maxUserPageSize
	}
	users, hasNext, endCursor, err := s.graphRepo.SearchSyncedUsers(ctx, search, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("user.ListSynced: %w", err)
	}
	if users == nil {
		users = []*domain.SyncedUser{}
	}
	return &SyncedUserList{Users: users, HasNext: hasNext, EndCursor: endCursor}, nil
}
