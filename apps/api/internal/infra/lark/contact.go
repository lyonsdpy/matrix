// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// ContactFetcher 通讯录全量拉取能力。
type ContactFetcher interface {
	// FetchAllDepartments 从根部门递归获取所有部门。
	FetchAllDepartments(ctx context.Context) ([]Department, error)

	// FetchDepartmentUsers 获取指定部门的直属用户列表（分页全量）。
	FetchDepartmentUsers(ctx context.Context, departmentID string) ([]User, error)

	// FetchUsersInDepartments 并发逐部门拉取直属用户并去重，返回这些部门下的全部用户。
	// 由调用方传入部门 ID 列表（通常来自 FetchAllDepartments），避免内部重复遍历部门。
	// opts.Workers 控制并发度（默认 8），opts.OnProgress 每完成一个部门触发一次回调（done/total）。
	FetchUsersInDepartments(ctx context.Context, deptIDs []string, opts FetchUsersOptions) ([]User, error)

	// GetUser 获取单个用户详情。
	GetUser(ctx context.Context, userID string) (*User, error)

	// GetDepartment 获取单个部门详情。
	GetDepartment(ctx context.Context, departmentID string) (*Department, error)
}

// larkRateLimit 飞书 API 调用间隔，避免触发 code=99991400 频率限制。
// 飞书企业版通讯录 API 限速约 100 QPS；并发场景下每个 worker 内仍保留 150ms 间隔，
// 8 worker × ~6.7 QPS ≈ 53 QPS，留出 ~50% 安全边际。
const larkRateLimit = 150 * time.Millisecond

// FetchUsersOptions 拉取用户阶段的并发与进度选项。零值表示用默认（Workers=8、无回调）。
type FetchUsersOptions struct {
	Workers    int                       // 并发 worker 数；<=0 取默认 8
	OnProgress func(done, total int)     // 完成一个部门触发一次；可为 nil
}

const defaultFetchUsersWorkers = 8

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

// FetchAllDepartments 用 fetch_child=true 对根部门 "0" 一次性递归拉取全部子孙部门（分页）。
// 飞书该参数返回所有层级子部门，请求数从逐层 BFS 的 O(部门数) 降到 O(部门数/页大小)，
// 大型集团（数千部门）由数千次请求降到数十次。每页之间等待 larkRateLimit 避免触发频率限制。
func (f *contactFetcher) FetchAllDepartments(ctx context.Context) ([]Department, error) {
	var result []Department
	var pageToken string

	for {
		time.Sleep(larkRateLimit)

		req := larkcontact.NewChildrenDepartmentReqBuilder().
			DepartmentId("0").
			UserIdType("user_id").
			DepartmentIdType("department_id").
			FetchChild(true). // 递归返回所有层级子孙部门，无需逐层遍历
			PageSize(50).
			PageToken(pageToken).
			Build()

		resp, err := f.client.Contact.Department.Children(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("contact: fetch all departments: %w", err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("contact: fetch all departments: code=%d, msg=%s", resp.Code, resp.Msg)
		}
		if resp.Data == nil {
			break
		}

		for _, dept := range resp.Data.Items {
			result = append(result, convertDepartment(dept))
		}

		if resp.Data.HasMore == nil || !*resp.Data.HasMore {
			break
		}
		if resp.Data.PageToken != nil {
			pageToken = *resp.Data.PageToken
		}
	}

	f.logger.Info("contact: all departments fetched", zap.Int("total", len(result)))
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

// FetchUsersInDepartments 并发逐部门拉取直属用户并按 user_id 去重。
// SDK v3.5.3 的 FindByDepartmentUserReqBuilder 不支持递归模式（fetch_user_type=2），
// 故仍需逐部门请求；通过 worker pool 把原本串行的 sleep 累加变成并发，整体耗时从
// 部门数 × 150ms 降到 (部门数 / Workers) × 150ms，本项目 2k+ 部门由 ~300s 降到 ~40s。
//
// 单部门内的分页仍然串行（飞书要求按 PageToken 顺序），仅跨部门并发。
func (f *contactFetcher) FetchUsersInDepartments(ctx context.Context, deptIDs []string, opts FetchUsersOptions) ([]User, error) {
	workers := opts.Workers
	if workers <= 0 {
		workers = defaultFetchUsersWorkers
	}
	total := len(deptIDs)

	var (
		mu     sync.Mutex
		seen   = make(map[string]struct{}, total*10)
		result []User
		done   atomic.Int64
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	for _, deptID := range deptIDs {
		deptID := deptID
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			users, err := f.FetchDepartmentUsers(ctx, deptID)
			if err != nil {
				// 单个部门拉失败不熔断整体（多年实践：偶发 99991400 重试也救不回，跳过更稳）
				f.logger.Warn("contact: fetch users in departments, skip dept",
					zap.String("dept_id", deptID), zap.Error(err))
				users = nil
			}
			if len(users) > 0 {
				mu.Lock()
				for _, u := range users {
					if _, dup := seen[u.UserID]; !dup {
						seen[u.UserID] = struct{}{}
						result = append(result, u)
					}
				}
				mu.Unlock()
			}
			completed := done.Add(1)
			if opts.OnProgress != nil {
				opts.OnProgress(int(completed), total)
			}
			if completed%200 == 0 {
				f.logger.Info("contact: users fetch progress",
					zap.Int64("departments_done", completed),
					zap.Int("departments_total", total))
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("contact: fetch users in departments: %w", err)
	}

	f.logger.Info("contact: fetch users in departments completed",
		zap.Int("departments_total", total),
		zap.Int("users_collected", len(result)))
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
	if u.OpenId != nil {
		user.OpenID = *u.OpenId
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

// userStatusToInt 将 SDK UserStatus 转换为在职状态整数（按优先级互斥判断）。
// 1=在职, 2=已冻结, 3=离职, 4=待入职, 0=未知。
// 注意：status 字段需应用具备 contact:user.employee:readonly 权限飞书才返回，
// 否则 s 为 nil，全部落到 0（未知）。
func userStatusToInt(s *larkcontact.UserStatus) int {
	if s == nil {
		return 0
	}
	// 离职优先：is_exited（主动退出）一段时间后会转为 is_resigned，二者都视为离职
	if (s.IsResigned != nil && *s.IsResigned) || (s.IsExited != nil && *s.IsExited) {
		return 3
	}
	if s.IsFrozen != nil && *s.IsFrozen {
		return 2
	}
	if s.IsUnjoin != nil && *s.IsUnjoin {
		return 4
	}
	if s.IsActivated != nil && *s.IsActivated {
		return 1
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
