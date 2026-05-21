/*
可接入网络的终端设备，包含：电脑、打印机
*/

package domain

type Endpoint struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         EndpointType `json:"type"`
	SerialNumber string       `json:"serial_number"`
	AssetNumber  string       `json:"asset_number"`
}

type EndpointType string

const (
	EndpointTypePC         EndpointType = "PC"
	EndpointTypeLaptop     EndpointType = "LAPTOP"
	EndpointTypePrinter    EndpointType = "PRINTER"
	EndpointTypeTV         EndpointType = "TV"
	EndpointTypeAttendance EndpointType = "ATTENDANCE"
	EdnpointTypePhone      EndpointType = "PHONE"
	EndpointTypeTablet     EndpointType = "TABLET"
	EndpointTypeOther      EndpointType = "OTHER"
)
