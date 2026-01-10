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
func (e *UserCreatedEvent) UserID() string { return e.userID }
func (e *UserCreatedEvent) Name() string   { return e.name }
func (e *UserCreatedEvent) Email() string  { return e.email }

// UserActivatedEvent User activated event
type UserActivatedEvent struct {
	userID     string
	occurredOn time.Time
}

func NewUserActivatedEvent(userID string) *UserActivatedEvent {
	return &UserActivatedEvent{
		userID:     userID,
		occurredOn: time.Now(),
	}
}

func (e *UserActivatedEvent) EventName() string       { return "user.activated" }
func (e *UserActivatedEvent) OccurredOn() time.Time  { return e.occurredOn }
func (e *UserActivatedEvent) GetAggregateID() string  { return e.userID }
func (e *UserActivatedEvent) UserID() string          { return e.userID }

// UserDeactivatedEvent User deactivated event
type UserDeactivatedEvent struct {
	userID     string
	occurredOn time.Time
}

func NewUserDeactivatedEvent(userID string) *UserDeactivatedEvent {
	return &UserDeactivatedEvent{
		userID:     userID,
		occurredOn: time.Now(),
	}
}

func (e *UserDeactivatedEvent) EventName() string       { return "user.deactivated" }
func (e *UserDeactivatedEvent) OccurredOn() time.Time  { return e.occurredOn }
func (e *UserDeactivatedEvent) GetAggregateID() string  { return e.userID }
func (e *UserDeactivatedEvent) UserID() string          { return e.userID }
