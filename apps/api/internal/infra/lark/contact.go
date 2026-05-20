// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"fmt"
	"time"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	"go.uber.org/zap"
)

// ContactFetcher 通讯录全量拉取能力。
type ContactFetcher interface {
	// FetchAllDepartments 从根部门递归获取所有部门。
	FetchAllDepartments(ctx context.Context) ([]Department, error)

	// FetchDepartmentUsers 获取指定部门的直属用户列表（分页全量）。
	FetchDepartmentUsers(ctx context.Context, departmentID string) ([]User, error)

	// FetchAllUsers 递归获取所有部门下的所有用户（去重）。
	FetchAllUsers(ctx context.Context) ([]User, error)

	// GetUser 获取单个用户详情。
	GetUser(ctx context.Context, userID string) (*User, error)

	// GetDepartment 获取单个部门详情。
	GetDepartment(ctx context.Context, departmentID string) (*Department, error)
}

// larkRateLimit 飞书 API 调用间隔，避免触发 code=99991400 频率限制。
// 飞书企业版通讯录 API 限速约 100 QPS；150ms 约 6.7 QPS，安全边际充足。
const larkRateLimit = 150 * time.Millisecond

// contactFetcher 实现 ContactFetcher。
type contactFetcher struct {
	client *Client
	logger *zap.Logger
}

// compile-time interface check
var _ ContactFetcher = (*contactFetcher)(nil)

// NewContactFetcher 创建通讯录拉取器。
func NewContactFetcher(client *Client, logger *zap.Logger) ContactFetcher {
	return &contactFetcher{client: client, logger: logger}
}

// FetchAllDepartments 从根部门递归获取所有部门（BFS）。
// 每次 API 调用之间等待 larkRateLimit，避免触发飞书频率限制。
// 单个部门子节点拉取失败时 warn+continue，不中断整体遍历。
func (f *contactFetcher) FetchAllDepartments(ctx context.Context) ([]Department, error) {
	var result []Department
	queue := []string{"0"} // 从根部门 "0" 开始

	for len(queue) > 0 {
		deptID := queue[0]
		queue = queue[1:]

		var pageToken string
		fetchFailed := false
		for {
			time.Sleep(larkRateLimit)

			req := larkcontact.NewChildrenDepartmentReqBuilder().
				DepartmentId(deptID).
				UserIdType("user_id").
				DepartmentIdType("department_id").
				PageSize(50).
				PageToken(pageToken).
				Build()

			resp, err := f.client.Contact.Department.Children(ctx, req)
			if err != nil {
				f.logger.Warn("contact: fetch departments under dept, skipping",
					zap.String("dept_id", deptID), zap.Error(err))
				fetchFailed = true
				break
			}
			if !resp.Success() {
				f.logger.Warn("contact: fetch departments under dept, skipping",
					zap.String("dept_id", deptID),
					zap.Int("code", resp.Code), zap.String("msg", resp.Msg))
				fetchFailed = true
				break
			}
			if resp.Data == nil {
				break
			}

			for _, dept := range resp.Data.Items {
				d := convertDepartment(dept)
				result = append(result, d)
				if dept.DepartmentId != nil {
					queue = append(queue, *dept.DepartmentId)
				}
			}

			if resp.Data.HasMore == nil || !*resp.Data.HasMore {
				break
			}
			if resp.Data.PageToken != nil {
				pageToken = *resp.Data.PageToken
			}
		}
		_ = fetchFailed // 已通过 warn 记录，继续遍历队列中其余部门
	}

	return result, nil
}

