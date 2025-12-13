package inject

import (
	"fmt"
	"log"
)

// User 结构体
type User struct {
	ID   int
	Name string
}

// GetNewUser 构造函数
func GetNewUser() *User {
	return &User{
		ID:   1,
		Name: "John Doe",
	}
}

// Service 依赖于User的服务
type Service struct {
	user *User
}

// NewService Service的构造函数，依赖于User
func NewService(user *User) *Service {
	return &Service{
		user: user,
	}
}

// DoSomething Service的方法
func (s *Service) DoSomething() {
	fmt.Printf("Service is doing something with user: %s (ID: %d)\n", s.user.Name, s.user.ID)
}

func Example() {
	// 注册User构造函数
	// err := inject.Register(GetNewUser)
	err := Provide(func() *User {
		return &User{
			ID:   1,
			Name: "John Doe",
		}
	})
	if err != nil {
		log.Fatalf("Failed to register User constructor: %v", err)
	}

	// 注册Service构造函数
	err = Provide(NewService)
	if err != nil {
		log.Fatalf("Failed to register Service constructor: %v", err)
	}

	// 解析User实例
	var user *User
	err = Resolve(&user)
	if err != nil {
		log.Fatalf("Failed to resolve User: %v", err)
	}
	fmt.Printf("Resolved User: %+v\n", user)

	// 解析Service实例
	var service *Service
	err = Resolve(&service)
	if err != nil {
		log.Fatalf("Failed to resolve Service: %v", err)
	}

	// 使用Service
	service.DoSomething()

	// 也可以使用Invoke直接调用函数
	err = GetContainer().Invoke(func(s *Service) {
		fmt.Println("Using Invoke with Service:")
		s.DoSomething()
	})
	if err != nil {
		log.Fatalf("Failed to invoke function: %v", err)
	}
}
