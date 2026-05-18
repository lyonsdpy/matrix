package lark

import (
	"context"
	"fmt"
	"testing"

	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larksecurity "github.com/larksuite/oapi-sdk-go/v3/service/security_and_compliance/v2"
	larkapplication "github.com/larksuite/oapi-sdk-go/v3/service/application/v6"
	"go.uber.org/zap"

	domain "matrix/api/domain/lark"
)

// --- Mock handlers ---

type mockContactHandler struct {
	lastUserCreated *domain.User
	lastUserDeleted *domain.User
	lastUserUpdated *domain.User
	lastDeptCreated *domain.Department
	lastDeptDeleted *domain.Department
	lastDeptUpdated *domain.Department
}

func (m *mockContactHandler) OnUserCreated(_ context.Context, u *domain.User) error {
	m.lastUserCreated = u
	return nil
}
func (m *mockContactHandler) OnUserDeleted(_ context.Context, u *domain.User) error {
	m.lastUserDeleted = u
	return nil
}
func (m *mockContactHandler) OnUserUpdated(_ context.Context, u *domain.User) error {
	m.lastUserUpdated = u
	return nil
}
func (m *mockContactHandler) OnDeptCreated(_ context.Context, d *domain.Department) error {
	m.lastDeptCreated = d
	return nil
}
func (m *mockContactHandler) OnDeptDeleted(_ context.Context, d *domain.Department) error {
	m.lastDeptDeleted = d
	return nil
}
func (m *mockContactHandler) OnDeptUpdated(_ context.Context, d *domain.Department) error {
	m.lastDeptUpdated = d
	return nil
}

type mockDeviceHandler struct {
	lastDevice *domain.Device
}

func (m *mockDeviceHandler) OnDeviceChanged(_ context.Context, d *domain.Device) error {
	m.lastDevice = d
	return nil
}

type mockMessageHandler struct {
	lastMessage *domain.Message
	lastUserID  string
	lastChatID  string
}

func (m *mockMessageHandler) OnMessageReceived(_ context.Context, msg *domain.Message) error {
	m.lastMessage = msg
	return nil
}
func (m *mockMessageHandler) OnBotChatEntered(_ context.Context, userID, chatID string) error {
	m.lastUserID = userID
	m.lastChatID = chatID
	return nil
}

// newTestDispatcher 创建用于测试的 EventDispatcher（不需要真实凭证）。
func newTestDispatcher(opts ...EventDispatcherOption) *EventDispatcher {
	return NewEventDispatcher("test_app_id", "test_app_secret", zap.NewNop(), opts...)
}

// --- 转换函数测试 ---

