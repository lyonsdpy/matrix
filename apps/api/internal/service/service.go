package service

import "matrix/api/internal/repository"

// Services holds all business-logic instances.
// Add concrete service fields here as the project grows.
type Services struct {
	repos *repository.Repositories
}

func New(repos *repository.Repositories) *Services {
	return &Services{repos: repos}
}
