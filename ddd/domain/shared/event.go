package shared

import (
	"fmt"
	"sync"
	"time"
)

type DomainEvent interface {
	EventName() string
	OccurredOn() time.Time
	GetAggregateID() string
}
type DomainEventPublisher interface {
	Publish(event DomainEvent) error
	Subscribe(eventName string, handler EventHandler) error
	Unsubscribe(eventName string, handler EventHandler) error
}
type EventHandler interface {
	Handle(event DomainEvent) error
	Name() string
}
type EventPublishOptions struct {
	Retry   int
	Timeout time.Duration
}
type EventSubscription struct {
	ID        string       `json:"id"`
	EventName string       `json:"event_name"`
	Handler   EventHandler `json:"-"`
	CreatedAt time.Time    `json:"created_at"`
	IsActive  bool         `json:"is_active"`
}
type EventPublishResult struct {
	EventName   string    `json:"event_name"`
	Success     bool      `json:"success"`
	Message     string    `json:"message,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

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

type EventBus struct {
	handlers  map[string][]EventHandler
	mu        sync.RWMutex
	history   []EventPublishResult
	muHistory sync.Mutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
		history:  make([]EventPublishResult, 0),
	}
}
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
		var errs []error
		for _, handler := range handlers {
			if err := handler.Handle(event); err != nil {
				errs = append(errs, fmt.Errorf("handler %s: %w", handler.Name(), err))
			}
		}
		if len(errs) > 0 {
			result.Success = false
			result.Message = fmt.Sprintf("%d handlers failed", len(errs))
			bus.muHistory.Lock()
			bus.history = append(bus.history, result)
			if len(bus.history) > 1000 {
				bus.history = bus.history[len(bus.history)-1000:]
			}
			bus.muHistory.Unlock()
			return fmt.Errorf("event %s: %d handlers failed: %v", event.EventName(), len(errs), errs)
		}
	} else {
		result.Message = "no handlers registered for this event"
	}

	bus.muHistory.Lock()
	bus.history = append(bus.history, result)
	if len(bus.history) > 1000 {
		bus.history = bus.history[len(bus.history)-1000:]
	}
	bus.muHistory.Unlock()

	return nil
}
func (bus *EventBus) Subscribe(eventName string, handler EventHandler) error {
	if eventName == "" {
		return fmt.Errorf("event name cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	for _, h := range bus.handlers[eventName] {
		if h.Name() == handler.Name() {
			return fmt.Errorf("handler %s already subscribed to %s", handler.Name(), eventName)
		}
	}

	bus.handlers[eventName] = append(bus.handlers[eventName], handler)
	return nil
}
func (bus *EventBus) Unsubscribe(eventName string, handler EventHandler) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers, exists := bus.handlers[eventName]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if h.Name() == handler.Name() {
			bus.handlers[eventName] = append(handlers[:i], handlers[i+1:]...)
			return nil
		}
	}

	return nil
}
func (bus *EventBus) GetPublishHistory() []EventPublishResult {
	bus.muHistory.Lock()
	defer bus.muHistory.Unlock()

	history := make([]EventPublishResult, len(bus.history))
	copy(history, bus.history)
	return history
}

type FuncHandler struct {
	name string
	fn   func(DomainEvent) error
}

func NewFuncHandler(name string, fn func(DomainEvent) error) *FuncHandler {
	if name == "" {
		name = fmt.Sprintf("func-handler-%d", time.Now().UnixNano())
	}
	return &FuncHandler{
		name: name,
		fn:   fn,
	}
}
func (h *FuncHandler) Handle(event DomainEvent) error {
	return h.fn(event)
}
func (h *FuncHandler) Name() string {
	return h.name
}

type MockEventPublisher struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

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
			go handler.Handle(event)
		}
	}

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

type LoggingEventHandler struct{}

func NewLoggingEventHandler() *LoggingEventHandler {
	return &LoggingEventHandler{}
}
func (h *LoggingEventHandler) Handle(event DomainEvent) error {
	fmt.Printf("[EVENT HANDLED] %s: aggregate=%s time=%s\n",
		event.EventName(),
		event.GetAggregateID(),
		event.OccurredOn().Format(time.RFC3339))
	return nil
}
func (h *LoggingEventHandler) Name() string {
	return "logging-event-handler"
}
