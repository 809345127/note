package shared

import (
	"fmt"
	"sync"
	"time"
)

// DomainEvent Domain Event Interface
type DomainEvent interface {
	EventName() string
	OccurredOn() time.Time
	GetAggregateID() string
}

// DomainEventPublisher Domain Event Publisher Interface
// DDD principle: Domain layer defines interfaces, infrastructure layer provides implementation
// Application layer publishes events through interfaces, not dependent on specific implementation
type DomainEventPublisher interface {
	// Publish Publish domain event
	Publish(event DomainEvent) error

	// Subscribe Subscribe to specific types of events
	Subscribe(eventName string, handler EventHandler) error

	// Unsubscribe Unsubscribe
	Unsubscribe(eventName string, handler EventHandler) error
}

// EventHandler Event Handler Interface
type EventHandler interface {
	// Handle Process event
	Handle(event DomainEvent) error

	// Name Return handler name for identification
	Name() string
}

// EventPublishOptions Event Publish Options
type EventPublishOptions struct {
	// Retry Number of retries on failure
	Retry int

	// Timeout Publishing timeout
	Timeout time.Duration
}

// EventSubscription Event Subscription
type EventSubscription struct {
	ID        string       `json:"id"`
	EventName string       `json:"event_name"`
	Handler   EventHandler `json:"-"`
	CreatedAt time.Time    `json:"created_at"`
	IsActive  bool         `json:"is_active"`
}

// EventPublishResult Event Publish Result
type EventPublishResult struct {
	EventName   string    `json:"event_name"`
	Success     bool      `json:"success"`
	Message     string    `json:"message,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

// ValidateEvent Validate Domain Event
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

// EventBus In-memory Event Bus Implementation
type EventBus struct {
	handlers  map[string][]EventHandler
	mu        sync.RWMutex
	history   []EventPublishResult
	muHistory sync.Mutex
}

// NewEventBus Create New Event Bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[string][]EventHandler),
		history:  make([]EventPublishResult, 0),
	}
}

// Publish Publish Event (Synchronous Execution)
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

// Subscribe Subscribe to Event
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

// Unsubscribe Unsubscribe
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

// GetPublishHistory Get publish history (for debugging)
func (bus *EventBus) GetPublishHistory() []EventPublishResult {
	bus.muHistory.Lock()
	defer bus.muHistory.Unlock()

	history := make([]EventPublishResult, len(bus.history))
	copy(history, bus.history)
	return history
}

// FuncHandler Functional Event Handler
type FuncHandler struct {
	name string
	fn   func(DomainEvent) error
}

// NewFuncHandler Create Functional Handler
func NewFuncHandler(name string, fn func(DomainEvent) error) *FuncHandler {
	if name == "" {
		name = fmt.Sprintf("func-handler-%d", time.Now().UnixNano())
	}
	return &FuncHandler{
		name: name,
		fn:   fn,
	}
}

// Handle Process event
func (h *FuncHandler) Handle(event DomainEvent) error {
	return h.fn(event)
}

// Name Return handler name
func (h *FuncHandler) Name() string {
	return h.name
}

// MockEventPublisher Mock Event Publisher
type MockEventPublisher struct {
	handlers map[string][]EventHandler
	mu       sync.RWMutex
}

// NewMockEventPublisher Create Mock Event Publisher
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

// LoggingEventHandler Logging Event Handler Example
type LoggingEventHandler struct{}

// NewLoggingEventHandler Create Logging Event Handler
func NewLoggingEventHandler() *LoggingEventHandler {
	return &LoggingEventHandler{}
}

// Handle Process event (log)
func (h *LoggingEventHandler) Handle(event DomainEvent) error {
	fmt.Printf("[EVENT HANDLED] %s: aggregate=%s time=%s\n",
		event.EventName(),
		event.GetAggregateID(),
		event.OccurredOn().Format(time.RFC3339))
	return nil
}

// Name Return handler name
func (h *LoggingEventHandler) Name() string {
	return "logging-event-handler"
}
