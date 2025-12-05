package mocks

import (
	"context"
	"errors"
	"sync"

	"ddd-example/domain/user"

	"github.com/google/uuid"
)

// MockUserRepository 用户仓储的Mock实现
// DDD原则：仓储只负责聚合根的持久化，不负责发布事件
// 事件发布由 UoW 保存到 outbox 表，后台服务异步发布
type MockUserRepository struct {
	users map[string]*user.User
	mu    sync.RWMutex
}

// NewMockUserRepository 创建Mock用户仓储
func NewMockUserRepository() *MockUserRepository {
	repo := &MockUserRepository{
		users: make(map[string]*user.User),
	}

	// 初始化一些测试数据
	repo.initializeTestData()
	return repo
}

// initializeTestData 初始化测试数据
func (r *MockUserRepository) initializeTestData() {
	// 创建测试用户
	user1, err1 := user.NewUser("张三", "zhangsan@example.com", 25)
	user2, err2 := user.NewUser("李四", "lisi@example.com", 30)
	user3, err3 := user.NewUser("王五", "wangwu@example.com", 35)

	if err1 == nil && err2 == nil && err3 == nil {
		// 使用用户的实际ID作为key（而不是硬编码的key）
		r.users[user1.ID()] = user1
		r.users[user2.ID()] = user2
		r.users[user3.ID()] = user3
	}
}

// NextIdentity 生成新的用户ID
// DDD原则：ID生成策略由仓储控制，便于统一管理和测试
func (r *MockUserRepository) NextIdentity() string {
	return "user-" + uuid.New().String()
}

func (r *MockUserRepository) Save(ctx context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[u.ID()] = u

	// 注意：不在仓储中发布事件！
	// 事件由 UoW 在事务提交前保存到 outbox 表
	// 后台 OutboxProcessor 异步发布到消息队列

	return nil
}

func (r *MockUserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (r *MockUserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.Email().Value() == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

// Remove 逻辑删除用户（DDD推荐做法）
func (r *MockUserRepository) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, exists := r.users[id]
	if !exists {
		return errors.New("user not found")
	}

	// 逻辑删除：标记为不活跃
	u.Deactivate()
	return nil
}
