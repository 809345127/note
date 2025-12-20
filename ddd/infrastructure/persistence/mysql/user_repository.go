package mysql

import (
	"context"
	"errors"

	"ddd/domain/shared"
	"ddd/domain/user"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/mysql/po"

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
// Uses optimistic locking for concurrency control
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	// Check if we're already in a UoW transaction
	if tx := persistence.TxFromContext(ctx); tx != nil {
		// Use the existing transaction from UoW
		return r.saveWithTx(tx, u)
	}

	// No UoW transaction - create our own for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.saveWithTx(tx, u)
	})
}

// saveWithTx performs the actual save operations within a transaction
// Uses optimistic locking: checks version to prevent concurrent modification
func (r *UserRepository) saveWithTx(tx *gorm.DB, u *user.User) error {
	userPO := po.FromUserDomain(u)

	if u.IsNew() {
		// New aggregate: insert
		if err := tx.Create(userPO).Error; err != nil {
			return err
		}
	} else {
		// Existing aggregate: update with optimistic lock check
		result := tx.Model(&po.UserPO{}).
			Where("id = ? AND version = ?", u.ID(), u.Version()).
			Updates(map[string]interface{}{
				"name":       userPO.Name,
				"email":      userPO.Email,
				"age":        userPO.Age,
				"is_active":  userPO.IsActive,
				"version":    u.Version() + 1,
				"updated_at": userPO.UpdatedAt,
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return user.ErrConcurrentModification
		}
	}

	// Clear new flag after successful save
	u.ClearNewFlag()
	return nil
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
	spec := user.ByEmailSpecification{Email: email}
	return r.findOneBySpecification(ctx, spec)
}

// FindBySpecification Find users by specification
// Implements the domain.Repository interface for flexible query composition
func (r *UserRepository) FindBySpecification(ctx context.Context, spec shared.Specification[*user.User]) ([]*user.User, error) {
	db := r.getDB(ctx)

	// Apply specification to query
	db = r.applySpecification(db, spec)
	if db.Error != nil {
		return nil, db.Error
	}

	// Execute query
	var userPOs []po.UserPO
	if err := db.Find(&userPOs).Error; err != nil {
		return nil, err
	}

	// Convert to domain objects
	users := make([]*user.User, len(userPOs))
	for i, userPO := range userPOs {
		users[i] = userPO.ToDomain()
	}

	return users, nil
}

// findOneBySpecification finds a single user by specification
// Adds LIMIT 1 and returns the first matching user or error
func (r *UserRepository) findOneBySpecification(ctx context.Context, spec shared.Specification[*user.User]) (*user.User, error) {
	db := r.getDB(ctx)

	// Apply specification to query
	db = r.applySpecification(db, spec)
	if db.Error != nil {
		return nil, db.Error
	}

	// Execute query with LIMIT 1
	var userPO po.UserPO
	result := db.First(&userPO)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}

// applySpecification applies a domain specification to a GORM query
// Uses type switches to handle different specification types
func (r *UserRepository) applySpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	if spec == nil {
		return db
	}

	// Handle composite specifications
	switch s := spec.(type) {
	case shared.AndSpecification[*user.User]:
		return r.applySpecification(r.applySpecification(db, s.Left), s.Right)
	// Note: OR and NOT specifications are more complex to implement with GORM
	// For simplicity in this first implementation, we only support AND
	default:
		return r.applyConcreteSpecification(db, spec)
	}
}

// applyConcreteSpecification applies concrete domain specifications
func (r *UserRepository) applyConcreteSpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	switch s := spec.(type) {
	case user.ByEmailSpecification:
		return db.Where("email = ?", s.Email)
	case user.ByStatusSpecification:
		return db.Where("is_active = ?", s.Active)
	case user.ByAgeRangeSpecification:
		// Handle optional min and max ages
		if s.Min > 0 {
			db = db.Where("age >= ?", s.Min)
		}
		if s.Max > 0 {
			db = db.Where("age <= ?", s.Max)
		}
		return db
	default:
		// Unknown specification type - return unchanged
		return db
	}
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
