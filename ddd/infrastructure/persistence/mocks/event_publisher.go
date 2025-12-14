package mocks

import (
	"fmt"
	"sync"

	"ddd/domain/shared"
)

// MockEventPublisher Mock implementation of domain event publisher
type MockEventPublisher struct {
	handlers map[string][]shared.EventHandler
	mu       sync.RWMutex
}

// NewMockEventPublisher Create Mock event publisher
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
			go handler.Handle(event) // Handle event asynchronously
		}
	}

	// Simulate event publishing log
	fmt.Printf("[EVENT PUBLISHED] %s at %s for aggregate %s\n",
		event.EventName(),
		event.OccurredOn().Format("2006-01-02 15:04:05"),
		event.GetAggregateID())

	return nil
}

func (p *MockEventPublisher) Subscribe(eventName string, handler shared.EventHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already subscribed
	for _, h := range p.handlers[eventName] {
		if h.Name() == handler.Name() {
			return nil // Already exists, do not add again
		}
	}

	p.handlers[eventName] = append(p.handlers[eventName], handler)
	return nil
}

// Unsubscribe Unsubscribe
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

// MockHandler Mock event handler
type MockHandler struct {
	name string
}

// NewMockHandler Create Mock handler
func NewMockHandler(name string) *MockHandler {
	return &MockHandler{name: name}
}

// Handle Handle event (only for testing)
func (h *MockHandler) Handle(event shared.DomainEvent) error {
	return nil
}

// Name Return handler name
func (h *MockHandler) Name() string {
	return h.name
}
