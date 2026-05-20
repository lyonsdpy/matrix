package lark

import (
	"testing"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
)

func TestConvertUser(t *testing.T) {
	tests := []struct {
		name  string
		input *larkcontact.User
		want  User
	}{
		{
			name:  "nil user",
			input: nil,
			want:  User{},
		},
		{
			name: "all fields activated",
			input: &larkcontact.User{
				UserId:        ptrStr("user123"),
				Name:          ptrStr("张三"),
				Email:         ptrStr("zs@example.com"),
				Mobile:        ptrStr("13800138000"),
				Status:        &larkcontact.UserStatus{IsActivated: ptrBool(true)},
				DepartmentIds: []string{"dept1", "dept2"},
			},
			want: User{
				UserID:        "user123",
				Name:          "张三",
				Email:         "zs@example.com",
				Mobile:        "13800138000",
				Status:        1,
				DepartmentIDs: []string{"dept1", "dept2"},
			},
		},
		{
			name: "frozen user",
			input: &larkcontact.User{
				UserId: ptrStr("user456"),
				Status: &larkcontact.UserStatus{IsFrozen: ptrBool(true)},
			},
			want: User{
				UserID: "user456",
				Status: 2,
			},
		},
		{
			name: "unjoin user",
			input: &larkcontact.User{
				UserId: ptrStr("user789"),
				Status: &larkcontact.UserStatus{IsUnjoin: ptrBool(true)},
			},
			want: User{
				UserID: "user789",
				Status: 4,
			},
		},
		{
			name: "nil status",
			input: &larkcontact.User{
				UserId: ptrStr("user000"),
			},
			want: User{
				UserID: "user000",
				Status: 0,
			},
		},
		{
			name: "empty department ids",
			input: &larkcontact.User{
				UserId:        ptrStr("user001"),
				DepartmentIds: nil,
			},
			want: User{
				UserID: "user001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertUser(tt.input)
			if got.UserID != tt.want.UserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.want.UserID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Email != tt.want.Email {
				t.Errorf("Email = %q, want %q", got.Email, tt.want.Email)
			}
			if got.Mobile != tt.want.Mobile {
				t.Errorf("Mobile = %q, want %q", got.Mobile, tt.want.Mobile)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %d, want %d", got.Status, tt.want.Status)
			}
			if len(got.DepartmentIDs) != len(tt.want.DepartmentIDs) {
				t.Errorf("DepartmentIDs len = %d, want %d", len(got.DepartmentIDs), len(tt.want.DepartmentIDs))
			}
		})
	}
}

func TestConvertDepartment(t *testing.T) {
	tests := []struct {
		name  string
		input *larkcontact.Department
		want  Department
	}{
		{
			name:  "nil department",
			input: nil,
			want:  Department{},
		},
		{
			name: "all fields active",
			input: &larkcontact.Department{
				DepartmentId:       ptrStr("dept001"),
				Name:               ptrStr("技术部"),
				ParentDepartmentId: ptrStr("dept000"),
				LeaderUserId:       ptrStr("user_lead"),
				MemberCount:        ptrInt(10),
				Status:             &larkcontact.DepartmentStatus{IsDeleted: ptrBool(false)},
			},
			want: Department{
				DepartmentID: "dept001",
				Name:         "技术部",
				ParentID:     "dept000",
				LeaderUserID: "user_lead",
				MemberCount:  10,
				Status:       0,
			},
		},
		{
			name: "deleted department",
			input: &larkcontact.Department{
				DepartmentId: ptrStr("dept002"),
				Status:       &larkcontact.DepartmentStatus{IsDeleted: ptrBool(true)},
			},
			want: Department{
				DepartmentID: "dept002",
				Status:       1,
			},
		},
		{
			name: "nil status",
			input: &larkcontact.Department{
				DepartmentId: ptrStr("dept003"),
				Status:       nil,
			},
			want: Department{
				DepartmentID: "dept003",
				Status:       0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDepartment(tt.input)
			if got.DepartmentID != tt.want.DepartmentID {
				t.Errorf("DepartmentID = %q, want %q", got.DepartmentID, tt.want.DepartmentID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.ParentID != tt.want.ParentID {
				t.Errorf("ParentID = %q, want %q", got.ParentID, tt.want.ParentID)
			}
			if got.LeaderUserID != tt.want.LeaderUserID {
				t.Errorf("LeaderUserID = %q, want %q", got.LeaderUserID, tt.want.LeaderUserID)
			}
			if got.MemberCount != tt.want.MemberCount {
				t.Errorf("MemberCount = %d, want %d", got.MemberCount, tt.want.MemberCount)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %d, want %d", got.Status, tt.want.Status)
			}
		})
	}
}
