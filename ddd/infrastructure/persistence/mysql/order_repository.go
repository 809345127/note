package mysql

import (
	"context"
	"errors"

	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/infrastructure/persistence"
	"ddd/infrastructure/persistence/mysql/po"

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

// getDB returns the transaction from context if available, otherwise the default db
func (r *OrderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

// Save Save order (create or update)
// Note: Manually manage saving of orders and order items, do not use GORM associations
// When called within UoW.Execute(), it uses the transaction from context
// When called standalone, it creates its own transaction for atomicity
// Uses dirty tracking for efficient updates and optimistic locking for concurrency control
func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	// Check if we're already in a UoW transaction
	if tx := persistence.TxFromContext(ctx); tx != nil {
		// Use the existing transaction from UoW
		return r.saveWithTx(tx, o)
	}

	// No UoW transaction - create our own for atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.saveWithTx(tx, o)
	})
}

// saveWithTx performs the actual save operations within a transaction
// Uses dirty tracking: only inserts new items and deletes removed items
// Uses optimistic locking: checks version to prevent concurrent modification
func (r *OrderRepository) saveWithTx(tx *gorm.DB, o *order.Order) error {
	orderPO, allItemPOs := po.FromOrderDomain(o)

	if o.IsNew() {
		// New aggregate: insert order and all items
		if err := tx.Create(orderPO).Error; err != nil {
			return err
		}
		if len(allItemPOs) > 0 {
			if err := tx.Create(&allItemPOs).Error; err != nil {
				return err
			}
		}
	} else {
		// Existing aggregate: use optimistic locking and dirty tracking

		// 1. Query current version from database to ensure we use the correct version for WHERE clause
		// DDD Principle: The repository is responsible for version synchronization between
		// the domain model and the persistence layer
		var currentOrderPO po.OrderPO
		if err := tx.First(&currentOrderPO, "id = ?", o.ID()).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return order.ErrOrderNotFound
			}
			return err
		}
		dbVersion := currentOrderPO.Version

		// 2. Update order with optimistic lock check using database version
		result := tx.Model(&po.OrderPO{}).
			Where("id = ? AND version = ?", o.ID(), dbVersion).
			Updates(map[string]any{
				"status":         orderPO.Status,
				"total_amount":   orderPO.TotalAmount,
				"total_currency": orderPO.TotalCurrency,
				"version":        dbVersion + 1,
				"updated_at":     orderPO.UpdatedAt,
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return order.ErrConcurrentModification
		}

		// 3. Notify the aggregate that persistence was successful
		// DDD Principle: The aggregate controls its own version increment, triggered by persistence
		o.IncrementVersionForSave()

		// 4. Delete removed items (dirty tracking)
		for _, item := range o.RemovedItems() {
			if err := tx.Delete(&po.OrderItemPO{}, "id = ?", item.ID()).Error; err != nil {
				return err
			}
		}

		// 5. Insert added items (dirty tracking)
		for _, item := range o.AddedItems() {
			itemPO := po.OrderItemPO{
				ID:               o.ID() + "-" + item.ProductID(),
				OrderID:          o.ID(),
				ProductID:        item.ProductID(),
				ProductName:      item.ProductName(),
				Quantity:         item.Quantity(),
				UnitPrice:        item.UnitPrice().Amount(),
				UnitCurrency:     item.UnitPrice().Currency(),
				Subtotal:         item.Subtotal().Amount(),
				SubtotalCurrency: item.Subtotal().Currency(),
			}
			if err := tx.Create(&itemPO).Error; err != nil {
				return err
			}
		}
	}

	// Clear dirty tracking after successful save
	o.ClearDirtyTracking()
	return nil
}

