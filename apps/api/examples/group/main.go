// Group 操作示例：演示 GroupGraphRepo + UserGraphRepo 内存存根的完整流程。
//
// 用法（在 apps/api 目录下执行）：
//
//	go run ./examples/group/...
//
// 此示例使用内存存根，无需 Neo4j 连接，可直接运行。
// Neo4j 版 GroupGraphRepo 待实现后，将示例切换为 Neo4j 实现。
package main

import (
	"context"
	"fmt"
	"log"

	"matrix/api/internal/repository/neo4j_repo"
)

func main() {
	ctx := context.Background()

	userRepo := neo4j_repo.NewUserGraphRepo()
	groupRepo := neo4j_repo.NewGroupGraphRepo(userRepo)

	// ── 1. 创建用户 ────────────────────────────────────────────────────────────
	fmt.Println("=== 1. CreateUser ===")
	alice, err := userRepo.CreateUser(ctx, "Alice", "ou_alice_001")
	if err != nil {
		log.Fatalf("CreateUser alice: %v", err)
	}
	bob, err := userRepo.CreateUser(ctx, "Bob", "ou_bob_002")
	if err != nil {
		log.Fatalf("CreateUser bob: %v", err)
	}
	fmt.Printf("created: id=%s name=%s\n", alice.ID, alice.Name)
	fmt.Printf("created: id=%s name=%s\n", bob.ID, bob.Name)

	// ── 2. 查询用户 ────────────────────────────────────────────────────────────
	fmt.Println("\n=== 2. GetUser ===")
	got, err := userRepo.GetUser(ctx, alice.ID)
	if err != nil {
		log.Fatalf("GetUser: %v", err)
	}
	fmt.Printf("got: id=%s name=%s feishuID=%s\n", got.ID, got.Name, got.FeishuID)

	// ── 3. 列出所有用户 ────────────────────────────────────────────────────────
	fmt.Println("\n=== 3. ListUsers（first=10）===")
	users, hasNext, _, err := userRepo.ListUsers(ctx, 10, "")
	if err != nil {
		log.Fatalf("ListUsers: %v", err)
	}
	for _, u := range users {
		fmt.Printf("  - id=%-10s  name=%s\n", u.ID, u.Name)
	}
	fmt.Printf("hasNext=%v\n", hasNext)

	// ── 4. 创建组 ──────────────────────────────────────────────────────────────
	fmt.Println("\n=== 4. CreateGroup ===")
	ops, err := groupRepo.GreateGroup(ctx, "网络运维组")
	if err != nil {
		log.Fatalf("GreateGroup ops: %v", err)
	}
	security, err := groupRepo.GreateGroup(ctx, "安全组")
	if err != nil {
		log.Fatalf("GreateGroup security: %v", err)
	}
	fmt.Printf("created: id=%s name=%s\n", ops.ID, ops.Name)
	fmt.Printf("created: id=%s name=%s\n", security.ID, security.Name)

	// ── 5. 查询单个组 ──────────────────────────────────────────────────────────
	fmt.Println("\n=== 5. GetGroup ===")
	g, err := groupRepo.GetGroup(ctx, ops.ID)
	if err != nil {
		log.Fatalf("GetGroup: %v", err)
	}
	fmt.Printf("got: id=%s name=%s\n", g.ID, g.Name)

	// ── 6. 列出所有组 ──────────────────────────────────────────────────────────
	fmt.Println("\n=== 6. ListGroups（first=10）===")
	groups, _, _, err := groupRepo.ListGroups(ctx, 10, "")
	if err != nil {
		log.Fatalf("ListGroups: %v", err)
	}
	for _, gr := range groups {
		fmt.Printf("  - id=%-8s  name=%s\n", gr.ID, gr.Name)
	}

	// ── 7. 将用户加入组 ────────────────────────────────────────────────────────
	fmt.Println("\n=== 7. AddUserToGroup ===")
	if err := groupRepo.AddUserToGroup(ctx, alice.ID, ops.ID); err != nil {
		log.Fatalf("AddUserToGroup alice->ops: %v", err)
	}
	if err := groupRepo.AddUserToGroup(ctx, bob.ID, ops.ID); err != nil {
		log.Fatalf("AddUserToGroup bob->ops: %v", err)
	}
	if err := groupRepo.AddUserToGroup(ctx, alice.ID, security.ID); err != nil {
		log.Fatalf("AddUserToGroup alice->security: %v", err)
	}
	fmt.Println("alice → 网络运维组, 安全组")
	fmt.Println("bob   → 网络运维组")

	// ── 8. 查询用户所属的所有组 ────────────────────────────────────────────────
	fmt.Println("\n=== 8. GetUserGroups（alice）===")
	aliceGroups, err := groupRepo.GetUserGroups(ctx, alice.ID)
	if err != nil {
		log.Fatalf("GetUserGroups: %v", err)
	}
	for _, link := range aliceGroups {
		fmt.Printf("  user=%s  group=%s  rel=%s\n", link.User.Name, link.Group.Name, link.Relation.Type)
	}

	// ── 9. 查询组内所有成员 ────────────────────────────────────────────────────
	fmt.Println("\n=== 9. GetGroupMembers（网络运维组）===")
	members, err := groupRepo.GetGroupMembers(ctx, ops.ID)
	if err != nil {
		log.Fatalf("GetGroupMembers: %v", err)
	}
	for _, link := range members {
		fmt.Printf("  user=%s  group=%s  rel=%s\n", link.User.Name, link.Group.Name, link.Relation.Type)
	}

	// ── 10. 组与组的父子关系 ───────────────────────────────────────────────────
	fmt.Println("\n=== 10. AddGroupToGroup + GetGroupChildren ===")
	parent, err := groupRepo.GreateGroup(ctx, "IT部门")
	if err != nil {
		log.Fatalf("GreateGroup parent: %v", err)
	}
	if err := groupRepo.AddGroupToGroup(ctx, parent.ID, ops.ID); err != nil {
		log.Fatalf("AddGroupToGroup: %v", err)
	}
	if err := groupRepo.AddGroupToGroup(ctx, parent.ID, security.ID); err != nil {
		log.Fatalf("AddGroupToGroup: %v", err)
	}

	children, err := groupRepo.GetGroupChildren(ctx, parent.ID)
	if err != nil {
		log.Fatalf("GetGroupChildren: %v", err)
	}
	fmt.Printf("IT部门 的子组（%d 个）：\n", len(children))
	for _, link := range children {
		fmt.Printf("  parent=%s  child=%s  rel=%s\n", parent.Name, link.Target.Name, link.Relation.Type)
	}
}
