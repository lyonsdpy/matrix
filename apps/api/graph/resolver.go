package graph

// 此文件由开发者手写，gqlgen generate 只在文件不存在时创建，不会覆盖。
// Resolver 是 GraphQL 层的依赖根，持有 repo 引用供所有子 resolver 使用。
// DataLoader 不在此处持有——它是 per-request 的，通过 context 传递。

import "matrix/api/internal/repository"

type Resolver struct {
	repo *repository.DeviceRepo
}

func NewResolver(repo *repository.DeviceRepo) *Resolver {
	return &Resolver{repo: repo}
}
