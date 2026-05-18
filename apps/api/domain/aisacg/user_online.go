package aisacg

import "context"

// UserOnlineFetcher 亚信 ACG 在线用户查询能力。
type UserOnlineFetcher interface {
	// GetOnlineTotal 获取在线用户总数统计。
	GetOnlineTotal(ctx context.Context) (*OnlineTotal, error)

	// ListOnlineUsers 根据组织 path 获取在线用户明细。
	ListOnlineUsers(ctx context.Context, path string) ([]UserOnline, error)

	// GetOnlineTree 获取在线用户顶层组织树。
	GetOnlineTree(ctx context.Context) ([]OnlineTreeNode, error)

	// GetOnlineTreeByPath 按 path 查询在线用户组织树。
	GetOnlineTreeByPath(ctx context.Context, path string) ([]OnlineTreeNode, error)
}
