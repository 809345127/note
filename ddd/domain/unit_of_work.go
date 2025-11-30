package domain

import (
	"context"
	"fmt"
	"sync"
)

// UnitOfWork 工作单元接口
// DDD原则：
// 1. 跟踪聚合根的变化
// 2. 管理事务边界
// 3. 协调仓储的保存操作
// 4. 保证聚合的一致性
//
// 使用模式：
// uow := unitOfWorkFactory.New()
//
//	err := uow.Execute(func() error {
//	    // 加载聚合根
//	    user, _ := userRepo.FindByID(userID)
//	    order, _ := orderRepo.FindByID(orderID)
//
//	    // 执行业务操作
//	    user.Deactivate()
//	    order.Cancel()
//
//	    // 保存（在工作单元执行时自动处理）
//	    uow.RegisterDirty(user)
//	    uow.RegisterDirty(order)
//
//	    return nil
//	})
type UnitOfWork interface {
	// Execute 在事务中执行业务操作
	// 自动处理begin、commit和rollback
	Execute(fn func() error) error

	// RegisterNew 注册新建的聚合根
	RegisterNew(aggregate AggregateRoot)

	// RegisterDirty 注册被修改的聚合根
	RegisterDirty(aggregate AggregateRoot)

	// RegisterClean 注册干净的聚合根（未改变）
	RegisterClean(aggregate AggregateRoot)

	// RegisterRemoved 注册被删除的聚合根
	RegisterRemoved(aggregate AggregateRoot)
}

// UnitOfWorkFactory 工作单元工厂
type UnitOfWorkFactory interface {
	// New 创建新的工作单元
	New() UnitOfWork
}

// TransactionManager 事务管理器接口
type TransactionManager interface {
	// Begin 开始事务
	Begin() error

	// Commit 提交事务
	Commit() error

	// Rollback 回滚事务
	Rollback() error

	// InTransaction 是否处于事务中
	InTransaction() bool
}

// OutboxRepository Outbox 仓储接口
// 用于保存领域事件到 outbox 表，与业务数据同事务提交
type OutboxRepository interface {
	// SaveEvent 保存事件到 outbox 表（在当前事务中）
	SaveEvent(ctx context.Context, event DomainEvent) error
}

// AggregateTracker 聚合根跟踪器
// 职责：跟踪聚合根变更，保存到仓储，收集事件到 outbox
// 注意：不直接发布事件，事件由后台 OutboxProcessor 异步发布
type AggregateTracker struct {
	mu         sync.RWMutex
	new        map[string]AggregateRoot // 新建的聚合
	dirty      map[string]AggregateRoot // 修改的聚合
	removed    map[string]AggregateRoot // 删除的聚合
	clean      map[string]AggregateRoot // 干净的聚合
	userRepo   UserRepository
	orderRepo  OrderRepository
	outboxRepo OutboxRepository // 用于保存事件到 outbox 表
}

// NewAggregateTracker 创建聚合根跟踪器
func NewAggregateTracker(userRepo UserRepository, orderRepo OrderRepository, outboxRepo OutboxRepository) *AggregateTracker {
	return &AggregateTracker{
		new:        make(map[string]AggregateRoot),
		dirty:      make(map[string]AggregateRoot),
		removed:    make(map[string]AggregateRoot),
		clean:      make(map[string]AggregateRoot),
		userRepo:   userRepo,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
	}
}

// RegisterNew 注册新建的聚合根
func (t *AggregateTracker) RegisterNew(aggregate AggregateRoot) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.getAggregateKey(aggregate)
	t.new[key] = aggregate
	// 从其他类别中移除
	delete(t.dirty, key)
	delete(t.removed, key)
	delete(t.clean, key)
}

// RegisterDirty 注册被修改的聚合根
func (t *AggregateTracker) RegisterDirty(aggregate AggregateRoot) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.getAggregateKey(aggregate)

	// 如果已经是新建状态，不需要再标记为修改
	if _, exists := t.new[key]; exists {
		return
	}

	t.dirty[key] = aggregate
	delete(t.removed, key)
	delete(t.clean, key)
}

// RegisterClean 注册干净的聚合根
func (t *AggregateTracker) RegisterClean(aggregate AggregateRoot) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.getAggregateKey(aggregate)

	// 如果不在其他状态，才标记为干净
	if _, exists := t.new[key]; !exists {
		if _, exists := t.dirty[key]; !exists {
			if _, exists := t.removed[key]; !exists {
				t.clean[key] = aggregate
			}
		}
	}
}

// RegisterRemoved 注册被删除的聚合根
func (t *AggregateTracker) RegisterRemoved(aggregate AggregateRoot) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.getAggregateKey(aggregate)

	// 从新建或修改状态转移到删除状态
	delete(t.new, key)
	delete(t.dirty, key)
	delete(t.clean, key)
	t.removed[key] = aggregate
}

