package domain

type IPv4Cidr struct {
	ID string `json:"id"`
}

type IPv4Addr struct {
	ID        string `json:"id"`
	IP        string `json:"address"`
	Mask      string `json:"mask"`
	Cidr      string `json:"cidr"`
	StartAddr uint64 `json:"start_addr"`
	EndAddr   uint64 `json:"end_addr"`
}
