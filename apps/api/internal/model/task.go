package model

import (
	"encoding/json"
	"time"
)

// Task 采集任务配置与运行状态
type Task struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	CollectorType string          `json:"collector_type"` // 采集器类型，对应注册表中的 key
	Config        json.RawMessage `json:"config"`         // 采集器专属配置，不同类型结构不同
	Enabled       bool            `json:"enabled"`
	IntervalSecs  int             `json:"interval_secs"` // 采集周期（秒），最小 5
	LastRunAt     *time.Time      `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time      `json:"next_run_at,omitempty"`
	Status        string          `json:"status"`                  // idle / running / error
	LastResult    string          `json:"last_result,omitempty"`   // 上次采集结果（JSON 字符串）
	LastError     string          `json:"last_error,omitempty"`    // 上次失败原因
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name          string          `json:"name"           binding:"required"`
	CollectorType string          `json:"collector_type" binding:"required"`
	Config        json.RawMessage `json:"config"`
	IntervalSecs  int             `json:"interval_secs"  binding:"required,min=5"`
}

// UpdateTaskRequest 更新任务请求（所有字段可选）
type UpdateTaskRequest struct {
	Name         *string         `json:"name"`
	Config       json.RawMessage `json:"config"`
	IntervalSecs *int            `json:"interval_secs" binding:"omitempty,min=5"`
}