// getAggregateKey 生成聚合根的唯一键
// key格式: <aggregateType>:<aggregateID>
func (t *AggregateTracker) getAggregateKey(aggregate AggregateRoot) string {
	// 根据具体类型判断
	switch agg := aggregate.(type) {
	case *User:
		return fmt.Sprintf("user:%s", agg.ID())
	case *Order:
		return fmt.Sprintf("order:%s", agg.ID())
	default:
		// 通用处理
		return fmt.Sprintf("aggregate:%s", aggregate.ID())
	}
}

// ProcessNew 处理新建的聚合根
func (t *AggregateTracker) ProcessNew(ctx context.Context) error {
	for key, aggregate := range t.new {
		if err := t.saveAggregate(ctx, aggregate); err != nil {
			return fmt.Errorf("failed to save new aggregate %s: %w", key, err)
		}
	}
	return nil
}

// ProcessDirty 处理修改的聚合根
func (t *AggregateTracker) ProcessDirty(ctx context.Context) error {
	for key, aggregate := range t.dirty {
		if err := t.saveAggregate(ctx, aggregate); err != nil {
			return fmt.Errorf("failed to update aggregate %s: %w", key, err)
		}
	}
	return nil
}

// ProcessRemoved 处理删除的聚合根
func (t *AggregateTracker) ProcessRemoved(ctx context.Context) error {
	for key, aggregate := range t.removed {
		if err := t.removeAggregate(ctx, aggregate); err != nil {
			return fmt.Errorf("failed to remove aggregate %s: %w", key, err)
		}
	}
	return nil
}

// saveAggregate 保存聚合根并将事件保存到 outbox 表
// 注意：不直接发布事件，事件由后台 OutboxProcessor 异步发布
func (t *AggregateTracker) saveAggregate(ctx context.Context, aggregate AggregateRoot) error {
	// 根据聚合根的类型调用对应的仓储
	// 这里使用类型断言检查聚合根的类型

	// 尝试作为User类型
	if user, ok := aggregate.(*User); ok {
		if err := t.userRepo.Save(ctx, user); err != nil {
			return err
		}
		// 保存事件到 outbox 表（同一事务）
		events := user.PullEvents()
		if err := t.saveEventsToOutbox(ctx, events); err != nil {
			return err
		}
		return nil
	}

	// 尝试作为Order类型
	if order, ok := aggregate.(*Order); ok {
		if err := t.orderRepo.Save(ctx, order); err != nil {
			return err
		}
		// 保存事件到 outbox 表（同一事务）
		events := order.PullEvents()
		if err := t.saveEventsToOutbox(ctx, events); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("unsupported aggregate type: %T", aggregate)
}

// removeAggregate 删除聚合根
func (t *AggregateTracker) removeAggregate(ctx context.Context, aggregate AggregateRoot) error {
	id := aggregate.ID()

	// 根据类型调用对应的删除方法
	switch aggregate.(type) {
	case *User:
		return t.userRepo.Remove(ctx, id)
	case *Order:
		return t.orderRepo.Remove(ctx, id)
	default:
		return fmt.Errorf("unsupported aggregate type: %T", aggregate)
	}
}

// saveEventsToOutbox 保存事件到 outbox 表
// 事件与业务数据在同一事务中提交，保证原子性
// 后台 OutboxProcessor 会异步读取 outbox 表并发布到消息队列
func (t *AggregateTracker) saveEventsToOutbox(ctx context.Context, events []DomainEvent) error {
	if t.outboxRepo == nil {
		// 如果没有配置 outbox 仓储，仅打印日志（开发/测试环境）
		for _, event := range events {
			fmt.Printf("[OUTBOX] Would save event: %s for aggregate %s\n", event.EventName(), event.GetAggregateID())
		}
		return nil
	}

	for _, event := range events {
		if err := t.outboxRepo.SaveEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to save event %s to outbox: %w", event.EventName(), err)
		}
	}
	return nil
}

// Clear 清空跟踪器（应该在事务成功提交后调用）
func (t *AggregateTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.new = make(map[string]AggregateRoot)
	t.dirty = make(map[string]AggregateRoot)
	t.removed = make(map[string]AggregateRoot)
	t.clean = make(map[string]AggregateRoot)
}

// IsEmpty 检查跟踪器是否为空
func (t *AggregateTracker) IsEmpty() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.new) == 0 && len(t.dirty) == 0 && len(t.removed) == 0
}

// GetStats 获取跟踪器统计信息
func (t *AggregateTracker) GetStats() map[string]int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]int{
		"new":     len(t.new),
		"dirty":   len(t.dirty),
		"removed": len(t.removed),
		"clean":   len(t.clean),
	}
}
