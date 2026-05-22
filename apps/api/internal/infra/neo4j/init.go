package neo4jdb

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// InitSchema 在 Neo4j 里创建 Device 节点的约束和索引。
// IF NOT EXISTS 保证幂等，重复调用不会报错。
func InitSchema(ctx context.Context, driver neo4j.Driver, dbName string) error {
	stmts := []string{
		// 唯一约束同时隐式创建索引
		`CREATE CONSTRAINT device_id_unique IF NOT EXISTS FOR (d:Device) REQUIRE d.id IS UNIQUE`,
		// 按管理 IP 查询是高频操作，单独加索引
		`CREATE INDEX device_mip_idx IF NOT EXISTS FOR (d:Device) ON (d.mip)`,
		`CREATE INDEX device_name_idx IF NOT EXISTS FOR (d:Device) ON (d.name)`,
	}
	for _, stmt := range stmts {
		_, err := neo4j.ExecuteQuery(ctx, driver, stmt, nil,
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(dbName),
		)
		if err != nil {
			return fmt.Errorf("neo4j init schema: %w", err)
		}
	}
	return nil
}