func TestEventDispatcherConvertUserEvent(t *testing.T) {
	tests := []struct {
		name  string
		input *larkcontact.UserEvent
		want  domain.User
	}{
		{
			name:  "nil input",
			input: nil,
			want:  domain.User{},
		},
		{
			name: "full user activated",
			input: &larkcontact.UserEvent{
				UserId:        ptrStr("uid123"),
				Name:          ptrStr("张三"),
				Email:         ptrStr("zs@example.com"),
				Mobile:        ptrStr("13800138000"),
				DepartmentIds: []string{"dept1", "dept2"},
				Status:        &larkcontact.UserStatus{IsActivated: ptrBool(true)},
			},
			want: domain.User{
				UserID:        "uid123",
				Name:          "张三",
				Email:         "zs@example.com",
				Mobile:        "13800138000",
				Status:        1,
				DepartmentIDs: []string{"dept1", "dept2"},
			},
		},
		{
			name: "frozen user",
			input: &larkcontact.UserEvent{
				UserId: ptrStr("uid456"),
				Status: &larkcontact.UserStatus{IsFrozen: ptrBool(true)},
			},
			want: domain.User{UserID: "uid456", Status: 2},
		},
		{
			name: "unjoin user",
			input: &larkcontact.UserEvent{
				UserId: ptrStr("uid789"),
				Status: &larkcontact.UserStatus{IsUnjoin: ptrBool(true)},
			},
			want: domain.User{UserID: "uid789", Status: 4},
		},
		{
			name: "nil status",
			input: &larkcontact.UserEvent{
				UserId: ptrStr("uid000"),
			},
			want: domain.User{UserID: "uid000", Status: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertUserEvent(tt.input)
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

func TestEventDispatcherConvertDepartmentEvent(t *testing.T) {
	tests := []struct {
		name  string
		input *larkcontact.DepartmentEvent
		want  domain.Department
	}{
		{
			name:  "nil input",
			input: nil,
			want:  domain.Department{},
		},
		{
			name: "all fields active",
			input: &larkcontact.DepartmentEvent{
				DepartmentId:       ptrStr("dept001"),
				Name:               ptrStr("技术部"),
				ParentDepartmentId: ptrStr("dept000"),
				LeaderUserId:       ptrStr("user_lead"),
				Status:             &larkcontact.DepartmentStatus{IsDeleted: ptrBool(false)},
			},
			want: domain.Department{
				DepartmentID: "dept001",
				Name:         "技术部",
				ParentID:     "dept000",
				LeaderUserID: "user_lead",
				Status:       0,
				// MemberCount: 事件体不含此字段，始终为 0
			},
		},
		{
			name: "deleted department",
			input: &larkcontact.DepartmentEvent{
				DepartmentId: ptrStr("dept002"),
				Status:       &larkcontact.DepartmentStatus{IsDeleted: ptrBool(true)},
			},
			want: domain.Department{DepartmentID: "dept002", Status: 1},
		},
		{
			name: "nil status",
			input: &larkcontact.DepartmentEvent{
				DepartmentId: ptrStr("dept003"),
			},
			want: domain.Department{DepartmentID: "dept003", Status: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDepartmentEvent(tt.input)
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
			if got.Status != tt.want.Status {
				t.Errorf("Status = %d, want %d", got.Status, tt.want.Status)
			}
		})
	}
}

func TestEventDispatcherConvertDeviceChangeEvent(t *testing.T) {
	tests := []struct {
		name  string
		input *larksecurity.DeviceChangeEvent
		want  domain.Device
	}{
		{
			name:  "nil input",
			input: nil,
			want:  domain.Device{},
		},
		{
			name: "all fields",
			input: &larksecurity.DeviceChangeEvent{
				DeviceRecordId:  ptrStr("dev001"),
				DeviceName:      ptrStr("MacBook Pro"),
				DeviceSystem:    ptrInt(2),
				CurrentUserId:   &larksecurity.UserId{UserId: ptrStr("user001")},
				DeviceOwnership: ptrInt(1),
				DeviceStatus:    ptrInt(3),
				SerialNumber:    ptrStr("SN001"),
			},
			want: domain.Device{
				DeviceID:     "dev001",
				DeviceName:   "MacBook Pro",
				Platform:     fmt.Sprintf("%d", 2),
				UserID:       "user001",
				Status:       fmt.Sprintf("%d", 1),
				TrustLevel:   fmt.Sprintf("%d", 3),
				SerialNumber: "SN001",
			},
		},
		{
			name: "nil current user id",
			input: &larksecurity.DeviceChangeEvent{
				DeviceRecordId: ptrStr("dev002"),
				CurrentUserId:  nil,
			},
			want: domain.Device{DeviceID: "dev002"},
		},
		{
			name: "current user id inner nil",
			input: &larksecurity.DeviceChangeEvent{
				DeviceRecordId: ptrStr("dev003"),
				CurrentUserId:  &larksecurity.UserId{UserId: nil},
			},
			want: domain.Device{DeviceID: "dev003"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertDeviceChangeEvent(tt.input)
			if got.DeviceID != tt.want.DeviceID {
				t.Errorf("DeviceID = %q, want %q", got.DeviceID, tt.want.DeviceID)
			}
			if got.DeviceName != tt.want.DeviceName {
				t.Errorf("DeviceName = %q, want %q", got.DeviceName, tt.want.DeviceName)
			}
			if got.Platform != tt.want.Platform {
				t.Errorf("Platform = %q, want %q", got.Platform, tt.want.Platform)
			}
			if got.UserID != tt.want.UserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.want.UserID)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.TrustLevel != tt.want.TrustLevel {
				t.Errorf("TrustLevel = %q, want %q", got.TrustLevel, tt.want.TrustLevel)
			}
			if got.SerialNumber != tt.want.SerialNumber {
				t.Errorf("SerialNumber = %q, want %q", got.SerialNumber, tt.want.SerialNumber)
			}
		})
	}
}

func TestEventDispatcherConvertEventMessage(t *testing.T) {
	tests := []struct {
		name   string
		msg    *larkim.EventMessage
		sender *larkim.EventSender
		want   domain.Message
	}{
		{
			name: "nil message",
			msg:  nil,
			want: domain.Message{},
		},
		{
			name: "full message with sender",
			msg: &larkim.EventMessage{
				MessageId:   ptrStr("msg001"),
				ChatId:      ptrStr("chat001"),
				ChatType:    ptrStr("p2p"),
				MessageType: ptrStr("text"),
				Content:     ptrStr(`{"text":"hello"}`),
				CreateTime:  ptrStr("1740000000000"),
			},
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{UserId: ptrStr("sender001")},
			},
			want: domain.Message{
				MessageID:  "msg001",
				ChatID:     "chat001",
				ChatType:   "p2p",
				MsgType:    "text",
				Content:    `{"text":"hello"}`,
				SenderID:   "sender001",
				CreateTime: "1740000000000",
			},
		},
		{
			name: "nil sender",
			msg: &larkim.EventMessage{
				MessageId: ptrStr("msg002"),
			},
			sender: nil,
			want:   domain.Message{MessageID: "msg002"},
		},
		{
			name: "sender with nil user id",
			msg: &larkim.EventMessage{
				MessageId: ptrStr("msg003"),
			},
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{UserId: nil},
			},
			want: domain.Message{MessageID: "msg003"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertEventMessage(tt.msg, tt.sender)
			if got.MessageID != tt.want.MessageID {
				t.Errorf("MessageID = %q, want %q", got.MessageID, tt.want.MessageID)
			}
			if got.ChatID != tt.want.ChatID {
				t.Errorf("ChatID = %q, want %q", got.ChatID, tt.want.ChatID)
			}
			if got.ChatType != tt.want.ChatType {
				t.Errorf("ChatType = %q, want %q", got.ChatType, tt.want.ChatType)
			}
			if got.MsgType != tt.want.MsgType {
				t.Errorf("MsgType = %q, want %q", got.MsgType, tt.want.MsgType)
			}
			if got.Content != tt.want.Content {
				t.Errorf("Content = %q, want %q", got.Content, tt.want.Content)
			}
			if got.SenderID != tt.want.SenderID {
				t.Errorf("SenderID = %q, want %q", got.SenderID, tt.want.SenderID)
			}
			if got.CreateTime != tt.want.CreateTime {
				t.Errorf("CreateTime = %q, want %q", got.CreateTime, tt.want.CreateTime)
			}
		})
	}
}

// --- 分发逻辑测试 ---

func TestEventDispatcherHandleContactUserCreated(t *testing.T) {
	ctx := context.Background()
	handler := &mockContactHandler{}
	d := newTestDispatcher(WithContactHandler(handler))

	event := &larkcontact.P2UserCreatedV3{
		Event: &larkcontact.P2UserCreatedV3Data{
			Object: &larkcontact.UserEvent{
				UserId: ptrStr("uid001"),
				Name:   ptrStr("李四"),
			},
		},
	}

	if err := d.handleUserCreated(ctx, event); err != nil {
		t.Fatalf("handleUserCreated returned error: %v", err)
	}
	if handler.lastUserCreated == nil {
		t.Fatal("OnUserCreated was not called")
	}
	if handler.lastUserCreated.UserID != "uid001" {
		t.Errorf("UserID = %q, want %q", handler.lastUserCreated.UserID, "uid001")
	}
	if handler.lastUserCreated.Name != "李四" {
		t.Errorf("Name = %q, want %q", handler.lastUserCreated.Name, "李四")
	}
}

func TestEventDispatcherHandleContactUserDeleted(t *testing.T) {
	ctx := context.Background()
	handler := &mockContactHandler{}
	d := newTestDispatcher(WithContactHandler(handler))

	event := &larkcontact.P2UserDeletedV3{
		Event: &larkcontact.P2UserDeletedV3Data{
			Object: &larkcontact.UserEvent{
				UserId: ptrStr("uid002"),
			},
		},
	}

	if err := d.handleUserDeleted(ctx, event); err != nil {
		t.Fatalf("handleUserDeleted returned error: %v", err)
	}
	if handler.lastUserDeleted == nil {
		t.Fatal("OnUserDeleted was not called")
	}
	if handler.lastUserDeleted.UserID != "uid002" {
		t.Errorf("UserID = %q, want %q", handler.lastUserDeleted.UserID, "uid002")
	}
}

func TestEventDispatcherHandleContactDeptCreated(t *testing.T) {
	ctx := context.Background()
	handler := &mockContactHandler{}
	d := newTestDispatcher(WithContactHandler(handler))

	event := &larkcontact.P2DepartmentCreatedV3{
		Event: &larkcontact.P2DepartmentCreatedV3Data{
			Object: &larkcontact.DepartmentEvent{
				DepartmentId: ptrStr("dept001"),
				Name:         ptrStr("研发部"),
			},
		},
	}

	if err := d.handleDeptCreated(ctx, event); err != nil {
		t.Fatalf("handleDeptCreated returned error: %v", err)
	}
	if handler.lastDeptCreated == nil {
		t.Fatal("OnDeptCreated was not called")
	}
	if handler.lastDeptCreated.DepartmentID != "dept001" {
		t.Errorf("DepartmentID = %q, want %q", handler.lastDeptCreated.DepartmentID, "dept001")
	}
}

func TestEventDispatcherHandleDeviceChanged(t *testing.T) {
	ctx := context.Background()
	handler := &mockDeviceHandler{}
	d := newTestDispatcher(WithDeviceHandler(handler))

	event := &larksecurity.P2DeviceRecordDeviceChangeEventV2{
		Event: &larksecurity.P2DeviceRecordDeviceChangeEventV2Data{
			After: &larksecurity.DeviceChangeEvent{
				DeviceRecordId: ptrStr("dev001"),
				DeviceName:     ptrStr("Test Device"),
			},
		},
	}

	if err := d.handleDeviceChanged(ctx, event); err != nil {
		t.Fatalf("handleDeviceChanged returned error: %v", err)
	}
	if handler.lastDevice == nil {
		t.Fatal("OnDeviceChanged was not called")
	}
	if handler.lastDevice.DeviceID != "dev001" {
		t.Errorf("DeviceID = %q, want %q", handler.lastDevice.DeviceID, "dev001")
	}
	if handler.lastDevice.DeviceName != "Test Device" {
		t.Errorf("DeviceName = %q, want %q", handler.lastDevice.DeviceName, "Test Device")
	}
}

func TestEventDispatcherHandleDeviceChangedUsesBeforeWhenAfterNil(t *testing.T) {
	ctx := context.Background()
	handler := &mockDeviceHandler{}
	d := newTestDispatcher(WithDeviceHandler(handler))

	// 删除事件：After 为 nil，使用 Before
	event := &larksecurity.P2DeviceRecordDeviceChangeEventV2{
		Event: &larksecurity.P2DeviceRecordDeviceChangeEventV2Data{
			After: nil,
			Before: &larksecurity.DeviceChangeEvent{
				DeviceRecordId: ptrStr("dev002"),
			},
		},
	}

	if err := d.handleDeviceChanged(ctx, event); err != nil {
		t.Fatalf("handleDeviceChanged returned error: %v", err)
	}
	if handler.lastDevice == nil {
		t.Fatal("OnDeviceChanged was not called")
	}
	if handler.lastDevice.DeviceID != "dev002" {
		t.Errorf("DeviceID = %q, want %q", handler.lastDevice.DeviceID, "dev002")
	}
}

func TestEventDispatcherHandleMessageReceived(t *testing.T) {
	ctx := context.Background()
	handler := &mockMessageHandler{}
	d := newTestDispatcher(WithMessageHandler(handler))

	event := &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{
			Sender: &larkim.EventSender{
				SenderId: &larkim.UserId{UserId: ptrStr("sender001")},
			},
			Message: &larkim.EventMessage{
				MessageId:   ptrStr("msg001"),
				ChatId:      ptrStr("chat001"),
				ChatType:    ptrStr("p2p"),
				MessageType: ptrStr("text"),
				Content:     ptrStr(`{"text":"hello"}`),
				CreateTime:  ptrStr("1740000000000"),
			},
		},
	}

	if err := d.handleMessageReceived(ctx, event); err != nil {
		t.Fatalf("handleMessageReceived returned error: %v", err)
	}
	if handler.lastMessage == nil {
		t.Fatal("OnMessageReceived was not called")
	}
	if handler.lastMessage.MessageID != "msg001" {
		t.Errorf("MessageID = %q, want %q", handler.lastMessage.MessageID, "msg001")
	}
	if handler.lastMessage.SenderID != "sender001" {
		t.Errorf("SenderID = %q, want %q", handler.lastMessage.SenderID, "sender001")
	}
}

