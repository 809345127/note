package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// 领域错误定义
// ============================================================================
//
// DDD原则：领域层定义业务相关的错误类型
// 这些错误表达业务规则的违反，而非技术错误

var (
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrInvalidName       = errors.New("name cannot be empty")
	ErrInvalidAge        = errors.New("age must be between 0 and 150")
	ErrUserNotActive     = errors.New("user is not active")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// ============================================================================
// User 聚合根
// ============================================================================
//
// User是一个简单的聚合根，没有包含内部实体
// 与Order不同，User聚合内只有User自身，没有子实体
//
// 聚合根特征：
// 1. 所有字段私有，通过方法暴露行为
// 2. 包含版本号用于乐观锁
// 3. 包含事件列表用于记录领域事件

// User 用户聚合根
type User struct {
	id        string
	name      string
	email     Email
	age       int
	isActive  bool
	version   int // 乐观锁版本号
	createdAt time.Time
	updatedAt time.Time

	events []DomainEvent
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
	user := &User{
		id:        uuid.New().String(),
		name:      name,
		email:     *emailVO,
		age:       age,
		isActive:  true,
		version:   0,
		createdAt: now,
		updatedAt: now,
		events:    make([]DomainEvent, 0),
	}

	// 记录领域事件
	user.events = append(user.events, NewUserCreatedEvent(user.id, user.name, user.email.Value()))

	return user, nil
}

// ============================================================================
// 领域行为方法
// ============================================================================
//
// DDD原则：实体的状态变更通过行为方法进行，而非直接修改字段
// 行为方法封装了业务规则，并自动维护版本号

// Activate 激活用户
// 业务场景：管理员激活被停用的用户账号
func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
	u.version++
}

// Deactivate 停用用户
// 业务场景：管理员停用违规用户或用户主动注销
func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
	u.version++
}

// UpdateName 更新用户名称
// 包含业务规则验证：名称不能为空
func (u *User) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	u.name = name
	u.updatedAt = time.Now()
	u.version++
	return nil
}

// CanMakePurchase 检查用户是否可以购买
// 这是一个业务规则查询方法，封装了"可购买"的业务定义
// 业务规则：用户必须激活且年满18岁
func (u *User) CanMakePurchase() bool {
	return u.isActive && u.age >= 18
}

// ============================================================================
// Getters - 只读访问器
// ============================================================================
//
// DDD原则：字段私有，通过getter暴露只读访问
func (u *User) ID() string        { return u.id }
func (u *User) Name() string      { return u.name }
func (u *User) Email() Email      { return u.email }
func (u *User) Age() int          { return u.age }
func (u *User) IsActive() bool    { return u.isActive }
func (u *User) Version() int      { return u.version }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// PullEvents 获取并清空聚合根的事件列表
func (u *User) PullEvents() []DomainEvent {
	events := make([]DomainEvent, len(u.events))
	copy(events, u.events)
	u.events = make([]DomainEvent, 0)
	return events
}

func (u *User) recordEvent(event DomainEvent) {
	u.events = append(u.events, event)
}

// UserReconstructionDTO 用户重建数据传输对象
// 仅限于仓储层使用，用于从数据库重建User聚合根
// ⚠️ 注意：此DTO仅应在仓储实现中使用，不应在应用层调用
type UserReconstructionDTO struct {
	ID        string
	Name      string
	Email     string
	Age       int
	IsActive  bool
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RebuildUserFromDTO 从DTO重建User聚合根
// 这是一个工厂方法，专门用于仓储层重建聚合根
// ⚠️ 注意：此方法仅应在仓储实现中使用，不应在应用层调用
func RebuildUserFromDTO(dto UserReconstructionDTO) *User {
	return &User{
		id:        dto.ID,
		name:      dto.Name,
		email:     Email{value: dto.Email},
		age:       dto.Age,
		isActive:  dto.IsActive,
		version:   dto.Version,
		createdAt: dto.CreatedAt,
		updatedAt: dto.UpdatedAt,
		events:    []DomainEvent{},
	}
}