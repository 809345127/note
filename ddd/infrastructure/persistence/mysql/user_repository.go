package mysql

import (
	"context"
	"errors"

	"ddd-example/domain/user"
	"ddd-example/infrastructure/persistence/mysql/po"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository 用户仓储的MySQL/GORM实现
// DDD原则：仓储只负责聚合根的持久化，不负责发布事件
// GORM使用规范：禁止使用关联功能，保持DDD聚合边界
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// NextIdentity 生成新的用户ID
func (r *UserRepository) NextIdentity() string {
	return "user-" + uuid.New().String()
}

// Save 保存用户（创建或更新）
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	userPO := po.FromUserDomain(u)

	// 使用 Save 方法，GORM 会根据主键判断是 Create 还是 Update
	result := r.db.WithContext(ctx).Save(userPO)
	return result.Error
}

// FindByID 根据ID查找用户
func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	var userPO po.UserPO

	result := r.db.WithContext(ctx).First(&userPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}

// FindByEmail 根据邮箱查找用户
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var userPO po.UserPO

	result := r.db.WithContext(ctx).First(&userPO, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}

	return userPO.ToDomain(), nil
}

// Remove 删除用户（逻辑删除：标记为不活跃）
// DDD原则：推荐逻辑删除而非物理删除，保留业务历史
func (r *UserRepository) Remove(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
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

// 编译时检查接口实现
var _ user.Repository = (*UserRepository)(nil)
