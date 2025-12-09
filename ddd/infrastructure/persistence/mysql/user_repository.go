package mysql

import (
	"context"
	"errors"

	"ddd-example/domain/user"
	"ddd-example/infrastructure/persistence"
	"ddd-example/infrastructure/persistence/mysql/po"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository MySQL/GORM implementation of user repository
// DDD principle: Repository is only responsible for persistence of aggregate roots, not event publishing
// GORM usage specification: Association features are prohibited to maintain DDD aggregate boundaries
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository Create user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// getDB returns the transaction from context if available, otherwise the default db
func (r *UserRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

// NextIdentity Generate new user ID
func (r *UserRepository) NextIdentity() string {
	return "user-" + uuid.New().String()
}

// Save Save user (create or update)
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	userPO := po.FromUserDomain(u)

	// Use Save method, GORM will determine whether to Create or Update based on primary key
	result := r.getDB(ctx).Save(userPO)
	return result.Error
}

// FindByID Find user by ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	var userPO po.UserPO

	result := r.getDB(ctx).First(&userPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}

// FindByEmail Find user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var userPO po.UserPO

	result := r.getDB(ctx).First(&userPO, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}

// Remove Delete user (logical deletion: mark as inactive)
// DDD principle: Logical deletion is recommended over physical deletion to preserve business history
func (r *UserRepository) Remove(ctx context.Context, id string) error {
	result := r.getDB(ctx).
		Model(&po.UserPO{}).
		Where("id = ?", id).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// Compile-time interface implementation check
var _ user.Repository = (*UserRepository)(nil)
