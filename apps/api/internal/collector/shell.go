package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ShellConfig Shell 命令采集器配置
type ShellConfig struct {
	Command     string   `json:"command"`      // 可执行文件路径或命令名，必填
	Args        []string `json:"args"`         // 参数列表
	TimeoutSecs int      `json:"timeout_secs"` // 默认 30
}

type shellCollector struct{}

func init() {
	Register(&shellCollector{})
}

func (c *shellCollector) Type() string { return "shell" }
func (c *shellCollector) Description() string {
	return "执行本地 Shell 命令并采集标准输出，适用于自定义采集脚本、SSH 探测、CLI 工具等"
}

func (c *shellCollector) Collect(ctx context.Context, rawConfig json.RawMessage) (*Result, error) {
	var cfg ShellConfig
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("解析 Shell 配置失败: %w", err)
	}
	if cfg.Command == "" {
		return nil, fmt.Errorf("command 不能为空")
	}
	timeout := 30 * time.Second
	if cfg.TimeoutSecs > 0 {
		timeout = time.Duration(cfg.TimeoutSecs) * time.Second
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, cfg.Command, cfg.Args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("命令执行失败: %w, stderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// 尝试以 JSON 形式保存，便于后续解析
	var data interface{}
	if json.Valid([]byte(output)) {
		data = json.RawMessage(output)
	} else {
		data = output
	}

	return &Result{
		Data:    data,
		Summary: fmt.Sprintf("cmd=%q exit=0 output_len=%d", cfg.Command, len(output)),
	}, nil
}
