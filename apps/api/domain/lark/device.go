package lark

import "context"

// DeviceFetcher 设备信息全量拉取能力。
type DeviceFetcher interface {
	// FetchAllDevices 分页查询所有设备信息。
	FetchAllDevices(ctx context.Context) ([]Device, error)

	// GetDevice 获取单个设备详情。
	GetDevice(ctx context.Context, deviceID string) (*Device, error)
}
