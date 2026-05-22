/*
可接入网络的终端设备，包含：电脑、打印机
*/

package domain

type Endpoint struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`               // 设备名称
	Type             EndpointType   `json:"type"`               // 设备类型
	Status           EndpointStatus `json:"status"`             // 设备状态
	AssetNumber      string         `json:"asset_number"`       // 资产编码
	SerialNumber     string         `json:"serial_number"`      // 生产序列号
	DiskSerialNumber string         `json:"disk_serial_number"` // 硬盘序列号
	Vendor           string         `json:"vendor"`             // 制造商
	Model            string         `json:"model"`              // 设备型号
	OSName           string         `json:"os_name"`            // 操作系统
	TemporalMeta                    // 时态元数据（版本、时间戳、软删除）
}

type EndpointType string

const (
	EndpointTypePC         EndpointType = "PC"         // 台式机
	EndpointTypeLaptop     EndpointType = "LAPTOP"     // 便携笔记本
	EndpointTypePrinter    EndpointType = "PRINTER"    // 打印机
	EndpointTypeTV         EndpointType = "TV"         // 智能电视
	EndpointTypeAttendance EndpointType = "ATTENDANCE" // 考勤机
	EdnpointTypePhone      EndpointType = "PHONE"      // 手机
	EndpointTypeTablet     EndpointType = "TABLET"     // 平板
	EndpointTypeOther      EndpointType = "OTHER"
)

type EndpointStatus string

const (
	EndpointStatusActive   EndpointStatus = "ACTIVE"
	EndpointStatusInactive EndpointStatus = "INACTIVE"
	EndpointStatusLost     EndpointStatus = "LOST"
	EndpointStatusRetired  EndpointStatus = "RETIRED"
)
