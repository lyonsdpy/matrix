// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"fmt"

	sdkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkapplication "github.com/larksuite/oapi-sdk-go/v3/service/application/v6"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larksecurity "github.com/larksuite/oapi-sdk-go/v3/service/security_and_compliance/v2"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"

	domain "matrix/api/domain/lark"
)

// EventDispatcher 管理飞书 SDK 长连接，统一接收所有事件并分发给注册的 handler。
// 使用飞书 Go SDK 的 ws 包建立 WebSocket 长连接。
type EventDispatcher struct {
	appID          string
	appSecret      string
	contactHandler domain.ContactEventHandler
	deviceHandler  domain.DeviceEventHandler
	messageHandler domain.MessageEventHandler
	cardHandler    domain.CardEventHandler
	menuHandler    domain.MenuEventHandler
	logger         *zap.Logger
}

// EventDispatcherOption 配置事件分发器的选项函数。
type EventDispatcherOption func(*EventDispatcher)

// WithContactHandler 注册通讯录事件处理器。
func WithContactHandler(h domain.ContactEventHandler) EventDispatcherOption {
	return func(d *EventDispatcher) { d.contactHandler = h }
}

// WithDeviceHandler 注册设备变更事件处理器。
func WithDeviceHandler(h domain.DeviceEventHandler) EventDispatcherOption {
	return func(d *EventDispatcher) { d.deviceHandler = h }
}

// WithMessageHandler 注册消息事件处理器。
func WithMessageHandler(h domain.MessageEventHandler) EventDispatcherOption {
	return func(d *EventDispatcher) { d.messageHandler = h }
}

// WithCardHandler 注册卡片交互事件处理器。
// 通过 EventDispatcher.OnP2CardActionTrigger 注册，走 WS 长连接接收 card.action.trigger 回调。
func WithCardHandler(h domain.CardEventHandler) EventDispatcherOption {
	return func(d *EventDispatcher) { d.cardHandler = h }
}

// WithMenuHandler 注册机器人菜单点击事件处理器。
func WithMenuHandler(h domain.MenuEventHandler) EventDispatcherOption {
	return func(d *EventDispatcher) { d.menuHandler = h }
}

// NewEventDispatcher 创建事件分发器。
// handler 参数均可为 nil，表示不处理该类事件。
func NewEventDispatcher(appID, appSecret string, logger *zap.Logger, opts ...EventDispatcherOption) *EventDispatcher {
	d := &EventDispatcher{
		appID:     appID,
		appSecret: appSecret,
		logger:    logger,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Start 建立长连接并开始接收事件（阻塞直到 ctx 取消）。
// 应在 goroutine 中调用，通过 ctx 取消来停止。
func (d *EventDispatcher) Start(ctx context.Context) error {
	evtDispatcher := sdkdispatcher.NewEventDispatcher("", "").
		OnP2UserCreatedV3(func(ctx context.Context, event *larkcontact.P2UserCreatedV3) error {
			return d.handleUserCreated(ctx, event)
		}).
		OnP2UserDeletedV3(func(ctx context.Context, event *larkcontact.P2UserDeletedV3) error {
			return d.handleUserDeleted(ctx, event)
		}).
		OnP2UserUpdatedV3(func(ctx context.Context, event *larkcontact.P2UserUpdatedV3) error {
			return d.handleUserUpdated(ctx, event)
		}).
		OnP2DepartmentCreatedV3(func(ctx context.Context, event *larkcontact.P2DepartmentCreatedV3) error {
			return d.handleDeptCreated(ctx, event)
		}).
		OnP2DepartmentDeletedV3(func(ctx context.Context, event *larkcontact.P2DepartmentDeletedV3) error {
			return d.handleDeptDeleted(ctx, event)
		}).
		OnP2DepartmentUpdatedV3(func(ctx context.Context, event *larkcontact.P2DepartmentUpdatedV3) error {
			return d.handleDeptUpdated(ctx, event)
		}).
		OnP2DeviceRecordDeviceChangeEventV2(func(ctx context.Context, event *larksecurity.P2DeviceRecordDeviceChangeEventV2) error {
			return d.handleDeviceChanged(ctx, event)
		}).
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			return d.handleMessageReceived(ctx, event)
		}).
		OnP2ChatAccessEventBotP2pChatEnteredV1(func(ctx context.Context, event *larkim.P2ChatAccessEventBotP2pChatEnteredV1) error {
			return d.handleBotChatEntered(ctx, event)
		}).
		OnP2BotMenuV6(func(ctx context.Context, event *larkapplication.P2BotMenuV6) error {
			return d.handleBotMenu(ctx, event)
		}).
		// 订阅但不处理的事件：注册空 handler 避免 SDK 打印 "not found handler" 噪音日志
		OnP2MessageReadV1(func(_ context.Context, _ *larkim.P2MessageReadV1) error { return nil }).
		OnP1UserStatusChangedV3(func(_ context.Context, _ *larkcontact.P1UserStatusChangedV3) error { return nil })

	if d.cardHandler != nil {
		evtDispatcher.OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			return nil, d.handleCardAction(ctx, event)
		})
	}

	wsClient := larkws.NewClient(d.appID, d.appSecret,
		larkws.WithEventHandler(evtDispatcher),
		larkws.WithAutoReconnect(true),
	)

	return wsClient.Start(ctx)
}

