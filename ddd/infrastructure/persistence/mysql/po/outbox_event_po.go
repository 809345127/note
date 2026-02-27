package po

import (
	"encoding/json"
	"time"

	"ddd/domain/shared"

	"github.com/google/uuid"
)

type OutboxEventPO struct {
	ID          string    `gorm:"primaryKey;size:64"`
	AggregateID string    `gorm:"size:64;index;not null"`
	EventType   string    `gorm:"size:100;index;not null"`
	Payload     string    `gorm:"type:json;not null"`
	Status      string    `gorm:"size:20;default:PENDING;not null"`
	RetryCount  int       `gorm:"default:0;not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (OutboxEventPO) TableName() string {
	return "outbox_events"
}

type EventStatus string

const (
	EventStatusPending    EventStatus = "PENDING"
	EventStatusProcessing EventStatus = "PROCESSING"
	EventStatusPublished  EventStatus = "PUBLISHED"
	EventStatusFailed     EventStatus = "FAILED"
)

func FromDomainEvent(event shared.DomainEvent) (*OutboxEventPO, error) {
	payload, err := serializeEventToJSON(event)
	if err != nil {
		return nil, err
	}
	eventID := uuid.Must(uuid.NewV7()).String()

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
func serializeEventToJSON(event shared.DomainEvent) (string, error) {
	eventData := map[string]interface{}{
		"event_name":   event.EventName(),
		"aggregate_id": event.GetAggregateID(),
		"occurred_on":  event.OccurredOn(),
	}
	if orderEvent, ok := event.(interface{ OrderID() string }); ok {
		eventData["order_id"] = orderEvent.OrderID()
		if userIDGetter, ok := event.(interface{ UserID() string }); ok {
			eventData["user_id"] = userIDGetter.UserID()
		}
		if totalAmountGetter, ok := event.(interface{ TotalAmount() shared.Money }); ok {
			money := totalAmountGetter.TotalAmount()
			eventData["total_amount"] = money.Amount()
			eventData["total_currency"] = money.Currency()
		}
	} else if userEvent, ok := event.(interface{ UserID() string }); ok {
		eventData["user_id"] = userEvent.UserID()
		if nameGetter, ok := event.(interface{ Name() string }); ok {
			eventData["name"] = nameGetter.Name()
		}
		if emailGetter, ok := event.(interface{ Email() string }); ok {
			eventData["email"] = emailGetter.Email()
		}
	}
	data, err := json.Marshal(eventData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
func (po *OutboxEventPO) ToEventData() (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(po.Payload), &data); err != nil {
		return nil, err
	}
	return data, nil
}
