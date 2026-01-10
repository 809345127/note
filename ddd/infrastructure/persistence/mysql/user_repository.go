package mysql

import (
	"context"
	"errors"
	"strings"

	"ddd/domain/shared"
	"ddd/domain/user"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/mysql/po"

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

// isDuplicateKeyError checks if the error is a MySQL duplicate key error
// MySQL error code 1062: Duplicate entry for key
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// Check for GORM's ErrDuplicatedKey
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// Fallback: check error message contains "Duplicate entry"
	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate entry") ||
		strings.Contains(errStr, "1062")
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
			// Handle duplicate key error (MySQL error code 1062)
			if isDuplicateKeyError(err) {
				return user.NewEmailAlreadyExistsError(userPO.Email)
			}
			return err
		}
	} else {
		// Existing aggregate: use optimistic locking and dirty tracking

		// 1. Query current version from database to ensure we use the correct version for WHERE clause
		// DDD Principle: The repository is responsible for version synchronization between
		// the domain model and the persistence layer
		var currentUserPO po.UserPO
		if err := tx.First(&currentUserPO, "id = ?", u.ID()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("user not found")
			}
			return err
		}
		dbVersion := currentUserPO.Version

		// 2. Update with optimistic lock check using database version
		result := tx.Model(&po.UserPO{}).
			Where("id = ? AND version = ?", u.ID(), dbVersion).
			Updates(map[string]interface{}{
				"name":       userPO.Name,
				"email":      userPO.Email,
				"age":        userPO.Age,
				"is_active":  userPO.IsActive,
				"version":    dbVersion + 1,
				"updated_at": userPO.UpdatedAt,
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return user.ErrConcurrentModification
		}

		// 3. Notify the aggregate that persistence was successful
		// DDD Principle: The aggregate controls its own version increment, triggered by persistence
		u.IncrementVersionForSave()
	}

	// Clear new flag after successful save
	u.ClearNewFlag()
	return nil
}

// FindByID Find user by ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	// Check if context is already cancelled before making DB call
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var userPO po.UserPO

	result := r.getDB(ctx).First(&userPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, shared.NewNotFoundError("user")
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
			return nil, nil // Not found is a normal case for FindByEmail
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}

// applySpecification applies a domain specification to a GORM query
// Uses type switches to handle different specification types
// DDD Principle: Infrastructure adapts domain specifications to persistence queries
func (r *UserRepository) applySpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	if spec == nil {
		return db
	}

	// Handle composite specifications
	switch s := spec.(type) {
	case shared.AndSpecification[*user.User]:
		return r.applySpecification(r.applySpecification(db, s.Left), s.Right)
	case shared.OrSpecification[*user.User]:
		// OR: Apply left specification, then chain with Or() for right
		leftDB := r.applySpecification(db, s.Left)
		return leftDB.Or(r.applySpecification(db.Session(&gorm.Session{}), s.Right))
	case shared.NotSpecification[*user.User]:
		// NOT: Apply negation of the inner specification
		return r.applyNotSpecification(db, s.Spec)
	default:
		return r.applyConcreteSpecification(db, spec)
	}
}

// applyNotSpecification applies negation of a specification
func (r *UserRepository) applyNotSpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	switch s := spec.(type) {
	case user.ByEmailSpecification:
		return db.Where("email != ?", s.Email)
	case user.ByStatusSpecification:
		return db.Where("is_active != ?", s.Active)
	case user.ByAgeRangeSpecification:
		// Negate age range: NOT (age >= min AND age <= max)
		// Becomes: age < min OR age > max
		if s.Min > 0 && s.Max > 0 {
			return db.Where("age < ? OR age > ?", s.Min, s.Max)
		} else if s.Min > 0 {
			return db.Where("age < ?", s.Min)
		} else if s.Max > 0 {
			return db.Where("age > ?", s.Max)
		}
		return db
	case shared.AndSpecification[*user.User]:
		// NOT (A AND B) = NOT A OR NOT B (De Morgan's law)
		leftSpec := shared.Not(s.Left)
		rightSpec := shared.Not(s.Right)
		return r.applySpecification(db, shared.Or(leftSpec, rightSpec))
	case shared.OrSpecification[*user.User]:
		// NOT (A OR B) = NOT A AND NOT B (De Morgan's law)
		leftSpec := shared.Not(s.Left)
		rightSpec := shared.Not(s.Right)
		return r.applySpecification(db, shared.And(leftSpec, rightSpec))
	case shared.NotSpecification[*user.User]:
		// NOT (NOT A) = A (double negation)
		return r.applySpecification(db, s.Spec)
	default:
		// For unknown specification types, return unchanged
		return db
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
