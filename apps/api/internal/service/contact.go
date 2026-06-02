package service

import (
	"context"
	"fmt"

	"matrix/api/domain"
)

// contactUserRepo 联合搜索的用户分支（接口定义在使用方）。
type contactUserRepo interface {
	SearchUsersByKeyword(ctx context.Context, q string, limit int) ([]*domain.SyncedUser, error)
}

// contactDeptRepo 联合搜索的部门分支。
type contactDeptRepo interface {
	SearchByName(ctx context.Context, q string, limit int) ([]*domain.DepartmentNode, error)
}

// ContactService 通讯录联合搜索：一次输入同时检索用户和部门，按类型分组返回。
// 用 goroutine 并发查 user 和 dept，两条 cypher 各自独立，加起来不超过单次延迟。
type ContactService struct {
	users contactUserRepo
	depts contactDeptRepo
}

func NewContactService(users contactUserRepo, depts contactDeptRepo) *ContactService {
	return &ContactService{users: users, depts: depts}
}

// SearchResult 联合搜索结果，按实体类型分组，前端可分别渲染 UserCard / DepartmentCard。
type SearchResult struct {
	Users       []*domain.SyncedUser     `json:"users"`
	Departments []*domain.DepartmentNode `json:"departments"`
}

const defaultSearchLimit = 20

// Search 并发查用户 + 部门，任一失败不影响另一分支返回。
func (s *ContactService) Search(ctx context.Context, q string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if q == "" {
		return &SearchResult{Users: []*domain.SyncedUser{}, Departments: []*domain.DepartmentNode{}}, nil
	}

	type userOut struct {
		v   []*domain.SyncedUser
		err error
	}
	type deptOut struct {
		v   []*domain.DepartmentNode
		err error
	}
	uc := make(chan userOut, 1)
	dc := make(chan deptOut, 1)

	go func() {
		v, err := s.users.SearchUsersByKeyword(ctx, q, limit)
		uc <- userOut{v: v, err: err}
	}()
	go func() {
		v, err := s.depts.SearchByName(ctx, q, limit)
		dc <- deptOut{v: v, err: err}
	}()

	u := <-uc
	d := <-dc

	// 任一失败时仍返回另一分支，整体 error 仅在两边都失败时上抛
	if u.err != nil && d.err != nil {
		return nil, fmt.Errorf("contact.Search: users=%v, depts=%v", u.err, d.err)
	}
	users := u.v
	if users == nil {
		users = []*domain.SyncedUser{}
	}
	depts := d.v
	if depts == nil {
		depts = []*domain.DepartmentNode{}
	}
	return &SearchResult{Users: users, Departments: depts}, nil
}
