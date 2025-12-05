package po

import (
	"time"

	"ddd-example/domain/user"
)

// UserPO 用户持久化对象
// 注意：只用于数据库映射，不包含任何业务逻辑
// 禁止在此定义 GORM 关联（如 HasMany, BelongsTo）
type UserPO struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Name      string    `gorm:"size:100;not null"`
	Email     string    `gorm:"size:255;uniqueIndex;not null"`
	Age       int       `gorm:"not null"`
	IsActive  bool      `gorm:"default:true"`
	Version   int       `gorm:"default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (UserPO) TableName() string {
	return "users"
}

// FromDomain 将领域模型转换为持久化对象
func FromUserDomain(u *user.User) *UserPO {
	return &UserPO{
		ID:        u.ID(),
		Name:      u.Name(),
		Email:     u.Email().Value(),
		Age:       u.Age(),
		IsActive:  u.IsActive(),
		Version:   u.Version(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

// ToDomain 将持久化对象转换为领域模型
func (po *UserPO) ToDomain() *user.User {
	return user.RebuildFromDTO(user.ReconstructionDTO{
		ID:        po.ID,
		Name:      po.Name,
		Email:     po.Email,
		Age:       po.Age,
		IsActive:  po.IsActive,
		Version:   po.Version,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	})
}
