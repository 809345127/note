package mysql

import (
	"context"
	"database/sql"
	"ddd-example/domain"
	"fmt"
	"time"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, age, is_active, version, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id)

	var user domain.User
	var email string
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&user.id, &user.name, &email, &user.age,
		&user.isActive, &user.version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %w", domain.ErrUserNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	user.email = domain.Email{value: email}
	user.createdAt = createdAt
	user.updatedAt = updatedAt
	user.events = []domain.DomainEvent{}

	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, email, age, is_active, version, created_at, updated_at
		FROM users
		WHERE email = ?
	`, email.Value())

	var user domain.User
	var emailStr string
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&user.id, &user.name, &emailStr, &user.age,
		&user.isActive, &user.version, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %w", domain.ErrUserNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	user.email = domain.Email{value: emailStr}
	user.createdAt = createdAt
	user.updatedAt = updatedAt
	user.events = []domain.DomainEvent{}

	return &user, nil
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	// 保存前移除所有事件，由UnitOfWork负责发布
	events := user.PullEvents()
	
	result, err := r.db.ExecContext(ctx, `
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
		user.ID(), user.Name(), user.Email().Value(), user.Age(),
		user.IsActive(), user.Version(), user.CreatedAt(), user.UpdatedAt())

	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		// 乐观锁冲突
		return fmt.Errorf("optimistic lock conflict: %w", domain.ErrOptimisticLockConflict)
	}

	// 恢复事件（由UnitOfWork负责发布）
	for _, event := range events {
		user.RecordEvent(event)
	}

	return nil
}

func (r *UserRepository) NextIdentity() string {
	return domain.NewUUID()
}

func (r *UserRepository) Remove(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
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
