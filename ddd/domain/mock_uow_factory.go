package domain

// MockUnitOfWorkFactory Mock工作单元工厂（用于测试和简单场景）
type MockUnitOfWorkFactory struct {
	userRepo   UserRepository
	orderRepo  OrderRepository
	publisher  DomainEventPublisher
}

// NewMockUnitOfWorkFactory 创建Mock工作单元工厂
func NewMockUnitOfWorkFactory(userRepo UserRepository, orderRepo OrderRepository, publisher DomainEventPublisher) *MockUnitOfWorkFactory {
	return &MockUnitOfWorkFactory{
		userRepo:  userRepo,
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

// New 创建Mock工作单元
// Mock工作单元不管理真实事务，直接执行操作
func (f *MockUnitOfWorkFactory) New() UnitOfWork {
	// 创建跟踪器
	tracker := NewAggregateTracker(f.userRepo, f.orderRepo, f.publisher)

	// 返回无事务管理器的工作单元
	return NewTxUnitOfWork(tracker, nil)
}

// SimpleUnitOfWorkFactory 简单工作单元工厂
type SimpleUnitOfWorkFactory struct {
	unitOfWork UnitOfWork
}

// NewSimpleUnitOfWorkFactory 创建简单工作单元工厂
func NewSimpleUnitOfWorkFactory(uow UnitOfWork) *SimpleUnitOfWorkFactory {
	return &SimpleUnitOfWorkFactory{unitOfWork: uow}
}

// New 返回预配置的工作单元
func (f *SimpleUnitOfWorkFactory) New() UnitOfWork {
	return f.unitOfWork
}
