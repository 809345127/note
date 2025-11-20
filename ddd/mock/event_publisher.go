package mock

import (
	"ddd-example/domain"
	"fmt"
	"sync"
)

// MockEventPublisher 领域事件发布器的Mock实现
type MockEventPublisher struct {
	handlers map[string][]func(domain.DomainEvent)
	mu       sync.RWMutex
}

// NewMockEventPublisher 创建Mock事件发布器
func NewMockEventPublisher() *MockEventPublisher {
	return &MockEventPublisher{
		handlers: make(map[string][]func(domain.DomainEvent)),
	}
}

func (p *MockEventPublisher) Publish(event domain.DomainEvent) error {
	p.mu.RLock()
	handlers, exists := p.handlers[event.EventName()]
	p.mu.RUnlock()
	
	if exists {
		for _, handler := range handlers {
			go handler(event) // 异步处理事件
		}
	}
	
	// 模拟事件发布日志
	fmt.Printf("[EVENT PUBLISHED] %s at %s for aggregate %s\n", 
		event.EventName(), 
		event.OccurredOn().Format("2006-01-02 15:04:05"),
		event.GetAggregateID())
	
	return nil
}

func (p *MockEventPublisher) Subscribe(eventName string, handler func(domain.DomainEvent)) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.handlers[eventName] = append(p.handlers[eventName], handler)
	return nil
}