func TestEventDispatcherHandleBotChatEntered(t *testing.T) {
	ctx := context.Background()
	handler := &mockMessageHandler{}
	d := newTestDispatcher(WithMessageHandler(handler))

	userID := "user001"
	chatID := "chat001"
	event := &larkim.P2ChatAccessEventBotP2pChatEnteredV1{
		Event: &larkim.P2ChatAccessEventBotP2pChatEnteredV1Data{
			ChatId:     &chatID,
			OperatorId: &larkim.UserId{UserId: &userID},
		},
	}

	if err := d.handleBotChatEntered(ctx, event); err != nil {
		t.Fatalf("handleBotChatEntered returned error: %v", err)
	}
	if handler.lastUserID != "user001" {
		t.Errorf("UserID = %q, want %q", handler.lastUserID, "user001")
	}
	if handler.lastChatID != "chat001" {
		t.Errorf("ChatID = %q, want %q", handler.lastChatID, "chat001")
	}
}

// --- 菜单事件测试 ---

// mockMenuHandler 实现 domain.MenuEventHandler，记录调用供断言。
type mockMenuHandler struct {
	lastEvent *domain.BotMenuEvent
}

func (m *mockMenuHandler) OnBotMenuClicked(_ context.Context, event *domain.BotMenuEvent) error {
	m.lastEvent = event
	return nil
}