// FindByID Find order by ID
// 错误处理:
//   - 订单未找到: 返回 order.NewOrderNotFoundError（领域错误，带堆栈）
//   - 其他数据库错误: 原样返回（基础设施错误，已包含足够信息）
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	db := r.getDB(ctx)
	var orderPO po.OrderPO

	// Query order
	result := db.First(&orderPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 领域错误: 使用带堆栈的构造函数
			// 堆栈会从这里开始捕获: FindByID -> NewOrderNotFoundError
			return nil, order.NewOrderNotFoundError(id)
		}
		// 基础设施错误: 直接返回，GORM 错误已包含 SQL、连接等信息
		return nil, result.Error
	}

	// Manually query order items (do not use GORM's Preload to keep aggregate boundaries clear)
	var itemPOs []po.OrderItemPO
	if err := db.Where("order_id = ?", id).Find(&itemPOs).Error; err != nil {
		return nil, err
	}

	// Rebuild to ensure a fresh domain object (important for retry scenarios)
	o := orderPO.ToDomain(itemPOs)
	return o, nil
}

// FindByUserID Find order list by user ID
func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	spec := order.ByUserIDSpecification{UserID: userID}
	return r.FindBySpecification(ctx, spec)
}

// FindDeliveredOrdersByUserID Find delivered orders by user ID
func (r *OrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	spec := shared.And(
		order.ByUserIDSpecification{UserID: userID},
		order.ByStatusSpecification{Status: order.StatusDelivered},
	)
	return r.FindBySpecification(ctx, spec)
}

// FindBySpecification Find orders by specification
// Implements the domain.Repository interface for flexible query composition
func (r *OrderRepository) FindBySpecification(ctx context.Context, spec shared.Specification[*order.Order]) ([]*order.Order, error) {
	db := r.getDB(ctx)

	// Apply specification to query
	db = r.applySpecification(db, spec)
	if db.Error != nil {
		return nil, db.Error
	}

	// Execute query with ordering
	var orderPOs []po.OrderPO
	if err := db.Order("created_at DESC").Find(&orderPOs).Error; err != nil {
		return nil, err
	}

	// Batch query order items
	orders := make([]*order.Order, len(orderPOs))
	for i, orderPO := range orderPOs {
		var itemPOs []po.OrderItemPO
		if err := db.Where("order_id = ?", orderPO.ID).Find(&itemPOs).Error; err != nil {
			return nil, err
		}
		orders[i] = orderPO.ToDomain(itemPOs)
	}

	return orders, nil
}

// applySpecification applies a domain specification to a GORM query
// Uses type switches to handle different specification types
func (r *OrderRepository) applySpecification(db *gorm.DB, spec shared.Specification[*order.Order]) *gorm.DB {
	if spec == nil {
		return db
	}

	// Handle composite specifications
	switch s := spec.(type) {
	case shared.AndSpecification[*order.Order]:
		return r.applySpecification(r.applySpecification(db, s.Left), s.Right)
	// Note: OR and NOT specifications are more complex to implement with GORM
	// For simplicity in this first implementation, we only support AND
	default:
		return r.applyConcreteSpecification(db, spec)
	}
}

// applyConcreteSpecification applies concrete domain specifications
func (r *OrderRepository) applyConcreteSpecification(db *gorm.DB, spec shared.Specification[*order.Order]) *gorm.DB {
	switch s := spec.(type) {
	case order.ByUserIDSpecification:
		return db.Where("user_id = ?", s.UserID)
	case order.ByStatusSpecification:
		return db.Where("status = ?", s.Status)
	case order.ByDateRangeSpecification:
		// Handle optional start and end dates
		if !s.Start.IsZero() {
			db = db.Where("created_at >= ?", s.Start)
		}
		if !s.End.IsZero() {
			db = db.Where("created_at <= ?", s.End)
		}
		return db
	default:
		// Unknown specification type - return unchanged
		return db
	}
}

// Remove Delete order (logical deletion: mark as cancelled)
// DDD principle: Logical deletion is recommended over physical deletion to preserve business history
func (r *OrderRepository) Remove(ctx context.Context, id string) error {
	result := r.getDB(ctx).
		Model(&po.OrderPO{}).
		Where("id = ?", id).
		Update("status", string(order.StatusCancelled))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return order.ErrOrderNotFound
	}

	return nil
}

// Compile-time interface implementation check
var _ order.Repository = (*OrderRepository)(nil)
