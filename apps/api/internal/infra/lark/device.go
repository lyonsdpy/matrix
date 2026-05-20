// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"fmt"

	larksecurity "github.com/larksuite/oapi-sdk-go/v3/service/security_and_compliance/v2"
	"go.uber.org/zap"
)

// DeviceFetcher 设备信息全量拉取能力。
type DeviceFetcher interface {
	// FetchAllDevices 分页查询所有设备信息。
	FetchAllDevices(ctx context.Context) ([]Device, error)

	// GetDevice 获取单个设备详情。
	GetDevice(ctx context.Context, deviceID string) (*Device, error)
}

// deviceFetcher 实现 DeviceFetcher。
type deviceFetcher struct {
	client *Client
	logger *zap.Logger
}

// compile-time interface check
var _ DeviceFetcher = (*deviceFetcher)(nil)

// NewDeviceFetcher 创建设备信息拉取器。
func NewDeviceFetcher(client *Client, logger *zap.Logger) DeviceFetcher {
	return &deviceFetcher{client: client, logger: logger}
}

// FetchAllDevices 分页查询所有设备信息。
func (f *deviceFetcher) FetchAllDevices(ctx context.Context) ([]Device, error) {
	var result []Device
	var pageToken string

	for {
		req := larksecurity.NewListDeviceRecordReqBuilder().
			PageSize(100).
			PageToken(pageToken).
			UserIdType("user_id").
			Build()

		resp, err := f.client.SecurityAndCompliance.V2.DeviceRecord.List(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("device: list all devices: %w", err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("device: list all devices: code=%d, msg=%s", resp.Code, resp.Msg)
		}
		if resp.Data == nil {
			break
		}

		for _, d := range resp.Data.Items {
			result = append(result, convertDevice(d))
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

// GetDevice 获取单个设备详情。
func (f *deviceFetcher) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	req := larksecurity.NewGetDeviceRecordReqBuilder().
		DeviceRecordId(deviceID).
		UserIdType("user_id").
		Build()

	resp, err := f.client.SecurityAndCompliance.V2.DeviceRecord.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("device: get device %s: %w", deviceID, err)
	}
	if !resp.Success() {
		return nil, fmt.Errorf("device: get device %s: code=%d, msg=%s", deviceID, resp.Code, resp.Msg)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("device: get device %s: empty response", deviceID)
	}

	d := convertDevice(resp.Data.DeviceRecord)
	return &d, nil
}

// convertDevice 将 SDK DeviceRecord 转换为 Device。
func convertDevice(d *larksecurity.DeviceRecord) Device {
	if d == nil {
		return Device{}
	}
	dev := Device{}
	if d.DeviceRecordId != nil {
		dev.DeviceID = *d.DeviceRecordId
	}
	if d.DeviceName != nil {
		dev.DeviceName = *d.DeviceName
	}
	if d.DeviceTerminalType != nil {
		dev.Platform = fmt.Sprintf("%d", *d.DeviceTerminalType)
	}
	if d.CurrentUserId != nil {
		dev.UserID = *d.CurrentUserId
	}
	if d.DeviceStatus != nil {
		dev.TrustLevel = fmt.Sprintf("%d", *d.DeviceStatus)
	}
	if d.DeviceOwnership != nil {
		dev.Status = fmt.Sprintf("%d", *d.DeviceOwnership)
	}
	if d.SerialNumber != nil {
		dev.SerialNumber = *d.SerialNumber
	}
	// OSVersion 和 LastOnlineTime 在 SDK 响应中不可用，保持零值
	return dev
}