// TestEventDispatcher_OnBotMenuV6_Dispatches 验证 EventKey / UserID / OpenID 被正确转换并传给 handler。
func TestEventDispatcher_OnBotMenuV6_Dispatches(t *testing.T) {
	ctx := context.Background()
	handler := &mockMenuHandler{}
	d := newTestDispatcher(WithMenuHandler(handler))

	eventKey := "device_status"
	userID := "user-abc"
	openID := "open-abc"
	event := &larkapplication.P2BotMenuV6{
		Event: &larkapplication.P2BotMenuV6Data{
			EventKey: &eventKey,
			Operator: &larkapplication.Operator{
				OperatorId: &larkapplication.UserId{
					UserId: &userID,
					OpenId: &openID,
				},
			},
		},
	}

	if err := d.handleBotMenu(ctx, event); err != nil {
		t.Fatalf("handleBotMenu returned error: %v", err)
	}
	if handler.lastEvent == nil {
		t.Fatal("OnBotMenuClicked was not called")
	}
	if handler.lastEvent.EventKey != "device_status" {
		t.Errorf("EventKey = %q, want %q", handler.lastEvent.EventKey, "device_status")
	}
	if handler.lastEvent.UserID != "user-abc" {
		t.Errorf("UserID = %q, want %q", handler.lastEvent.UserID, "user-abc")
	}
	if handler.lastEvent.OpenID != "open-abc" {
		t.Errorf("OpenID = %q, want %q", handler.lastEvent.OpenID, "open-abc")
	}
	if handler.lastEvent.Type != domain.EventBotMenu {
		t.Errorf("Type = %q, want %q", handler.lastEvent.Type, domain.EventBotMenu)
	}
}

