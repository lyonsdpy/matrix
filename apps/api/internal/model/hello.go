package model

// HelloRequest 打招呼请求参数
type HelloRequest struct {
	Name string `form:"name" binding:"required"` // 姓名，必填
}

// HelloResponse 打招呼响应结构
type HelloResponse struct {
	Message string `json:"message"` // 问候语
}
