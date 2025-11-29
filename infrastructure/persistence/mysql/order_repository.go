package mysql

import (
	"context"
	"database/sql"
	"ddd-example/domain"
	"fmt"
	"time"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	// 查询订单基本信息
	orderRow := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, total_amount, total_currency, status, version, created_at, updated_at
		FROM orders
		WHERE id = ?
	`, id)

	var order domain.Order
	var totalAmount, unitPrice, subtotal int64
	var totalCurrency, status string
	var createdAt, updatedAt time.Time
	var version int

	err := orderRow.Scan(
		&order.id, &order.userID, &totalAmount, &totalCurrency, &status, &version,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %w", domain.ErrOrderNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan order: %w", err)
	}

	// 设置值
	order.totalAmount = domain.NewMoney(totalAmount, totalCurrency)
	order.status = domain.ParseOrderStatus(status)
	order.version = version
	order.createdAt = createdAt
	order.updatedAt = updatedAt

	// 查询订单项
	itemsRows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, product_name, quantity, unit_price, unit_currency, subtotal, subtotal_currency
		FROM order_items
		WHERE order_id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer itemsRows.Close()

	items := []domain.OrderItem{}
	for itemsRows.Next() {
		var item domain.OrderItem
		var unitPrice, subtotal int64
		var unitCurrency, subtotalCurrency string

		err := itemsRows.Scan(
			&item.id, &item.productID, &item.productName, &item.quantity,
			&unitPrice, &unitCurrency, &subtotal, &subtotalCurrency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}

		item.unitPrice = domain.NewMoney(unitPrice, unitCurrency)
		item.subtotal = domain.NewMoney(subtotal, subtotalCurrency)

		items = append(items, item)
	}

	order.items = items
	order.events = []domain.DomainEvent{}

	return &order, nil
}

func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, total_amount, total_currency, status, version, created_at, updated_at
		FROM orders
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	orders := []*domain.Order{}
	for rows.Next() {
		var order domain.Order
		var totalAmount int64
		var totalCurrency, status string
		var createdAt, updatedAt time.Time
		var version int

		err := rows.Scan(
			&order.id, &order.userID, &totalAmount, &totalCurrency, &status, &version,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		order.totalAmount = domain.NewMoney(totalAmount, totalCurrency)
		order.status = domain.ParseOrderStatus(status)
		order.version = version
		order.createdAt = createdAt
		order.updatedAt = updatedAt

		orders = append(orders, &order)
	}

	// Load order items for all orders
	for _, order := range orders {
		itemsRows, err := r.db.QueryContext(ctx, `
			SELECT id, product_id, product_name, quantity, unit_price, unit_currency, subtotal, subtotal_currency
			FROM order_items
			WHERE order_id = ?
		`, order.ID())
		if err != nil {
			return nil, fmt.Errorf("failed to query order items: %w", err)
		}
		defer itemsRows.Close()

		items := []domain.OrderItem{}
		for itemsRows.Next() {
			var item domain.OrderItem
			var unitPrice, subtotal int64
			var unitCurrency, subtotalCurrency string

			err := itemsRows.Scan(
				&item.id, &item.productID, &item.productName, &item.quantity,
				&unitPrice, &unitCurrency, &subtotal, &subtotalCurrency,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan order item: %w", err)
			}

			item.unitPrice = domain.NewMoney(unitPrice, unitCurrency)
			item.subtotal = domain.NewMoney(subtotal, subtotalCurrency)

			items = append(items, item)
		}

		order.items = items
		order.events = []domain.DomainEvent{}
	}

	return orders, nil
}

func (r *OrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, total_amount, total_currency, status, version, created_at, updated_at
		FROM orders
		WHERE user_id = ? AND status = ?
		ORDER BY created_at DESC
	`, userID, domain.OrderStatusDelivered.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query delivered orders: %w", err)
	}
	defer rows.Close()

	orders := []*domain.Order{}
	for rows.Next() {
		var order domain.Order
		var totalAmount int64
		var totalCurrency string
		var createdAt, updatedAt time.Time
		var version int

		err := rows.Scan(
			&order.id, &order.userID, &totalAmount, &totalCurrency, &order.status, &version,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		order.totalAmount = domain.NewMoney(totalAmount, totalCurrency)
		order.version = version
		order.createdAt = createdAt
		order.updatedAt = updatedAt

		orders = append(orders, &order)
	}

	// Load order items for all orders
	for _, order := range orders {
		itemsRows, err := r.db.QueryContext(ctx, `
			SELECT id, product_id, product_name, quantity, unit_price, unit_currency, subtotal, subtotal_currency
			FROM order_items
			WHERE order_id = ?
		`, order.ID())
		if err != nil {
			return nil, fmt.Errorf("failed to query order items: %w", err)
		}
		defer itemsRows.Close()

		items := []domain.OrderItem{}
		for itemsRows.Next() {
			var item domain.OrderItem
			var unitPrice, subtotal int64
			var unitCurrency, subtotalCurrency string

			err := itemsRows.Scan(
				&item.id, &item.productID, &item.productName, &item.quantity,
				&unitPrice, &unitCurrency, &subtotal, &subtotalCurrency,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan order item: %w", err)
			}

			item.unitPrice = domain.NewMoney(unitPrice, unitCurrency)
			item.subtotal = domain.NewMoney(subtotal, subtotalCurrency)

			items = append(items, item)
		}

		order.items = items
		order.events = []domain.DomainEvent{}
	}

	return orders, nil
}

func (r *OrderRepository) Save(ctx context.Context, order *domain.Order) error {
	// 开始事务
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 保存订单基本信息
	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, user_id, total_amount, total_currency, status, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			total_amount = VALUES(total_amount),
			total_currency = VALUES(total_currency),
			status = VALUES(status),
			version = VALUES(version) + 1,
			updated_at = VALUES(updated_at)
	`,
		order.ID(), order.UserID(), order.TotalAmount().Amount(), order.TotalAmount().Currency(),
		order.Status().String(), order.Version(), order.CreatedAt(), order.UpdatedAt())
	if err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	// 删除旧的订单项（如果是更新操作）
	_, err = tx.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = ?`, order.ID())
	if err != nil {
		return fmt.Errorf("failed to delete order items: %w", err)
	}

	// 保存新的订单项
	for _, item := range order.Items() {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (id, order_id, product_id, product_name, quantity, 
				unit_price, unit_currency, subtotal, subtotal_currency)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID(), order.ID(), item.ProductID(), item.ProductName(), item.Quantity(),
			item.UnitPrice().Amount(), item.UnitPrice().Currency(),
			item.Subtotal().Amount(), item.Subtotal().Currency())
		if err != nil {
			return fmt.Errorf("failed to save order item: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *OrderRepository) NextIdentity() string {
	return domain.NewUUID()
}

func (r *OrderRepository) Remove(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to remove order: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("order not found: %w", domain.ErrOrderNotFound)
	}

	return nil
}