// TestEventDispatcher_OnBotMenuV6_NilHandler 验证 handler 为 nil 时静默跳过，不 panic。
func TestEventDispatcher_OnBotMenuV6_NilHandler(t *testing.T) {
	ctx := context.Background()
	d := newTestDispatcher() // 不注入 menuHandler

	eventKey := "device_status"
	event := &larkapplication.P2BotMenuV6{
		Event: &larkapplication.P2BotMenuV6Data{
			EventKey: &eventKey,
		},
	}

	if err := d.handleBotMenu(ctx, event); err != nil {
		t.Errorf("handleBotMenu with nil handler = %v, want nil", err)
	}
}

// TestEventDispatcher_OnBotMenuV6_NilEvent 验证 event.Event 为 nil 时静默跳过，不 panic。
func TestEventDispatcher_OnBotMenuV6_NilEvent(t *testing.T) {
	ctx := context.Background()
	handler := &mockMenuHandler{}
	d := newTestDispatcher(WithMenuHandler(handler))

	event := &larkapplication.P2BotMenuV6{
		Event: nil,
	}

	if err := d.handleBotMenu(ctx, event); err != nil {
		t.Errorf("handleBotMenu with nil event = %v, want nil", err)
	}
	if handler.lastEvent != nil {
		t.Errorf("OnBotMenuClicked should not have been called, but was")
	}
}

