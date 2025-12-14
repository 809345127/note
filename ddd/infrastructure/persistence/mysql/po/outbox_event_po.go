package po

import (
	"encoding/json"
	"time"

	"ddd/domain/shared"

	"github.com/google/uuid"
)

// OutboxEventPO Outbox event persistence object
// Implements transactional outbox pattern for reliable event publishing
type OutboxEventPO struct {
	ID          string    `gorm:"primaryKey;size:64"`
	AggregateID string    `gorm:"size:64;index;not null"`
	EventType   string    `gorm:"size:100;index;not null"` // e.g., "user.created", "order.placed"
	Payload     string    `gorm:"type:json;not null"`      // JSON serialized event data
	Status      string    `gorm:"size:20;default:PENDING;not null"` // PENDING, PROCESSING, PUBLISHED, FAILED
	RetryCount  int       `gorm:"default:0;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName Specify table name
func (OutboxEventPO) TableName() string {
	return "outbox_events"
}

// EventStatus Outbox event status enum
type EventStatus string

const (
	EventStatusPending    EventStatus = "PENDING"
	EventStatusProcessing EventStatus = "PROCESSING"
	EventStatusPublished  EventStatus = "PUBLISHED"
	EventStatusFailed     EventStatus = "FAILED"
)

// FromDomainEvent Convert domain event to outbox persistence object
func FromDomainEvent(event shared.DomainEvent) (*OutboxEventPO, error) {
	// Serialize event to JSON
	payload, err := serializeEventToJSON(event)
	if err != nil {
		return nil, err
	}

	// Generate a unique ID for the outbox event
	eventID := uuid.New().String()

	return &OutboxEventPO{
		ID:          eventID,
		AggregateID: event.GetAggregateID(),
		EventType:   event.EventName(),
		Payload:     payload,
		Status:      string(EventStatusPending),
		RetryCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// serializeEventToJSON Serialize domain event to JSON string
func serializeEventToJSON(event shared.DomainEvent) (string, error) {
	// Create a generic map to store event data
	eventData := map[string]interface{}{
		"event_name":   event.EventName(),
		"aggregate_id": event.GetAggregateID(),
		"occurred_on":  event.OccurredOn(),
	}

	// Add event-specific data based on event type
	// Check for OrderPlacedEvent first (has OrderID() method)
	if orderEvent, ok := event.(interface{ OrderID() string }); ok {
		eventData["order_id"] = orderEvent.OrderID()
		// OrderPlacedEvent also has UserID() method
		if userIDGetter, ok := event.(interface{ UserID() string }); ok {
			eventData["user_id"] = userIDGetter.UserID()
		}
		// OrderPlacedEvent has TotalAmount() method
		if totalAmountGetter, ok := event.(interface{ TotalAmount() shared.Money }); ok {
			money := totalAmountGetter.TotalAmount()
			eventData["total_amount"] = money.Amount()
			eventData["total_currency"] = money.Currency()
		}
	} else if userEvent, ok := event.(interface{ UserID() string }); ok {
		// UserCreatedEvent has UserID(), Name(), Email() methods
		eventData["user_id"] = userEvent.UserID()
		if nameGetter, ok := event.(interface{ Name() string }); ok {
			eventData["name"] = nameGetter.Name()
		}
		if emailGetter, ok := event.(interface{ Email() string }); ok {
			eventData["email"] = emailGetter.Email()
		}
	}
	// Additional event types can be added here as needed

	// Marshal to JSON
	data, err := json.Marshal(eventData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}


// ToEventData Extract event data from outbox PO (for debugging/testing)
func (po *OutboxEventPO) ToEventData() (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(po.Payload), &data); err != nil {
		return nil, err
	}
	return data, nil
}