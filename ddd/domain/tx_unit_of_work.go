package domain

import (
	"context"
	"fmt"
	"sync"
)

// TxUnitOfWork 事务工作单元实现
type TxUnitOfWork struct {
	tracker         *AggregateTracker
	txManager       TransactionManager
	mu              sync.Mutex
	isExecuted      bool
	handlers        map[string][]func(ExecuteEvent) error
	isolationLevel  IsolationLevel
}

// IsolationLevel 事务隔离级别
type IsolationLevel string

const (
	// ReadUncommitted 未提交读
	ReadUncommitted IsolationLevel = "READ_UNCOMMITTED"
	// ReadCommitted 提交读
	ReadCommitted IsolationLevel = "READ_COMMITTED"
	// RepeatableRead 可重复读
	RepeatableRead IsolationLevel = "REPEATABLE_READ"
	// Serializable 可串行化（最高隔离级别）
	Serializable IsolationLevel = "SERIALIZABLE"
)

// TxUnitOfWorkConfig 配置
type TxUnitOfWorkConfig struct {
	// 隔离级别
	IsolationLevel IsolationLevel

	// 超时时间
	Timeout int64
}

// ExecuteEvent 执行事件
type ExecuteEvent struct {
	Type      ExecuteEventType
	Aggregates []AggregateRoot
	Error     error
}

// ExecuteEventType 执行事件类型
type ExecuteEventType string

const (
	EventBeforeCommit ExecuteEventType = "BEFORE_COMMIT"
	EventAfterCommit  ExecuteEventType = "AFTER_COMMIT"
	EventError        ExecuteEventType = "ERROR"
)

// NewTxUnitOfWork 创建事务工作单元
func NewTxUnitOfWork(tracker *AggregateTracker, txManager TransactionManager) *TxUnitOfWork {
	return &TxUnitOfWork{
		tracker:        tracker,
		txManager:      txManager,
		handlers:       make(map[string][]func(ExecuteEvent) error),
		isolationLevel: ReadCommitted,
	}
}

// Execute 在事务中执行业务操作
func (uow *TxUnitOfWork) Execute(fn func() error) error {
	uow.mu.Lock()
	if uow.isExecuted {
		uow.mu.Unlock()
		return fmt.Errorf("unit of work already executed")
	}
	uow.isExecuted = true
	uow.mu.Unlock()

	// 创建默认 context
	ctx := context.Background()

	// 开始事务
	if err := uow.begin(); err != nil {
		return err
	}

	// 执行操作
	var executeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				executeErr = fmt.Errorf("panic in execute: %v", r)
			}
		}()
		executeErr = fn()
	}()

	// 处理结果
	if executeErr != nil {
		uow.rollback()
		uow.emitEvent(ExecuteEvent{Type: EventError, Error: executeErr})
		return executeErr
	}

	// 保存变更
	if err := uow.saveChanges(ctx); err != nil {
		uow.rollback()
		uow.emitEvent(ExecuteEvent{Type: EventError, Error: err})
		return err
	}

	// 提交事务
	if err := uow.commit(); err != nil {
		uow.rollback()
		uow.emitEvent(ExecuteEvent{Type: EventError, Error: err})
		return err
	}

	return nil
}

// begin 开始事务
func (uow *TxUnitOfWork) begin() error {
	if uow.txManager == nil {
		return fmt.Errorf("transaction manager is not set")
	}

	return uow.txManager.Begin()
}

// saveChanges 保存变更的聚合根
func (uow *TxUnitOfWork) saveChanges(ctx context.Context) error {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	// 1. 处理新建的聚合
	if err := uow.tracker.ProcessNew(ctx); err != nil {
		return fmt.Errorf("failed to process new aggregates: %w", err)
	}

	// 2. 处理修改的聚合
	if err := uow.tracker.ProcessDirty(ctx); err != nil {
		return fmt.Errorf("failed to process dirty aggregates: %w", err)
	}

	// 3. 处理删除的聚合
	if err := uow.tracker.ProcessRemoved(ctx); err != nil {
		return fmt.Errorf("failed to process removed aggregates: %w", err)
	}

	return nil
}

// commit 提交事务
func (uow *TxUnitOfWork) commit() error {
	if uow.txManager == nil || !uow.txManager.InTransaction() {
		return fmt.Errorf("no active transaction")
	}

	return uow.txManager.Commit()
}

// rollback 回滚事务
func (uow *TxUnitOfWork) rollback() {
	if uow.txManager == nil || !uow.txManager.InTransaction() {
		return
	}

	uow.txManager.Rollback()
	uow.tracker.Clear()
}

// RegisterNew 注册新建的聚合根
func (uow *TxUnitOfWork) RegisterNew(aggregate AggregateRoot) {
	uow.tracker.RegisterNew(aggregate)
}

// RegisterDirty 注册被修改的聚合根
func (uow *TxUnitOfWork) RegisterDirty(aggregate AggregateRoot) {
	uow.tracker.RegisterDirty(aggregate)
}

// RegisterClean 注册干净的聚合根
func (uow *TxUnitOfWork) RegisterClean(aggregate AggregateRoot) {
	uow.tracker.RegisterClean(aggregate)
}

// RegisterRemoved 注册被删除的聚合根
func (uow *TxUnitOfWork) RegisterRemoved(aggregate AggregateRoot) {
	uow.tracker.RegisterRemoved(aggregate)
}

// emitEvent 发送事件
func (uow *TxUnitOfWork) emitEvent(evt ExecuteEvent) {
	uow.mu.Lock()
	defer uow.mu.Unlock()

	handlers, exists := uow.handlers[string(evt.Type)]
	if !exists {
		return
	}

	for _, handler := range handlers {
		go handler(evt)
	}
}

// GetTracker 获取聚合根跟踪器
func (uow *TxUnitOfWork) GetTracker() *AggregateTracker {
	return uow.tracker
}

// SetIsolationLevel 设置隔离级别
func (uow *TxUnitOfWork) SetIsolationLevel(level IsolationLevel) {
	uow.mu.Lock()
	defer uow.mu.Unlock()
	uow.isolationLevel = level
}

// GetIsolationLevel 获取隔离级别
func (uow *TxUnitOfWork) GetIsolationLevel() IsolationLevel {
	uow.mu.Lock()
	defer uow.mu.Unlock()
	return uow.isolationLevel
}

// MockTxUnitOfWorkFactory Mock工作单元工厂
type MockTxUnitOfWorkFactory struct {
	tracker   *AggregateTracker
	txManager TransactionManager
}

// NewMockTxUnitOfWorkFactory 创建Mock工作单元工厂
func NewMockTxUnitOfWorkFactory(userRepo UserRepository, orderRepo OrderRepository, publisher DomainEventPublisher) *MockTxUnitOfWorkFactory {
	tracker := NewAggregateTracker(userRepo, orderRepo, publisher)
	return &MockTxUnitOfWorkFactory{
		tracker:   tracker,
		txManager: nil, // Mock不需要真实事务管理器
	}
}

// New 创建Mock工作单元
func (f *MockTxUnitOfWorkFactory) New() UnitOfWork {
	return NewTxUnitOfWork(f.tracker, f.txManager)
}
