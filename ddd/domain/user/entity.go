package user

import (
	"fmt"
	"time"

	"ddd/domain/shared"

	"github.com/google/uuid"
)

// User 是用户聚合根。
type User struct {
	id        string
	name      string
	email     Email
	age       int
	isActive  bool
	version   int
	createdAt time.Time
	updatedAt time.Time
	isNew     bool

	events []shared.DomainEvent
}

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

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user ID: %w", err)
	}

	now := time.Now()
	u := &User{
		id:        id.String(),
		name:      name,
		email:     *emailVO,
		age:       age,
		isActive:  true,
		version:   0,
		createdAt: now,
		updatedAt: now,
		isNew:     true,
		events:    make([]shared.DomainEvent, 0),
	}
	u.events = append(u.events, NewUserCreatedEvent(u.id, u.name, u.email.Value()))
	return u, nil
}

func (u *User) Activate() {
	if u.isActive {
		return
	}
	u.isActive = true
	u.updatedAt = time.Now()
	u.events = append(u.events, NewUserActivatedEvent(u.id))
}

func (u *User) Deactivate() {
	if !u.isActive {
		return
	}
	u.isActive = false
	u.updatedAt = time.Now()
	u.events = append(u.events, NewUserDeactivatedEvent(u.id))
}

func (u *User) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	u.name = name
	u.updatedAt = time.Now()
	return nil
}

func (u *User) IncrementVersionForSave() {
	u.version++
	u.updatedAt = time.Now()
}

func (u *User) CanMakePurchase() bool {
	return u.isActive && u.age >= 18
}

func (u *User) ID() string           { return u.id }
func (u *User) Name() string         { return u.name }
func (u *User) Email() Email         { return u.email }
func (u *User) Age() int             { return u.age }
func (u *User) IsActive() bool       { return u.isActive }
func (u *User) Version() int         { return u.version }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }
func (u *User) IsNew() bool          { return u.isNew }
func (u *User) ClearNewFlag()        { u.isNew = false }

func (u *User) PullEvents() []shared.DomainEvent {
	events := make([]shared.DomainEvent, len(u.events))
	copy(events, u.events)
	u.events = make([]shared.DomainEvent, 0)
	return events
}

// ReconstructionDTO 仅供仓储层重建聚合使用。
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

// RebuildFromDTO 仅供仓储层调用。
func RebuildFromDTO(dto ReconstructionDTO) *User {
	emailVO, err := NewEmail(dto.Email)
	if err != nil {
		emailVO = &Email{value: dto.Email}
	}
	return &User{
		id:        dto.ID,
		name:      dto.Name,
		email:     *emailVO,
		age:       dto.Age,
		isActive:  dto.IsActive,
		version:   dto.Version,
		createdAt: dto.CreatedAt,
		updatedAt: dto.UpdatedAt,
		isNew:     false,
		events:    nil,
	}
}

var _ shared.AggregateRoot = (*User)(nil)
