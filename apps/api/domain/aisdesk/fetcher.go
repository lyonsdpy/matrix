package aisdesk

import "context"

// DeviceFetcher 亚信桌面管理设备数据读取能力。
// 所有方法均为只读操作，仅返回安装了 Agent 的设备（TDevice.AgentInstalled=1）。
type DeviceFetcher interface {
	// ListDevices 返回所有安装了 Agent 的设备基础信息（TDevice + TComputer）。
	ListDevices(ctx context.Context) ([]Device, error)

	// ListInstalledSoftware 返回指定设备的已安装软件列表（AgentInstalled=1 过滤由 ListDevices 保证，此处按 deviceID 查询）。
	ListInstalledSoftware(ctx context.Context, deviceID int) ([]InstalledSoftware, error)

	// ListHardwareComponents 返回指定设备的硬件组件详情列表（AgentInstalled=1 过滤由 ListDevices 保证，此处按 deviceID 查询）。
	ListHardwareComponents(ctx context.Context, deviceID int) ([]HardwareComponent, error)
}
