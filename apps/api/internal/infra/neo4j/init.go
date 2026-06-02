package neo4jdb

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// InitSchema 在 Neo4j 里创建 Device / User 节点的约束和索引。
// IF NOT EXISTS 保证幂等，重复调用不会报错。
func InitSchema(ctx context.Context, driver neo4j.Driver, dbName string) error {
	stmts := []string{
		// 唯一约束同时隐式创建索引
		`CREATE CONSTRAINT device_id_unique IF NOT EXISTS FOR (d:Device) REQUIRE d.id IS UNIQUE`,
		// 按管理 IP 查询是高频操作，单独加索引
		`CREATE INDEX device_mip_idx IF NOT EXISTS FOR (d:Device) ON (d.mip)`,
		`CREATE INDEX device_name_idx IF NOT EXISTS FOR (d:Device) ON (d.name)`,
		// User 节点以 feishu_id（open_id）为业务唯一键，同步与登录关联均依赖它
		`CREATE CONSTRAINT user_feishu_id_unique IF NOT EXISTS FOR (u:User) REQUIRE u.feishu_id IS UNIQUE`,
		// Department 节点以飞书 department_id 为唯一键，PARENT_OF / MEMBER_OF 关系依赖它
		`CREATE CONSTRAINT dept_id_unique IF NOT EXISTS FOR (d:Department) REQUIRE d.department_id IS UNIQUE`,
		`CREATE INDEX dept_name_idx IF NOT EXISTS FOR (d:Department) ON (d.name)`,
		// Endpoint 节点以飞书 device_record_id 为业务唯一键，sync 与 CURRENT_LOGIN / LATEST_LOGIN 关系都依赖它
		`CREATE CONSTRAINT endpoint_feishu_id_unique IF NOT EXISTS FOR (e:Endpoint) REQUIRE e.feishu_device_id IS UNIQUE`,
		// 按 id 取详情高频，建索引（唯一约束隐式索引是 feishu_device_id 的，id 单独索引）
		`CREATE INDEX endpoint_id_idx IF NOT EXISTS FOR (e:Endpoint) ON (e.id)`,
		`CREATE INDEX endpoint_name_idx IF NOT EXISTS FOR (e:Endpoint) ON (e.name)`,
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
