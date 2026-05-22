package neo4jdb

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"matrix/api/pkg/config"
)

// Open 创建并验证 Neo4j 驱动连接。
// URI 为空时返回 nil driver，由 repository 层降级为内存存根（开发环境无需启动 Neo4j）。
func Open(cfg config.Neo4j) (neo4j.Driver, error) {
	if cfg.URI == "" {
		return nil, nil
	}
	driver, err := neo4j.NewDriver(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j: new driver: %w", err)
	}
	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		_ = driver.Close(context.Background())
		return nil, fmt.Errorf("neo4j: verify connectivity %s: %w", cfg.URI, err)
	}
	return driver, nil
}
