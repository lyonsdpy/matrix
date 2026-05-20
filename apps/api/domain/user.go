package domain

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FeishuID string `json:"feishu_id"`
}

type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