// Stop 关闭长连接。WS 模式通过 ctx 取消停止，此方法为兼容接口。
func (d *EventDispatcher) Stop(_ context.Context) error {
	return nil
}

// --- 事件处理函数 ---

func (d *EventDispatcher) handleUserCreated(ctx context.Context, event *larkcontact.P2UserCreatedV3) error {
	if d.contactHandler == nil || event.Event == nil {
		return nil
	}
	user := convertUserEvent(event.Event.Object)
	if err := d.contactHandler.OnUserCreated(ctx, &user); err != nil {
		d.logger.Error("event_dispatcher: OnUserCreated", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleUserDeleted(ctx context.Context, event *larkcontact.P2UserDeletedV3) error {
	if d.contactHandler == nil || event.Event == nil {
		return nil
	}
	user := convertUserEvent(event.Event.Object)
	if err := d.contactHandler.OnUserDeleted(ctx, &user); err != nil {
		d.logger.Error("event_dispatcher: OnUserDeleted", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleUserUpdated(ctx context.Context, event *larkcontact.P2UserUpdatedV3) error {
	if d.contactHandler == nil || event.Event == nil {
		return nil
	}
	user := convertUserEvent(event.Event.Object)
	if err := d.contactHandler.OnUserUpdated(ctx, &user); err != nil {
		d.logger.Error("event_dispatcher: OnUserUpdated", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleDeptCreated(ctx context.Context, event *larkcontact.P2DepartmentCreatedV3) error {
	if d.contactHandler == nil || event.Event == nil {
		return nil
	}
	dept := convertDepartmentEvent(event.Event.Object)
	if err := d.contactHandler.OnDeptCreated(ctx, &dept); err != nil {
		d.logger.Error("event_dispatcher: OnDeptCreated", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleDeptDeleted(ctx context.Context, event *larkcontact.P2DepartmentDeletedV3) error {
	if d.contactHandler == nil || event.Event == nil {
		return nil
	}
	dept := convertDepartmentEvent(event.Event.Object)
	if err := d.contactHandler.OnDeptDeleted(ctx, &dept); err != nil {
		d.logger.Error("event_dispatcher: OnDeptDeleted", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleDeptUpdated(ctx context.Context, event *larkcontact.P2DepartmentUpdatedV3) error {
	if d.contactHandler == nil || event.Event == nil {
		return nil
	}
	dept := convertDepartmentEvent(event.Event.Object)
	if err := d.contactHandler.OnDeptUpdated(ctx, &dept); err != nil {
		d.logger.Error("event_dispatcher: OnDeptUpdated", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleDeviceChanged(ctx context.Context, event *larksecurity.P2DeviceRecordDeviceChangeEventV2) error {
	if d.deviceHandler == nil || event.Event == nil {
		return nil
	}
	// 优先使用变更后的状态（After），删除事件时使用变更前的状态（Before）
	snapshot := event.Event.After
	if snapshot == nil {
		snapshot = event.Event.Before
	}
	device := convertDeviceChangeEvent(snapshot)
	if err := d.deviceHandler.OnDeviceChanged(ctx, &device); err != nil {
		d.logger.Error("event_dispatcher: OnDeviceChanged", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleMessageReceived(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if d.messageHandler == nil || event.Event == nil {
		return nil
	}
	msg := convertEventMessage(event.Event.Message, event.Event.Sender)
	if err := d.messageHandler.OnMessageReceived(ctx, &msg); err != nil {
		d.logger.Error("event_dispatcher: OnMessageReceived", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleBotMenu(ctx context.Context, event *larkapplication.P2BotMenuV6) error {
	if d.menuHandler == nil || event.Event == nil {
		return nil
	}
	var userID, openID, eventKey string
	if event.Event.EventKey != nil {
		eventKey = *event.Event.EventKey
	}
	if event.Event.Operator != nil && event.Event.Operator.OperatorId != nil {
		if event.Event.Operator.OperatorId.UserId != nil {
			userID = *event.Event.Operator.OperatorId.UserId
		}
		if event.Event.Operator.OperatorId.OpenId != nil {
			openID = *event.Event.Operator.OperatorId.OpenId
		}
	}
	menuEvent := domain.BotMenuEvent{
		Type:     domain.EventBotMenu,
		EventKey: eventKey,
		UserID:   userID,
		OpenID:   openID,
	}
	if err := d.menuHandler.OnBotMenuClicked(ctx, &menuEvent); err != nil {
		d.logger.Error("event_dispatcher: OnBotMenuClicked", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleBotChatEntered(ctx context.Context, event *larkim.P2ChatAccessEventBotP2pChatEnteredV1) error {
	if d.messageHandler == nil || event.Event == nil {
		return nil
	}
	var userID string
	if event.Event.OperatorId != nil && event.Event.OperatorId.UserId != nil {
		userID = *event.Event.OperatorId.UserId
	}
	var chatID string
	if event.Event.ChatId != nil {
		chatID = *event.Event.ChatId
	}
	if err := d.messageHandler.OnBotChatEntered(ctx, userID, chatID); err != nil {
		d.logger.Error("event_dispatcher: OnBotChatEntered", zap.Error(err))
	}
	return nil
}

func (d *EventDispatcher) handleCardAction(ctx context.Context, event *callback.CardActionTriggerEvent) error {
	if d.cardHandler == nil || event.Event == nil {
		return nil
	}
	action := domain.CardAction{}
	if event.Event.Action != nil {
		if v, ok := event.Event.Action.Value["action"].(string); ok {
			action.Action = v
		}
	}
	if event.Event.Operator != nil {
		if event.Event.Operator.UserID != nil {
			action.UserID = *event.Event.Operator.UserID
		}
		action.OpenID = event.Event.Operator.OpenID
	}
	if event.Event.Context != nil {
		action.ChatID = event.Event.Context.OpenChatID
	}
	if err := d.cardHandler.OnCardAction(ctx, &action); err != nil {
		d.logger.Error("event_dispatcher: OnCardAction", zap.Error(err))
	}
	return nil
}

// --- 转换函数 ---

// convertUserEvent 将 SDK UserEvent（事件体中的用户数据）转换为 domain.User。
func convertUserEvent(u *larkcontact.UserEvent) domain.User {
	if u == nil {
		return domain.User{}
	}
	user := domain.User{
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

// convertDepartmentEvent 将 SDK DepartmentEvent（事件体中的部门数据）转换为 domain.Department。
// 注意：事件体不含 MemberCount，该字段保持零值。
func convertDepartmentEvent(d *larkcontact.DepartmentEvent) domain.Department {
	if d == nil {
		return domain.Department{}
	}
	dept := domain.Department{}
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
	dept.Status = deptStatusToInt(d.Status)
	return dept
}

// convertDeviceChangeEvent 将 SDK DeviceChangeEvent（设备变更快照）转换为 domain.Device。
func convertDeviceChangeEvent(e *larksecurity.DeviceChangeEvent) domain.Device {
	if e == nil {
		return domain.Device{}
	}
	dev := domain.Device{}
	if e.DeviceRecordId != nil {
		dev.DeviceID = *e.DeviceRecordId
	}
	if e.DeviceName != nil {
		dev.DeviceName = *e.DeviceName
	}
	if e.DeviceSystem != nil {
		dev.Platform = fmt.Sprintf("%d", *e.DeviceSystem)
	}
	if e.CurrentUserId != nil && e.CurrentUserId.UserId != nil {
		dev.UserID = *e.CurrentUserId.UserId
	}
	if e.DeviceOwnership != nil {
		dev.Status = fmt.Sprintf("%d", *e.DeviceOwnership)
	}
	if e.DeviceStatus != nil {
		dev.TrustLevel = fmt.Sprintf("%d", *e.DeviceStatus)
	}
	if e.SerialNumber != nil {
		dev.SerialNumber = *e.SerialNumber
	}
	return dev
}

// convertEventMessage 将 SDK EventMessage 和 EventSender 转换为 domain.Message。
func convertEventMessage(msg *larkim.EventMessage, sender *larkim.EventSender) domain.Message {
	if msg == nil {
		return domain.Message{}
	}
	m := domain.Message{}
	if msg.MessageId != nil {
		m.MessageID = *msg.MessageId
	}
	if msg.ChatId != nil {
		m.ChatID = *msg.ChatId
	}
	if msg.ChatType != nil {
		m.ChatType = *msg.ChatType
	}
	if msg.MessageType != nil {
		m.MsgType = *msg.MessageType
	}
	if msg.Content != nil {
		m.Content = *msg.Content
	}
	if msg.CreateTime != nil {
		m.CreateTime = *msg.CreateTime
	}
	if sender != nil && sender.SenderId != nil && sender.SenderId.UserId != nil {
		m.SenderID = *sender.SenderId.UserId
	}
	return m
}
