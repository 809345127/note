package mysql

import (
	"context"
	"database/sql"
	"ddd-example/domain"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// OrderRepository MySQL订单仓储实现
// DDD原则：仓储负责聚合根的持久化，并在保存后发布领域事件
type OrderRepository struct {
	db             *sql.DB
	eventPublisher domain.DomainEventPublisher
}

// NewOrderRepository 创建订单仓储
// eventPublisher: 事件发布器，仓储在Save后发布聚合根产生的事件
func NewOrderRepository(db *sql.DB, eventPublisher domain.DomainEventPublisher) *OrderRepository {
	return &OrderRepository{
		db:             db,
		eventPublisher: eventPublisher,
	}
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	// 查询订单基本信息
	orderRow := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, total_amount, total_currency, status, version, created_at, updated_at
		FROM orders
		WHERE id = ?
	`, id)

	var orderID, userID string
	var totalAmountValue int64
	var totalCurrency string
	var status string
	var version int
	var createdAt, updatedAt time.Time

	err := orderRow.Scan(
		&orderID, &userID, &totalAmountValue, &totalCurrency, &status, &version,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %w", domain.ErrOrderNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan order: %w", err)
	}

	// 解析订单状态
	orderStatus := domain.OrderStatus(status)
	totalAmount := domain.NewMoney(totalAmountValue, totalCurrency)

	// 查询订单项并构建items
	items, err := r.findOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to find order items: %w", err)
	}

	// 使用DTO重建Order
	dto := domain.OrderReconstructionDTO{
		ID:          orderID,
		UserID:      userID,
		Items:       items,
		TotalAmount: *totalAmount,
		Status:      orderStatus,
		Version:     version,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	return domain.RebuildOrderFromDTO(dto), nil
}

func (r *OrderRepository) findOrderItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	itemsRows, err := r.db.QueryContext(ctx, `
		SELECT id, product_id, product_name, quantity, unit_price, unit_currency, subtotal, subtotal_currency
		FROM order_items
		WHERE order_id = ?
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer itemsRows.Close()

	var items []domain.OrderItem
	for itemsRows.Next() {
		var itemID, productID, productName string
		var quantity int
		var unitPriceValue, subtotalValue int64
		var unitPriceCurrency, subtotalCurrency string

		err := itemsRows.Scan(
			&itemID, &productID, &productName, &quantity,
			&unitPriceValue, &unitPriceCurrency, &subtotalValue, &subtotalCurrency,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}

		unitPrice := domain.NewMoney(unitPriceValue, unitPriceCurrency)
		subtotal := domain.NewMoney(subtotalValue, subtotalCurrency)

		itemDTO := domain.OrderItemReconstructionDTO{
			ID:          itemID,
			ProductID:   productID,
			ProductName: productName,
			Quantity:    quantity,
			UnitPrice:   *unitPrice,
			Subtotal:    *subtotal,
		}

		items = append(items, domain.RebuildOrderItemFromDTO(itemDTO))
	}

	return items, nil
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

	var orders []*domain.Order
	for rows.Next() {
		var orderID, userID string
		var totalAmountValue int64
		var totalCurrency string
		var status string
		var version int
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&orderID, &userID, &totalAmountValue, &totalCurrency, &status, &version,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		orderStatus := domain.OrderStatus(status)
		totalAmount := domain.NewMoney(totalAmountValue, totalCurrency)

		// 加载订单项
		items, err := r.findOrderItems(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to load order items: %w", err)
		}

		dto := domain.OrderReconstructionDTO{
			ID:          orderID,
			UserID:      userID,
			Items:       items,
			TotalAmount: *totalAmount,
			Status:      orderStatus,
			Version:     version,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		orders = append(orders, domain.RebuildOrderFromDTO(dto))
	}

	return orders, nil
}

func (r *OrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, total_amount, total_currency, status, version, created_at, updated_at
		FROM orders
		WHERE user_id = ? AND status = ?
		ORDER BY created_at DESC
	`, userID, string(domain.OrderStatusDelivered))
	if err != nil {
		return nil, fmt.Errorf("failed to query delivered orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var orderID, userID string
		var totalAmountValue int64
		var totalCurrency string
		var status string
		var version int
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&orderID, &userID, &totalAmountValue, &totalCurrency, &status, &version,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		orderStatus := domain.OrderStatus(status)
		totalAmount := domain.NewMoney(totalAmountValue, totalCurrency)

		// 加载订单项
		items, err := r.findOrderItems(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to load order items: %w", err)
		}

		dto := domain.OrderReconstructionDTO{
			ID:          orderID,
			UserID:      userID,
			Items:       items,
			TotalAmount: *totalAmount,
			Status:      orderStatus,
			Version:     version,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}

		orders = append(orders, domain.RebuildOrderFromDTO(dto))
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
		string(order.Status()), order.Version(), order.CreatedAt(), order.UpdatedAt())
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
		unitPrice := item.UnitPrice()
		subtotal := item.Subtotal()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (id, order_id, product_id, product_name, quantity, 
				unit_price, unit_currency, subtotal, subtotal_currency)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.ID(), order.ID(), item.ProductID(), item.ProductName(), item.Quantity(),
			unitPrice.Amount(), unitPrice.Currency(),
			subtotal.Amount(), subtotal.Currency())
		if err != nil {
			return fmt.Errorf("failed to save order item: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// DDD原则：仓储在保存成功后发布聚合根产生的领域事件
	r.publishEvents(order.PullEvents())

	return nil
}

// publishEvents 发布领域事件
// 事件发布失败不影响主流程（最终一致性）
func (r *OrderRepository) publishEvents(events []domain.DomainEvent) {
	if r.eventPublisher == nil {
		return
	}
	for _, event := range events {
		if err := r.eventPublisher.Publish(event); err != nil {
			log.Printf("[WARN] Failed to publish event %s: %v", event.EventName(), err)
		}
	}
}

func (r *OrderRepository) NextIdentity() string {
	return uuid.New().String()
}

// Remove 逻辑删除订单（标记为已取消）
// DDD原则：推荐逻辑删除而非物理删除，保留业务历史
func (r *OrderRepository) Remove(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = ?, updated_at = NOW() WHERE id = ?
	`, string(domain.OrderStatusCancelled), id)
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