// FetchDepartmentUsers 获取指定部门的直属用户列表（分页全量）。
func (f *contactFetcher) FetchDepartmentUsers(ctx context.Context, departmentID string) ([]User, error) {
	var result []User
	var pageToken string

	for {
		time.Sleep(larkRateLimit)

		req := larkcontact.NewFindByDepartmentUserReqBuilder().
			UserIdType("user_id").
			DepartmentIdType("department_id").
			DepartmentId(departmentID).
			PageSize(50).
			PageToken(pageToken).
			Build()

		resp, err := f.client.Contact.User.FindByDepartment(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("contact: fetch users in dept %s: %w", departmentID, err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("contact: fetch users in dept %s: code=%d, msg=%s", departmentID, resp.Code, resp.Msg)
		}
		if resp.Data == nil {
			break
		}

		for _, u := range resp.Data.Items {
			result = append(result, convertUser(u))
		}

		if resp.Data.HasMore == nil || !*resp.Data.HasMore {
			break
		}
		if resp.Data.PageToken != nil {
			pageToken = *resp.Data.PageToken
		}
	}

	return result, nil
}

// FetchAllUsers 遍历所有部门，逐部门拉取直属用户并去重，返回全公司用户列表。
// SDK v3.5.3 的 FindByDepartmentUserReqBuilder 不支持递归模式（fetch_user_type=2），
// 因此通过先 FetchAllDepartments 再逐部门请求来保证全量获取。
func (f *contactFetcher) FetchAllUsers(ctx context.Context) ([]User, error) {
	depts, err := f.FetchAllDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("contact: fetch all users, list depts: %w", err)
	}

	seen := make(map[string]struct{}, len(depts)*10)
	var result []User

	for _, dept := range depts {
		deptUsers, err := f.FetchDepartmentUsers(ctx, dept.DepartmentID)
		if err != nil {
			f.logger.Warn("contact: fetch all users, skip dept",
				zap.String("dept_id", dept.DepartmentID), zap.Error(err))
			continue
		}
		for _, u := range deptUsers {
			if _, dup := seen[u.UserID]; !dup {
				seen[u.UserID] = struct{}{}
				result = append(result, u)
			}
		}
	}

	f.logger.Info("contact: fetch all users completed", zap.Int("total", len(result)))
	return result, nil
}

// GetUser 获取单个用户详情。
func (f *contactFetcher) GetUser(ctx context.Context, userID string) (*User, error) {
	req := larkcontact.NewGetUserReqBuilder().
		UserId(userID).
		UserIdType("user_id").
		DepartmentIdType("department_id").
		Build()

	resp, err := f.client.Contact.User.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("contact: get user %s: %w", userID, err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("contact: get user %s: code=%d, msg=%s", userID, resp.Code, resp.Msg)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("contact: get user %s: empty response", userID)
	}

	u := convertUser(resp.Data.User)
	return &u, nil
}

// GetDepartment 获取单个部门详情。
func (f *contactFetcher) GetDepartment(ctx context.Context, departmentID string) (*Department, error) {
	req := larkcontact.NewGetDepartmentReqBuilder().
		DepartmentId(departmentID).
		UserIdType("user_id").
		DepartmentIdType("department_id").
		Build()

	resp, err := f.client.Contact.Department.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("contact: get department %s: %w", departmentID, err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("contact: get department %s: code=%d, msg=%s", departmentID, resp.Code, resp.Msg)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("contact: get department %s: empty response", departmentID)
	}

	d := convertDepartment(resp.Data.Department)
	return &d, nil
}

// convertUser 将 SDK User 转换为 User。
func convertUser(u *larkcontact.User) User {
	if u == nil {
		return User{}
	}
	user := User{
		DepartmentIDs: u.DepartmentIds,
	}
	if u.UserId != nil {
		user.UserID = *u.UserId
	}
	if u.Name != nil {
		user.Name = *u.Name
	}
	if u.Email != nil {
		user.Email = *u.Email
	}
	if u.Mobile != nil {
		user.Mobile = *u.Mobile
	}
	user.Status = userStatusToInt(u.Status)
	return user
}

// convertDepartment 将 SDK Department 转换为 Department。
func convertDepartment(d *larkcontact.Department) Department {
	if d == nil {
		return Department{}
	}
	dept := Department{}
	if d.DepartmentId != nil {
		dept.DepartmentID = *d.DepartmentId
	}
	if d.Name != nil {
		dept.Name = *d.Name
	}
	if d.ParentDepartmentId != nil {
		dept.ParentID = *d.ParentDepartmentId
	}
	if d.LeaderUserId != nil {
		dept.LeaderUserID = *d.LeaderUserId
	}
	if d.MemberCount != nil {
		dept.MemberCount = *d.MemberCount
	}
	dept.Status = deptStatusToInt(d.Status)
	return dept
}

// userStatusToInt 将 SDK UserStatus 转换为 domain 整数状态。
// 1=已激活, 2=已禁用, 4=未激活, 0=未知。
func userStatusToInt(s *larkcontact.UserStatus) int {
	if s == nil {
		return 0
	}
	if s.IsActivated != nil && *s.IsActivated {
		return 1
	}
	if s.IsFrozen != nil && *s.IsFrozen {
		return 2
	}
	if s.IsUnjoin != nil && *s.IsUnjoin {
		return 4
	}
	return 0
}

// deptStatusToInt 将 SDK DepartmentStatus 转换为 domain 整数状态。
// 0=正常, 1=已删除。
func deptStatusToInt(s *larkcontact.DepartmentStatus) int {
	if s == nil {
		return 0
	}
	if s.IsDeleted != nil && *s.IsDeleted {
		return 1
	}
	return 0
}
