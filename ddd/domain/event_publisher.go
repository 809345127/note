package domain

import (
	"fmt"
	"sync"
	"time"
)

// DomainEventPublisher 领域事件发布器接口
// DDD原则：领域层定义接口，基础设施层提供实现
// 应用层通过接口发布事件，不依赖具体实现
type DomainEventPublisher interface {
	// Publish 发布领域事件
	Publish(event DomainEvent) error

	// Subscribe 订阅特定类型的事件
	Subscribe(eventName string, handler EventHandler) error

	// Unsubscribe 取消订阅
	Unsubscribe(eventName string, handler EventHandler) error
}

// EventHandler 事件处理器接口
type EventHandler interface {
	// Handle 处理事件
	Handle(event DomainEvent) error

	// Name 返回处理器名称，用于标识
	Name() string
}

// EventHandlerFunc 函数式事件处理器类型
type EventHandlerFunc func(event DomainEvent) error

// EventPublishOptions 事件发布选项
type EventPublishOptions struct {
	// Retry 失败时重试次数
	Retry int

	// Timeout 发布超时时间
	Timeout time.Duration
}

// EventSubscription 事件订阅
type EventSubscription struct {
	ID        string          `json:"id"`
	EventName string          `json:"event_name"`
	Handler   EventHandler    `json:"-"`
	CreatedAt time.Time       `json:"created_at"`
	IsActive  bool            `json:"is_active"`
}

