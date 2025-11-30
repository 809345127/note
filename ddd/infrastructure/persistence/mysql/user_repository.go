package mysql

import (
	"context"
	"database/sql"
	"ddd-example/domain"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserRepository MySQL用户仓储实现
// DDD原则：仓储只负责聚合根的持久化，不负责发布事件
// 事件发布由 UoW 保存到 outbox 表，后台服务异步发布
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建用户仓储
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, age, is_active, version, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id)

	var userID, name, email string
	var age int
	var isActive bool
	var version int
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&userID, &name, &email, &age,
		&isActive, &version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %w", domain.ErrUserNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	// 使用DTO重建User
	dto := domain.UserReconstructionDTO{
		ID:        userID,
		Name:      name,
		Email:     email,
		Age:       age,
		IsActive:  isActive,
		Version:   version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return domain.RebuildUserFromDTO(dto), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.db.QueryContext(ctx, `
		SELECT id, name, email, age, is_active, version, created_at, updated_at
		FROM users
		WHERE email = ?
		LIMIT 1
	`, email)
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, fmt.Errorf("user not found: %w", domain.ErrUserNotFound)
	}

	var userID, name, emailStr string
	var age int
	var isActive bool
	var version int
	var createdAt, updatedAt time.Time

	err = row.Scan(
		&userID, &name, &emailStr, &age,
		&isActive, &version, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	// 使用DTO重建User
	dto := domain.UserReconstructionDTO{
		ID:        userID,
		Name:      name,
		Email:     emailStr,
		Age:       age,
		IsActive:  isActive,
		Version:   version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return domain.RebuildUserFromDTO(dto), nil
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	email := user.Email()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, name, email, age, is_active, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			email = VALUES(email),
			age = VALUES(age),
			is_active = VALUES(is_active),
			version = VALUES(version) + 1,
			updated_at = VALUES(updated_at)
	`,
		user.ID(), user.Name(), email.Value(), user.Age(),
		user.IsActive(), user.Version(), user.CreatedAt(), user.UpdatedAt())

	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	// 注意：不在仓储中发布事件！
	// 事件由 UoW 在事务提交前保存到 outbox 表
	// 后台 OutboxProcessor 异步发布到消息队列

	return nil
}

func (r *UserRepository) NextIdentity() string {
	return uuid.New().String()
}

// Remove 逻辑删除用户
// DDD原则：推荐逻辑删除而非物理删除，保留业务历史
func (r *UserRepository) Remove(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE users SET is_active = false, updated_at = NOW() WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("user not found: %w", domain.ErrUserNotFound)
	}

	return nil
}
