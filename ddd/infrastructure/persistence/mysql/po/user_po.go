package po

import (
	"time"

	"ddd/domain/user"
)

// UserPO User persistence object
// Note: Only used for database mapping, does not contain any business logic
// Defining GORM associations (like HasMany, BelongsTo) is prohibited here
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

// TableName Specify table name
func (UserPO) TableName() string {
	return "users"
}

// FromDomain Convert domain model to persistence object
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

// ToDomain Convert persistence object to domain model
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
