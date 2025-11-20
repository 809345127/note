package mock

import (
	"ddd-example/domain"
	"errors"
	"sync"
)

// MockUserRepository 用户仓储的Mock实现
type MockUserRepository struct {
	users map[string]*domain.User
	mu    sync.RWMutex
}

// NewMockUserRepository 创建Mock用户仓储
func NewMockUserRepository() *MockUserRepository {
	repo := &MockUserRepository{
		users: make(map[string]*domain.User),
	}
	
	// 初始化一些测试数据
	repo.initializeTestData()
	return repo
}

// initializeTestData 初始化测试数据
func (r *MockUserRepository) initializeTestData() {
	// 创建测试用户
	user1, _ := domain.NewUser("张三", "zhangsan@example.com", 25)
	user2, _ := domain.NewUser("李四", "lisi@example.com", 30)
	user3, _ := domain.NewUser("王五", "wangwu@example.com", 35)
	
	// 设置固定ID以便测试
	r.users["user-1"] = user1
	r.users["user-2"] = user2
	r.users["user-3"] = user3
}

func (r *MockUserRepository) Save(user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.users[user.ID()] = user
	return nil
}

func (r *MockUserRepository) FindByID(id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	user, exists := r.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (r *MockUserRepository) FindByEmail(email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, user := range r.users {
		if user.Email().Value() == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *MockUserRepository) FindAll() ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	users := make([]*domain.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

func (r *MockUserRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.users, id)
	return nil
}