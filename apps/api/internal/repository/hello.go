package repository

import "fmt"

// HelloRepository 问候数据访问接口
type HelloRepository interface {
	GetGreeting(name string) string
}

type helloRepo struct{}

func newHelloRepo() HelloRepository {
	return &helloRepo{}
}

func (r *helloRepo) GetGreeting(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}
