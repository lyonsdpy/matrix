package collector

import (
	"context"
	"encoding/json"
	"fmt"
)

// Result 单次采集的输出
type Result struct {
	Data    any    `json:"data"`
	Summary string `json:"summary"` // 供日志/状态展示的一句话摘要
}

// Collector 采集器接口。
// 每种采集方式实现此接口，在各自的 init() 中调用 Register 注册。
type Collector interface {
	Type() string
	Description() string
	Collect(ctx context.Context, rawConfig json.RawMessage) (*Result, error)
}

// Info 采集器元信息，供 API 返回给前端
type Info struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

var registry = map[string]Collector{}

// Register 注册采集器，在 init() 中调用
func Register(c Collector) {
	registry[c.Type()] = c
}

// Get 按类型名取采集器
func Get(typ string) (Collector, error) {
	c, ok := registry[typ]
	if !ok {
		return nil, fmt.Errorf("未知采集器类型 %q，可用类型：%v", typ, availableTypes())
	}
	return c, nil
}

// List 返回所有已注册采集器的元信息
func List() []Info {
	result := make([]Info, 0, len(registry))
	for _, c := range registry {
		result = append(result, Info{Type: c.Type(), Description: c.Description()})
	}
	return result
}

func availableTypes() []string {
	types := make([]string, 0, len(registry))
	for k := range registry {
		types = append(types, k)
	}
	return types
}
