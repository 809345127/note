package domain

// MockUnitOfWorkFactory Mock工作单元工厂（用于测试和简单场景）
// 注意：事件不直接发布，而是保存到 outbox（或打印日志）
type MockUnitOfWorkFactory struct {
	userRepo   UserRepository
	orderRepo  OrderRepository
	outboxRepo OutboxRepository
}

// NewMockUnitOfWorkFactory 创建Mock工作单元工厂
// outboxRepo: 可以传 nil，表示不持久化事件（仅打印日志）
func NewMockUnitOfWorkFactory(userRepo UserRepository, orderRepo OrderRepository, outboxRepo OutboxRepository) *MockUnitOfWorkFactory {
	return &MockUnitOfWorkFactory{
		userRepo:   userRepo,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
	}
}

// New 创建Mock工作单元
// Mock工作单元不管理真实事务，直接执行操作
// 事件会保存到 outbox（如果配置了），由后台服务异步发布
func (f *MockUnitOfWorkFactory) New() UnitOfWork {
	// 创建跟踪器
	tracker := NewAggregateTracker(f.userRepo, f.orderRepo, f.outboxRepo)

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
