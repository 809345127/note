package user

import (
	"time"

	"ddd-example/domain/shared"

	"github.com/google/uuid"
)

// User User aggregate root
// User is a simple aggregate root with no internal entities
// Unlike Order, User aggregate only contains User itself, no child entities
//
// Aggregate root characteristics:
// 1. All fields are private, behaviors exposed through methods
// 2. Contains version number for optimistic locking
// 3. Contains event list for recording domain events
type User struct {
	id        string
	name      string
	email     Email
	age       int
	isActive  bool
	version   int // Optimistic lock version number
	createdAt time.Time
	updatedAt time.Time

	events []shared.DomainEvent
}

// NewUser Create new user entity
func NewUser(name string, email string, age int) (*User, error) {
	if name == "" {
		return nil, ErrInvalidName
	}

	emailVO, err := NewEmail(email)
	if err != nil {
		return nil, err
	}

	if age < 0 || age > 150 {
		return nil, ErrInvalidAge
	}

	now := time.Now()
	user := &User{
		id:        uuid.New().String(),
		name:      name,
		email:     *emailVO,
		age:       age,
		isActive:  true,
		version:   0,
		createdAt: now,
		updatedAt: now,
		events:    make([]shared.DomainEvent, 0),
	}

	// Record domain event
	user.events = append(user.events, NewUserCreatedEvent(user.id, user.name, user.email.Value()))

	return user, nil
}

// ============================================================================
// Domain Behavior Methods
// ============================================================================
//
// DDD Principle: Entity state changes through behavior methods, not direct field modification
// Behavior methods encapsulate business rules and automatically maintain version numbers

// Activate Activate user
// Business scenario: Admin activates deactivated user account
func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
	u.version++
}

// Deactivate Deactivate user
// Business scenario: Admin deactivates violating user or user voluntarily deactivates account
func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
	u.version++
}

// UpdateName Update user name
// Includes business rule validation: name cannot be empty
func (u *User) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	u.name = name
	u.updatedAt = time.Now()
	u.version++
	return nil
}

// CanMakePurchase Check if user can make purchase
// This is a business rule query method that encapsulates the business definition of "can purchase"
// Business rule: User must be active and at least 18 years old
func (u *User) CanMakePurchase() bool {
	return u.isActive && u.age >= 18
}

// ============================================================================
// Getters - Read-only Accessors
// ============================================================================
//
// DDD Principle: Fields are private, exposed through getters for read-only access
func (u *User) ID() string           { return u.id }
func (u *User) Name() string         { return u.name }
func (u *User) Email() Email         { return u.email }
func (u *User) Age() int             { return u.age }
func (u *User) IsActive() bool       { return u.isActive }
func (u *User) Version() int         { return u.version }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// PullEvents Get and clear aggregate root's event list
func (u *User) PullEvents() []shared.DomainEvent {
	events := make([]shared.DomainEvent, len(u.events))
	copy(events, u.events)
	u.events = make([]shared.DomainEvent, 0)
	return events
}

// ReconstructionDTO User reconstruction data transfer object
// Limited to repository layer usage, for reconstructing User aggregate root from database
// ⚠️ Note: This DTO should only be used in repository implementation, not called from application layer
type ReconstructionDTO struct {
	ID        string
	Name      string
	Email     string
	Age       int
	IsActive  bool
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RebuildFromDTO Reconstruct User aggregate root from DTO
// This is a factory method specifically for repository layer to reconstruct aggregate root
// ⚠️ Note: This method should only be used in repository implementation, not called from application layer
func RebuildFromDTO(dto ReconstructionDTO) *User {
	return &User{
		id:        dto.ID,
		name:      dto.Name,
		email:     Email{value: dto.Email},
		age:       dto.Age,
		isActive:  dto.IsActive,
		version:   dto.Version,
		createdAt: dto.CreatedAt,
		updatedAt: dto.UpdatedAt,
		events:    []shared.DomainEvent{},
	}
}

// Compile-time check that User implements AggregateRoot interface
var _ shared.AggregateRoot = (*User)(nil)
