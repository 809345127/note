package mocks

import (
	"context"
	"ddd-example/domain"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
)

// MockUserRepository 用户仓储的Mock实现
// DDD原则：仓储负责聚合根的持久化，并在保存后发布领域事件
type MockUserRepository struct {
	users          map[string]*domain.User
	mu             sync.RWMutex
	eventPublisher domain.DomainEventPublisher
}

// NewMockUserRepository 创建Mock用户仓储
// eventPublisher: 事件发布器，仓储在Save后发布聚合根产生的事件
func NewMockUserRepository(eventPublisher domain.DomainEventPublisher) *MockUserRepository {
	repo := &MockUserRepository{
		users:          make(map[string]*domain.User),
		eventPublisher: eventPublisher,
	}

	// 初始化一些测试数据
	repo.initializeTestData()
	return repo
}

// initializeTestData 初始化测试数据
func (r *MockUserRepository) initializeTestData() {
	// 创建测试用户
	user1, err1 := domain.NewUser("张三", "zhangsan@example.com", 25)
	user2, err2 := domain.NewUser("李四", "lisi@example.com", 30)
	user3, err3 := domain.NewUser("王五", "wangwu@example.com", 35)

	if err1 == nil && err2 == nil && err3 == nil {
		// 使用固定ID以便测试
		// 在实际应用中，应该将创建的实体保存到仓储中
		// 这里直接使用预定义的ID
		r.users["user-1"] = user1
		r.users["user-2"] = user2
		r.users["user-3"] = user3
	}
}

// NextIdentity 生成新的用户ID
// DDD原则：ID生成策略由仓储控制，便于统一管理和测试
func (r *MockUserRepository) NextIdentity() string {
	return "user-" + uuid.New().String()
}

func (r *MockUserRepository) Save(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID()] = user

	// DDD原则：仓储在保存成功后发布聚合根产生的领域事件
	r.publishEvents(user.PullEvents())

	return nil
}

// publishEvents 发布领域事件
func (r *MockUserRepository) publishEvents(events []domain.DomainEvent) {
	if r.eventPublisher == nil {
		return
	}
	for _, event := range events {
		if err := r.eventPublisher.Publish(event); err != nil {
			log.Printf("[WARN] Failed to publish event %s: %v", event.EventName(), err)
		}
	}
}

func (r *MockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (r *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email().Value() == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

// Remove 逻辑删除用户（DDD推荐做法）
func (r *MockUserRepository) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return errors.New("user not found")
	}

	// 逻辑删除：标记为不活跃
	user.Deactivate()
	return nil
}