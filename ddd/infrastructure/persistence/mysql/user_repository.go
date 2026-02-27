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

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}
func (r *UserRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate entry") ||
		strings.Contains(errStr, "1062")
}
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return r.saveWithTx(tx, u)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.saveWithTx(tx, u)
	})
}
func (r *UserRepository) saveWithTx(tx *gorm.DB, u *user.User) error {
	userPO := po.FromUserDomain(u)

	if u.IsNew() {
		if err := tx.Create(userPO).Error; err != nil {
			if isDuplicateKeyError(err) {
				return user.NewEmailAlreadyExistsError(userPO.Email)
			}
			return err
		}
	} else {
		expectedVersion := u.Version()

		// 严格乐观锁：必须使用聚合当前版本作为更新条件，避免静默覆盖并发写入。
		result := tx.Model(&po.UserPO{}).
			Where("id = ? AND version = ?", u.ID(), expectedVersion).
			Updates(map[string]interface{}{
				"name":       userPO.Name,
				"email":      userPO.Email,
				"age":        userPO.Age,
				"is_active":  userPO.IsActive,
				"version":    expectedVersion + 1,
				"updated_at": userPO.UpdatedAt,
			})

		if result.Error != nil {
			if isDuplicateKeyError(result.Error) {
				return user.NewEmailAlreadyExistsError(userPO.Email)
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := tx.Model(&po.UserPO{}).Where("id = ?", u.ID()).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return user.NewUserNotFoundError(u.ID())
			}
			return user.NewConcurrentModificationError(u.ID())
		}

		u.IncrementVersionForSave()
	}
	u.ClearNewFlag()
	return nil
}
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
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
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	spec := user.ByEmailSpecification{Email: email}
	return r.findOneBySpecification(ctx, spec)
}
func (r *UserRepository) FindBySpecification(ctx context.Context, spec shared.Specification[*user.User]) ([]*user.User, error) {
	db := r.getDB(ctx)
	db = r.applySpecification(db, spec)
	if db.Error != nil {
		return nil, db.Error
	}
	var userPOs []po.UserPO
	if err := db.Find(&userPOs).Error; err != nil {
		return nil, err
	}
	users := make([]*user.User, len(userPOs))
	for i, userPO := range userPOs {
		users[i] = userPO.ToDomain()
	}

	return users, nil
}
func (r *UserRepository) findOneBySpecification(ctx context.Context, spec shared.Specification[*user.User]) (*user.User, error) {
	db := r.getDB(ctx)
	db = r.applySpecification(db, spec)
	if db.Error != nil {
		return nil, db.Error
	}
	var userPO po.UserPO
	result := db.First(&userPO)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}
func (r *UserRepository) applySpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	if spec == nil {
		return db
	}
	switch s := spec.(type) {
	case shared.AndSpecification[*user.User]:
		return r.applySpecification(r.applySpecification(db, s.Left), s.Right)
	case shared.OrSpecification[*user.User]:
		leftDB := r.applySpecification(db, s.Left)
		return leftDB.Or(r.applySpecification(db.Session(&gorm.Session{}), s.Right))
	case shared.NotSpecification[*user.User]:
		return r.applyNotSpecification(db, s.Spec)
	default:
		return r.applyConcreteSpecification(db, spec)
	}
}
func (r *UserRepository) applyNotSpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	switch s := spec.(type) {
	case user.ByEmailSpecification:
		return db.Where("email != ?", s.Email)
	case user.ByStatusSpecification:
		return db.Where("is_active != ?", s.Active)
	case user.ByAgeRangeSpecification:
		if s.Min > 0 && s.Max > 0 {
			return db.Where("age < ? OR age > ?", s.Min, s.Max)
		} else if s.Min > 0 {
			return db.Where("age < ?", s.Min)
		} else if s.Max > 0 {
			return db.Where("age > ?", s.Max)
		}
		return db
	case shared.AndSpecification[*user.User]:
		leftSpec := shared.Not(s.Left)
		rightSpec := shared.Not(s.Right)
		return r.applySpecification(db, shared.Or(leftSpec, rightSpec))
	case shared.OrSpecification[*user.User]:
		leftSpec := shared.Not(s.Left)
		rightSpec := shared.Not(s.Right)
		return r.applySpecification(db, shared.And(leftSpec, rightSpec))
	case shared.NotSpecification[*user.User]:
		return r.applySpecification(db, s.Spec)
	default:
		return db
	}
}
func (r *UserRepository) applyConcreteSpecification(db *gorm.DB, spec shared.Specification[*user.User]) *gorm.DB {
	switch s := spec.(type) {
	case user.ByEmailSpecification:
		return db.Where("email = ?", s.Email)
	case user.ByStatusSpecification:
		return db.Where("is_active = ?", s.Active)
	case user.ByAgeRangeSpecification:
		if s.Min > 0 {
			db = db.Where("age >= ?", s.Min)
		}
		if s.Max > 0 {
			db = db.Where("age <= ?", s.Max)
		}
		return db
	default:
		return db
	}
}
func (r *UserRepository) Remove(ctx context.Context, id string) error {
	result := r.getDB(ctx).
		Model(&po.UserPO{}).
		Where("id = ?", id).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return user.NewUserNotFoundError(id)
	}

	return nil
}

var _ user.Repository = (*UserRepository)(nil)