// TestEventDispatcherNilHandlerSkip 验证 handler 为 nil 时跳过，不 panic。
func TestEventDispatcherNilHandlerSkip(t *testing.T) {
	ctx := context.Background()
	d := newTestDispatcher() // 不注入任何 handler

	if err := d.handleUserCreated(ctx, &larkcontact.P2UserCreatedV3{
		Event: &larkcontact.P2UserCreatedV3Data{Object: &larkcontact.UserEvent{}},
	}); err != nil {
		t.Errorf("handleUserCreated with nil handler = %v, want nil", err)
	}

	if err := d.handleDeviceChanged(ctx, &larksecurity.P2DeviceRecordDeviceChangeEventV2{
		Event: &larksecurity.P2DeviceRecordDeviceChangeEventV2Data{
			After: &larksecurity.DeviceChangeEvent{},
		},
	}); err != nil {
		t.Errorf("handleDeviceChanged with nil handler = %v, want nil", err)
	}

	if err := d.handleMessageReceived(ctx, &larkim.P2MessageReceiveV1{
		Event: &larkim.P2MessageReceiveV1Data{Message: &larkim.EventMessage{}},
	}); err != nil {
		t.Errorf("handleMessageReceived with nil handler = %v, want nil", err)
	}

	if err := d.handleBotChatEntered(ctx, &larkim.P2ChatAccessEventBotP2pChatEnteredV1{
		Event: &larkim.P2ChatAccessEventBotP2pChatEnteredV1Data{},
	}); err != nil {
		t.Errorf("handleBotChatEntered with nil handler = %v, want nil", err)
	}
}
