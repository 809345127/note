package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrInvalidName      = errors.New("name cannot be empty")
	ErrInvalidAge       = errors.New("age must be between 0 and 150")
	ErrUserNotActive    = errors.New("user is not active")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// User 用户实体 - 包含业务逻辑和状态
type User struct {
	id        string
	name      string
	email     Email
	age       int
	isActive  bool
	createdAt time.Time
	updatedAt time.Time
}

// NewUser 创建新用户实体
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
	return &User{
		id:        uuid.New().String(),
		name:      name,
		email:     *emailVO,
		age:       age,
		isActive:  true,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// 领域行为方法
func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
}

func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
}

func (u *User) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	u.name = name
	u.updatedAt = time.Now()
	return nil
}

func (u *User) CanMakePurchase() bool {
	return u.isActive && u.age >= 18
}

// 获取器方法 - 保持封装性
func (u *User) ID() string        { return u.id }
func (u *User) Name() string      { return u.name }
func (u *User) Email() Email      { return u.email }
func (u *User) Age() int          { return u.age }
func (u *User) IsActive() bool    { return u.isActive }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }