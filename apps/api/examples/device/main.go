// Device 操作示例：通过 GraphQL 客户端演示 Device 完整 CRUD + 连接关系。
// createDevice 支持 connectTo 参数，可在创建节点的同时建立关系，无需单独调用 createConnection。
// 输出为标准 GraphQL 响应格式 {"data": {...}}。
//
// 前置条件：服务已启动 → go run ./cmd/server/...
//
// 用法（在 apps/api 目录下执行）：
//
//	go run ./examples/device/...
//	go run ./examples/device/... -addr http://localhost:8080 -username admin -password admin123
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"matrix/api/examples/gqlclient"
)

const createDeviceMutation = `
	mutation CreateDevice($name: String!, $deviceType: String!, $mip: String!, $connectTo: [ID!]) {
		createDevice(name: $name, deviceType: $deviceType, mip: $mip, connectTo: $connectTo) {
			id name type mip version createdAt
			connections {
				target   { id name type mip }
				relation { id from_id to_id type }
			}
		}
	}`

func main() {
	addr := flag.String("addr", "http://localhost:8080", "API server address")
	username := flag.String("username", "admin", "login username")
	password := flag.String("password", "admin123", "login password")
	flag.Parse()

	c, err := gqlclient.New(*addr)
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	// ── 登录 ────────────────────────────────────────────────────────────────────
	fmt.Println("=== login ===")
	if err := c.Login(*username, *password); err != nil {
		log.Fatalf("%v\n提示：请先启动服务 go run ./cmd/server/...", err)
	}
	fmt.Println("ok")

	// ════════════════════════════════════════════════════════════════════════════
	// 写入：创建设备并一次性建立连接关系
	//
	//   router ──→ sw-01
	//     └──────→ sw-02 ──→ sw-01
	// ════════════════════════════════════════════════════════════════════════════

	// 1. 先创建 router（此时还没有连接目标，connectTo 为空）
	fmt.Println("\n=== mutation createDevice: router（无连接）===")
	resp := do(c, createDeviceMutation, map[string]any{
		"name": "example-router-01", "deviceType": "router", "mip": "10.99.0.254",
	})
	routerID := extractID(resp.Data, "createDevice", "id")

	// 2. 创建 sw-01，同时连接到 router
	fmt.Println("\n=== mutation createDevice: sw-01（connectTo: router）===")
	resp = do(c, createDeviceMutation, map[string]any{
		"name": "example-sw-01", "deviceType": "switch", "mip": "10.99.1.1",
		"connectTo": []string{routerID},
	})
	sw01ID := extractID(resp.Data, "createDevice", "id")

	// 3. 创建 sw-02，同时连接到 router 和 sw-01
	fmt.Println("\n=== mutation createDevice: sw-02（connectTo: router + sw-01）===")
	resp = do(c, createDeviceMutation, map[string]any{
		"name": "example-sw-02", "deviceType": "switch", "mip": "10.99.1.2",
		"connectTo": []string{routerID, sw01ID},
	})
	sw02ID := extractID(resp.Data, "createDevice", "id")

	// ════════════════════════════════════════════════════════════════════════════
	// 查询：单设备（含全部 connections）
	// ════════════════════════════════════════════════════════════════════════════

	fmt.Println("\n=== query device: router（含 connections）===")
	do(c, `
		query GetDevice($id: ID!) {
			device(id: $id) {
				id name type mip version createdAt updatedAt
				connections {
					target   { id name type mip }
					relation { id from_id to_id type }
				}
			}
		}
	`, map[string]any{"id": routerID})

	fmt.Println("\n=== query device: sw-01（含 connections）===")
	do(c, `
		query GetDevice($id: ID!) {
			device(id: $id) {
				id name type mip version
				connections {
					target   { id name type mip }
					relation { id from_id to_id type }
				}
			}
		}
	`, map[string]any{"id": sw01ID})

	// ════════════════════════════════════════════════════════════════════════════
	// 查询：分页列表（每个节点附带 connections）
	// ════════════════════════════════════════════════════════════════════════════

	fmt.Println("\n=== query devices（first=10，含 connections）===")
	do(c, `
		query ListDevices($first: Int) {
			devices(first: $first) {
				nodes {
					id name type mip version
					connections {
						target   { id name }
						relation { type }
					}
				}
				pageInfo { hasNextPage endCursor }
			}
		}
	`, map[string]any{"first": 10})

	// ════════════════════════════════════════════════════════════════════════════
	// 查询：拓扑图（从 router 出发，depth=2）
	// ════════════════════════════════════════════════════════════════════════════

	fmt.Println("\n=== query deviceTopology（router, depth=2）===")
	do(c, `
		query DeviceTopology($id: ID!, $depth: Int) {
			deviceTopology(id: $id, depth: $depth) {
				nodes { id labels props }
				edges { id from to type }
			}
		}
	`, map[string]any{"id": routerID, "depth": 2})

	// ════════════════════════════════════════════════════════════════════════════
	// 写入：更新设备
	// ════════════════════════════════════════════════════════════════════════════

	fmt.Println("\n=== mutation updateDevice: sw-01（改 name + mip）===")
	do(c, `
		mutation UpdateDevice($id: ID!, $input: UpdateDeviceInput!) {
			updateDevice(id: $id, input: $input) {
				id name type mip version updatedAt
			}
		}
	`, map[string]any{
		"id":    sw01ID,
		"input": map[string]any{"name": "example-sw-01-renamed", "mip": "10.99.1.100"},
	})

	// ════════════════════════════════════════════════════════════════════════════
	// 清理：软删除全部 3 台设备
	// ════════════════════════════════════════════════════════════════════════════

	deleteMutation := `mutation DeleteDevice($id: ID!) { deleteDevice(id: $id) }`

	fmt.Println("\n=== mutation deleteDevice: router ===")
	do(c, deleteMutation, map[string]any{"id": routerID})

	fmt.Println("\n=== mutation deleteDevice: sw-01 ===")
	do(c, deleteMutation, map[string]any{"id": sw01ID})

	fmt.Println("\n=== mutation deleteDevice: sw-02 ===")
	do(c, deleteMutation, map[string]any{"id": sw02ID})

	// ════════════════════════════════════════════════════════════════════════════
	// 验证：已删除设备查询返回 null
	// ════════════════════════════════════════════════════════════════════════════

	fmt.Println("\n=== query device: router（已删除，预期 data.device=null）===")
	do(c, `query { device(id: "`+routerID+`") { id name } }`, nil)
}

// do 发送请求并打印 GraphQL 响应，出错时直接 fatal。
func do(c *gqlclient.Client, query string, variables map[string]any) *gqlclient.Response {
	resp, err := c.Do(query, variables)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}
	resp.Print()
	return resp
}

// extractID 从 resp.Data 中按路径取出字符串字段（如 "createDevice" → "id"）。
func extractID(data json.RawMessage, keys ...string) string {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(data, &m)
	for i, k := range keys {
		v, ok := m[k]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			var s string
			_ = json.Unmarshal(v, &s)
			return s
		}
		_ = json.Unmarshal(v, &m)
	}
	return ""
}
