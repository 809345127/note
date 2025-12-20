package mocks

import (
	"context"
	"errors"
	"sync"

	"ddd/domain/shared"
	"ddd/domain/user"

	"github.com/google/uuid"
)

// MockUserRepository Mock implementation of user repository
// DDD principle: Repository is only responsible for persistence of aggregate roots, not event publishing
// Events are saved to outbox table by UoW and published asynchronously by background services
type MockUserRepository struct {
	users map[string]*user.User
	mu    sync.RWMutex
}

// NewMockUserRepository Create Mock user repository
func NewMockUserRepository() *MockUserRepository {
	repo := &MockUserRepository{
		users: make(map[string]*user.User),
	}

	// Initialize some test data
	repo.initializeTestData()
	return repo
}

// initializeTestData Initialize test data
func (r *MockUserRepository) initializeTestData() {
	// Create test users
	user1, err1 := user.NewUser("Zhang San", "zhangsan@example.com", 25)
	user2, err2 := user.NewUser("Li Si", "lisi@example.com", 30)
	user3, err3 := user.NewUser("Wang Wu", "wangwu@example.com", 35)

	if err1 == nil && err2 == nil && err3 == nil {
		// Use the actual user ID as key (instead of hardcoded key)
		r.users[user1.ID()] = user1
		r.users[user2.ID()] = user2
		r.users[user3.ID()] = user3
	}
}

// NextIdentity Generate new user ID
// DDD principle: ID generation strategy is controlled by repository for unified management and testing
func (r *MockUserRepository) NextIdentity() string {
	return "user-" + uuid.New().String()
}

func (r *MockUserRepository) Save(ctx context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[u.ID()] = u

	// Note: Do not publish events in repository!
	// Events are saved to outbox table by UoW before transaction commit
	// Background OutboxProcessor publishes to message queue asynchronously

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
	spec := user.ByEmailSpecification{Email: email}
	users, err := r.FindBySpecification(ctx, spec)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, errors.New("user not found")
	}
	return users[0], nil
}

func (r *MockUserRepository) FindBySpecification(ctx context.Context, spec shared.Specification[*user.User]) ([]*user.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var users []*user.User
	for _, u := range r.users {
		if spec.IsSatisfiedBy(ctx, u) {
			users = append(users, u)
		}
	}
	return users, nil
}

// Remove Logical deletion of user (DDD recommended practice)
func (r *MockUserRepository) Remove(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, exists := r.users[id]
	if !exists {
		return errors.New("user not found")
	}

	// Logical deletion: mark as inactive
	u.Deactivate()
	return nil
}