// EventPublishResult 事件发布结果
type EventPublishResult struct {
	EventName string `json:"event_name"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

// ValidateEvent 验证领域事件
func ValidateEvent(event DomainEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if event.EventName() == "" {
		return fmt.Errorf("event name cannot be empty")
	}

	aggregateID := event.GetAggregateID()
	if aggregateID == "" {
		return fmt.Errorf("aggregate ID cannot be empty")
	}

	occurredOn := event.OccurredOn()
	if occurredOn.IsZero() {
		return fmt.Errorf("occurred on time cannot be zero")
	}

	return nil
}

// EventBus 内存事件总线实现
type EventBus struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex

	// 发布历史（调试用）
	history []EventPublishResult
	muHistory sync.Mutex
}

// NewEventBus 创建新的事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
		history:  make([]EventPublishResult, 0),
	}
}

// Publish 发布事件（同步执行）
func (bus *EventBus) Publish(event DomainEvent) error {
	if err := ValidateEvent(event); err != nil {
		return err
	}

	bus.mu.RLock()
	handlers, exists := bus.handlers[event.EventName()]
	bus.mu.RUnlock()

	result := EventPublishResult{
		EventName:   event.EventName(),
		Success:     true,
		PublishedAt: time.Now(),
	}

	if exists && len(handlers) > 0 {
		// 执行所有处理器（同步）
		// 生产环境应该考虑使用异步处理
		for _, handler := range handlers {
			if err := handler.Handle(event); err != nil {
				result.Success = false
				result.Message = fmt.Sprintf("handler %s failed: %v", handler.Name(), err)
				// 继续执行其他处理器
			}
		}
	} else {
		result.Message = "no handlers registered for this event"
	}

	// 记录发布历史
	bus.muHistory.Lock()
	bus.history = append(bus.history, result)
	if len(bus.history) > 1000 {
		// 限制历史记录长度
		bus.history = bus.history[len(bus.history)-1000:]
	}
	bus.muHistory.Unlock()

	return nil
}

// Subscribe 订阅事件
func (bus *EventBus) Subscribe(eventName string, handler EventHandler) error {
	if eventName == "" {
		return fmt.Errorf("event name cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	// 检查是否已经订阅
	for _, h := range bus.handlers[eventName] {
		if h.Name() == handler.Name() {
			return fmt.Errorf("handler %s already subscribed to %s", handler.Name(), eventName)
		}
	}

	bus.handlers[eventName] = append(bus.handlers[eventName], handler)
	return nil
}

// Unsubscribe 取消订阅
func (bus *EventBus) Unsubscribe(eventName string, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers, exists := bus.handlers[eventName]
	if !exists {
		return nil
	}

	// 移除处理器
	for i, h := range handlers {
		if h.Name() == handler.Name() {
			bus.handlers[eventName] = append(handlers[:i], handlers[i+1:]...)
			return nil
		}
	}

	return nil
}

// GetPublishHistory 获取发布历史（调试用）
func (bus *EventBus) GetPublishHistory() []EventPublishResult {
	bus.muHistory.Lock()
	defer bus.muHistory.Unlock()

	history := make([]EventPublishResult, len(bus.history))
	copy(history, bus.history)
	return history
}

// FuncHandler 函数式事件处理器
type FuncHandler struct {
	name string
	fn   func(DomainEvent) error
}

// NewFuncHandler 创建函数式处理器
func NewFuncHandler(name string, fn func(DomainEvent) error) *FuncHandler {
	if name == "" {
		name = fmt.Sprintf("func-handler-%d", time.Now().UnixNano())
	}
	return &FuncHandler{
		name: name,
		fn:   fn,
	}
}

// Handle 处理事件
func (h *FuncHandler) Handle(event DomainEvent) error {
	return h.fn(event)
}

// Name 返回处理器名称
func (h *FuncHandler) Name() string {
	return h.name
}
// MockEventPublisher Mock事件发布器
type MockEventPublisher struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewMockEventPublisher 创建Mock事件发布器
func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		handlers: make(map[string][]EventHandler),
	}
}

func (p *MockEventPublisher) Publish(event DomainEvent) error {
	if err := ValidateEvent(event); err != nil {
		return err
	}

	p.mu.RLock()
	handlers, exists := p.handlers[event.EventName()]
	p.mu.RUnlock()

	if exists {
		for _, handler := range handlers {
			// 异步处理（模拟真实消息队列行为）
			go handler.Handle(event)
		}
	}

	// 模拟日志输出
	fmt.Printf("[EVENT PUBLISHED] %s at %s for aggregate %s\n",
		event.EventName(),
		event.OccurredOn().Format("2006-01-02 15:04:05"),
		event.GetAggregateID())

	return nil
}

func (p *MockEventPublisher) Subscribe(eventName string, handler EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.handlers[eventName] = append(p.handlers[eventName], handler)
	return nil
}

func (p *MockEventPublisher) Unsubscribe(eventName string, handler EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	handlers, exists := p.handlers[eventName]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if h.Name() == handler.Name() {
			p.handlers[eventName] = append(handlers[:i], handlers[i+1:]...)
			return nil
		}
	}

	return nil
}

// LoggingEventHandler 日志事件处理器示例
type LoggingEventHandler struct{}

// NewLoggingEventHandler 创建日志事件处理器
func NewLoggingEventHandler() *LoggingEventHandler {
	return &LoggingEventHandler{}
}

// Handle 处理事件（记录日志）
func (h *LoggingEventHandler) Handle(event DomainEvent) error {
	fmt.Printf("[EVENT HANDLED] %s: aggregate=%s time=%s\n",
		event.EventName(),
		event.GetAggregateID(),
		event.OccurredOn().Format(time.RFC3339))
	return nil
}

// Name 返回处理器名称
func (h *LoggingEventHandler) Name() string {
	return "logging-event-handler"
}

// EventPublishAdapter 适配器模式：将DomainEventPublisher转为具体事件发布器
type EventPublishAdapter struct {
	publisher DomainEventPublisher
}

// NewEventPublishAdapter 创建适配器
func NewEventPublishAdapter(publisher DomainEventPublisher) *EventPublishAdapter {
	return &EventPublishAdapter{
		publisher: publisher,
	}
}

// PublishUserCreatedEvent 发布用户创建事件
func (a *EventPublishAdapter) PublishUserCreatedEvent(userID, name, email string) error {
	event := NewUserCreatedEvent(userID, name, email)
	return a.publisher.Publish(event)
}

// PublishOrderPlacedEvent 发布订单创建事件
func (a *EventPublishAdapter) PublishOrderPlacedEvent(orderID, userID string, totalAmount Money) error {
	event := NewOrderPlacedEvent(orderID, userID, totalAmount)
	return a.publisher.Publish(event)
}
