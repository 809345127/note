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

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	if tx := persistence.TxFromContext(ctx); tx != nil {
		return r.saveWithTx(tx, o)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.saveWithTx(tx, o)
	})
}

func (r *OrderRepository) saveWithTx(tx *gorm.DB, o *order.Order) error {
	orderPO, allItemPOs := po.FromOrderDomain(o)

	if o.IsNew() {
		if err := tx.Create(orderPO).Error; err != nil {
			return err
		}
		if len(allItemPOs) > 0 {
			if err := tx.Create(&allItemPOs).Error; err != nil {
				return err
			}
		}
	} else {
		expectedVersion := o.Version()
		result := tx.Model(&po.OrderPO{}).
			Where("id = ? AND version = ?", o.ID(), expectedVersion).
			Updates(map[string]any{
				"status":         orderPO.Status,
				"total_amount":   orderPO.TotalAmount,
				"total_currency": orderPO.TotalCurrency,
				"version":        expectedVersion + 1,
				"updated_at":     orderPO.UpdatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var count int64
			if err := tx.Model(&po.OrderPO{}).Where("id = ?", o.ID()).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return order.NewOrderNotFoundError(o.ID())
			}
			return order.NewConcurrentModificationError(o.ID())
		}

		o.IncrementVersionForSave()

		for _, item := range o.RemovedItems() {
			if err := tx.Delete(&po.OrderItemPO{}, "id = ?", item.ID()).Error; err != nil {
				return err
			}
		}

		for _, item := range o.AddedItems() {
			itemPO := po.OrderItemPO{
				ID:               item.ID(),
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

	o.ClearDirtyTracking()
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	db := r.getDB(ctx)
	var orderPO po.OrderPO
	result := db.First(&orderPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, order.NewOrderNotFoundError(id)
		}
		return nil, result.Error
	}

	var itemPOs []po.OrderItemPO
	if err := db.Where("order_id = ?", id).Find(&itemPOs).Error; err != nil {
		return nil, err
	}

	return orderPO.ToDomain(itemPOs), nil
}

func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	spec := order.ByUserIDSpecification{UserID: userID}
	return r.FindBySpecification(ctx, spec)
}

func (r *OrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	spec := shared.And(
		order.ByUserIDSpecification{UserID: userID},
		order.ByStatusSpecification{Status: order.StatusDelivered},
	)
	return r.FindBySpecification(ctx, spec)
}

func (r *OrderRepository) FindBySpecification(ctx context.Context, spec shared.Specification[*order.Order]) ([]*order.Order, error) {
	baseDB := r.getDB(ctx)
	db := r.applySpecification(baseDB, spec)
	if db.Error != nil {
		return nil, db.Error
	}

	var orderPOs []po.OrderPO
	if err := db.Order("created_at DESC").Find(&orderPOs).Error; err != nil {
		return nil, err
	}
	if len(orderPOs) == 0 {
		return []*order.Order{}, nil
	}

	orderIDs := make([]string, 0, len(orderPOs))
	for _, orderPO := range orderPOs {
		orderIDs = append(orderIDs, orderPO.ID)
	}

	var itemPOs []po.OrderItemPO
	if err := baseDB.Model(&po.OrderItemPO{}).Where("order_id IN ?", orderIDs).Find(&itemPOs).Error; err != nil {
		return nil, err
	}

	itemMap := make(map[string][]po.OrderItemPO, len(orderPOs))
	for _, itemPO := range itemPOs {
		itemMap[itemPO.OrderID] = append(itemMap[itemPO.OrderID], itemPO)
	}

	orders := make([]*order.Order, len(orderPOs))
	for i, orderPO := range orderPOs {
		orders[i] = orderPO.ToDomain(itemMap[orderPO.ID])
	}
	return orders, nil
}

func (r *OrderRepository) applySpecification(db *gorm.DB, spec shared.Specification[*order.Order]) *gorm.DB {
	if spec == nil {
		return db
	}

	switch s := spec.(type) {
	case shared.AndSpecification[*order.Order]:
		return r.applySpecification(r.applySpecification(db, s.Left), s.Right)
	case shared.OrSpecification[*order.Order]:
		leftDB := r.applySpecification(db, s.Left)
		return leftDB.Or(r.applySpecification(db.Session(&gorm.Session{}), s.Right))
	case shared.NotSpecification[*order.Order]:
		return r.applyNotSpecification(db, s.Spec)
	default:
		return r.applyConcreteSpecification(db, spec)
	}
}

func (r *OrderRepository) applyNotSpecification(db *gorm.DB, spec shared.Specification[*order.Order]) *gorm.DB {
	switch s := spec.(type) {
	case order.ByUserIDSpecification:
		return db.Where("user_id != ?", s.UserID)
	case order.ByStatusSpecification:
		return db.Where("status != ?", s.Status)
	case order.ByDateRangeSpecification:
		if !s.Start.IsZero() && !s.End.IsZero() {
			return db.Where("created_at < ? OR created_at > ?", s.Start, s.End)
		}
		if !s.Start.IsZero() {
			return db.Where("created_at < ?", s.Start)
		}
		if !s.End.IsZero() {
			return db.Where("created_at > ?", s.End)
		}
		return db
	case shared.AndSpecification[*order.Order]:
		leftSpec := shared.Not(s.Left)
		rightSpec := shared.Not(s.Right)
		return r.applySpecification(db, shared.Or(leftSpec, rightSpec))
	case shared.OrSpecification[*order.Order]:
		leftSpec := shared.Not(s.Left)
		rightSpec := shared.Not(s.Right)
		return r.applySpecification(db, shared.And(leftSpec, rightSpec))
	case shared.NotSpecification[*order.Order]:
		return r.applySpecification(db, s.Spec)
	default:
		return db
	}
}

func (r *OrderRepository) applyConcreteSpecification(db *gorm.DB, spec shared.Specification[*order.Order]) *gorm.DB {
	switch s := spec.(type) {
	case order.ByUserIDSpecification:
		return db.Where("user_id = ?", s.UserID)
	case order.ByStatusSpecification:
		return db.Where("status = ?", s.Status)
	case order.ByDateRangeSpecification:
		if !s.Start.IsZero() {
			db = db.Where("created_at >= ?", s.Start)
		}
		if !s.End.IsZero() {
			db = db.Where("created_at <= ?", s.End)
		}
		return db
	default:
		return db
	}
}

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

var _ order.Repository = (*OrderRepository)(nil)
