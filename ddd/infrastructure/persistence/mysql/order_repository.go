package mysql

import (
	"context"
	"errors"

	"ddd-example/domain/order"
	"ddd-example/infrastructure/persistence/mysql/po"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderRepository MySQL/GORM implementation of order repository
// DDD principle: Repository is only responsible for persistence of aggregate roots, not event publishing
// GORM usage specification: Association features are prohibited to maintain DDD aggregate boundaries
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository Create order repository
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// NextIdentity Generate new order ID
func (r *OrderRepository) NextIdentity() string {
	return "order-" + uuid.New().String()
}

// Save Save order (create or update)
// Note: Manually manage saving of orders and order items, do not use GORM associations
func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	orderPO, itemPOs := po.FromOrderDomain(o)

	// Use transaction to ensure atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Save order
		if err := tx.Save(orderPO).Error; err != nil {
			return err
		}

		// Delete old order items (simple strategy: delete then insert)
		if err := tx.Where("order_id = ?", o.ID()).Delete(&po.OrderItemPO{}).Error; err != nil {
			return err
		}

		// Save new order items
		if len(itemPOs) > 0 {
			if err := tx.Create(&itemPOs).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// FindByID Find order by ID
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	var orderPO po.OrderPO

	// Query order
	result := r.db.WithContext(ctx).First(&orderPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, result.Error
	}

	// Manually query order items (do not use GORM's Preload to keep aggregate boundaries clear)
	var itemPOs []po.OrderItemPO
	if err := r.db.WithContext(ctx).Where("order_id = ?", id).Find(&itemPOs).Error; err != nil {
		return nil, err
	}

	return orderPO.ToDomain(itemPOs), nil
}

// FindByUserID Find order list by user ID
func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	var orderPOs []po.OrderPO

	// Query orders
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orderPOs).Error; err != nil {
		return nil, err
	}

	// Batch query order items
	orders := make([]*order.Order, len(orderPOs))
	for i, orderPO := range orderPOs {
		var itemPOs []po.OrderItemPO
		if err := r.db.WithContext(ctx).Where("order_id = ?", orderPO.ID).Find(&itemPOs).Error; err != nil {
			return nil, err
		}
		orders[i] = orderPO.ToDomain(itemPOs)
	}

	return orders, nil
}

// FindDeliveredOrdersByUserID Find delivered orders by user ID
func (r *OrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	var orderPOs []po.OrderPO

	// Query delivered orders
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, string(order.StatusDelivered)).
		Order("created_at DESC").
		Find(&orderPOs).Error; err != nil {
		return nil, err
	}

	// Batch query order items
	orders := make([]*order.Order, len(orderPOs))
	for i, orderPO := range orderPOs {
		var itemPOs []po.OrderItemPO
		if err := r.db.WithContext(ctx).Where("order_id = ?", orderPO.ID).Find(&itemPOs).Error; err != nil {
			return nil, err
		}
		orders[i] = orderPO.ToDomain(itemPOs)
	}

	return orders, nil
}

// Remove Delete order (logical deletion: mark as cancelled)
// DDD principle: Logical deletion is recommended over physical deletion to preserve business history
func (r *OrderRepository) Remove(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&po.OrderPO{}).
		Where("id = ?", id).
		Update("status", string(order.StatusCancelled))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("order not found")
	}

	return nil
}

// Compile-time interface implementation check
var _ order.Repository = (*OrderRepository)(nil)
