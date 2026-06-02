// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"fmt"
	"strings"

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

// placeholderValues 飞书 device_record 字段缺值时回填的占位字符串集合。
// 设备没上报真实硬件标识（iOS/Android 隐私限制、未授权 MDM 等）时，飞书会返回这些字面量。
// 写入 Neo4j 之前统一过滤为空串，前端就能正常显示 "—"。
// 注意：判定时已经 ToLower 比较，所以这里的 key 都用小写。
var placeholderValues = map[string]struct{}{
	"default string":      {}, // 通用占位
	"unknown":             {},
	"00000000-0000-0000-0000-000000000000": {}, // UUID 默认
	"00:00:00:00:00:00":   {}, // MAC 默认
}

func sanitizeField(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if _, ok := placeholderValues[strings.ToLower(t)]; ok {
		return ""
	}
	return t
}

// convertDevice 将 SDK DeviceRecord 转换为 Device。
// 飞书原始字段：https://open.feishu.cn/document/security_and_compliance-v1/security_and_compliance-v2/device_record/list
// device_system / device_terminal_type / device_status / device_ownership / certification_level
// 都是整数编号，这里统一以 string 存储原始编号，前端做枚举翻译。
// 硬件标识字段（serial/uuid/mac 等）经 sanitizeField 过滤已知占位值。
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
		dev.CurrentUserID = *d.CurrentUserId
	}
	if d.LatestUserId != nil {
		dev.LatestUserID = *d.LatestUserId
	}
	if d.DeviceStatus != nil {
		dev.TrustLevel = fmt.Sprintf("%d", *d.DeviceStatus)
	}
	if d.DeviceOwnership != nil {
		dev.Ownership = fmt.Sprintf("%d", *d.DeviceOwnership)
		dev.Status = dev.Ownership // 兼容旧字段
	}
	if d.CertificationLevel != nil {
		dev.Certification = fmt.Sprintf("%d", *d.CertificationLevel)
	}
	if d.SerialNumber != nil {
		dev.SerialNumber = sanitizeField(*d.SerialNumber)
	}
	if d.DiskSerialNumber != nil {
		dev.DiskSerialNumber = sanitizeField(*d.DiskSerialNumber)
	}
	if d.Uuid != nil {
		dev.BoardUUID = sanitizeField(*d.Uuid)
	}
	if d.MacAddress != nil {
		dev.MACAddress = sanitizeField(*d.MacAddress)
	}
	if d.Model != nil {
		dev.Model = sanitizeField(*d.Model)
	}
	if d.DeviceSystem != nil {
		dev.OSCode = fmt.Sprintf("%d", *d.DeviceSystem)
	}
	if d.Version != nil {
		dev.Version = sanitizeField(*d.Version)
	}
	if d.IsManaged != nil {
		dev.IsManaged = *d.IsManaged
	}
	if d.MdmDeviceId != nil {
		dev.MDMDeviceID = sanitizeField(*d.MdmDeviceId)
	}
	if d.MdmProviderName != nil {
		dev.MDMProvider = sanitizeField(*d.MdmProviderName)
	}
	return dev
}
