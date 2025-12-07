package user

import "time"

// UserCreatedEvent User created event
type UserCreatedEvent struct {
	userID     string
	name       string
	email      string
	occurredOn time.Time
}

func NewUserCreatedEvent(userID, name, email string) *UserCreatedEvent {
	return &UserCreatedEvent{
		userID:     userID,
		name:       name,
		email:      email,
		occurredOn: time.Now(),
	}
}

func (e *UserCreatedEvent) EventName() string       { return "user.created" }
func (e *UserCreatedEvent) OccurredOn() time.Time   { return e.occurredOn }
func (e *UserCreatedEvent) GetAggregateID() string  { return e.userID }
func (e *UserCreatedEvent) UserID() string          { return e.userID }
func (e *UserCreatedEvent) Name() string            { return e.name }
func (e *UserCreatedEvent) Email() string           { return e.email }
