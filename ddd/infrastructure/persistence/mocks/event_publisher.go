package mocks

import (
	"fmt"
	"sync"

	"ddd-example/domain/shared"
)

// MockEventPublisher 领域事件发布器的Mock实现
type MockEventPublisher struct {
	handlers map[string][]shared.EventHandler
	mu       sync.RWMutex
}

// NewMockEventPublisher 创建Mock事件发布器
func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		handlers: make(map[string][]shared.EventHandler),
	}
}

func (p *MockEventPublisher) Publish(event shared.DomainEvent) error {
	p.mu.RLock()
	handlers, exists := p.handlers[event.EventName()]
	p.mu.RUnlock()

	if exists {
		for _, handler := range handlers {
			go handler.Handle(event) // 异步处理事件
		}
	}

	// 模拟事件发布日志
	fmt.Printf("[EVENT PUBLISHED] %s at %s for aggregate %s\n",
		event.EventName(),
		event.OccurredOn().Format("2006-01-02 15:04:05"),
		event.GetAggregateID())

	return nil
}

func (p *MockEventPublisher) Subscribe(eventName string, handler shared.EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否已订阅
	for _, h := range p.handlers[eventName] {
		if h.Name() == handler.Name() {
			return nil // 已存在，不再添加
		}
	}

	p.handlers[eventName] = append(p.handlers[eventName], handler)
	return nil
}

// Unsubscribe 取消订阅
func (p *MockEventPublisher) Unsubscribe(eventName string, handler shared.EventHandler) error {
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

// MockHandler Mock事件处理器
type MockHandler struct {
	name string
}

// NewMockHandler 创建Mock处理器
func NewMockHandler(name string) *MockHandler {
	return &MockHandler{name: name}
}

// Handle 处理事件（仅用于测试）
func (h *MockHandler) Handle(event shared.DomainEvent) error {
	return nil
}

// Name 返回处理器名称
func (h *MockHandler) Name() string {
	return h.name
}
