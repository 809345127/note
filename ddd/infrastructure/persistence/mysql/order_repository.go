package mysql

import (
	"context"
	"errors"

	"ddd-example/domain/order"
	"ddd-example/infrastructure/persistence/mysql/po"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderRepository 订单仓储的MySQL/GORM实现
// DDD原则：仓储只负责聚合根的持久化，不负责发布事件
// GORM使用规范：禁止使用关联功能，保持DDD聚合边界
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓储
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// NextIdentity 生成新的订单ID
func (r *OrderRepository) NextIdentity() string {
	return "order-" + uuid.New().String()
}

// Save 保存订单（创建或更新）
// 注意：手动管理订单和订单项的保存，不使用GORM关联
func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	orderPO, itemPOs := po.FromOrderDomain(o)

	// 使用事务保证原子性
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 保存订单
		if err := tx.Save(orderPO).Error; err != nil {
			return err
		}

		// 删除旧的订单项（简单策略：先删后增）
		if err := tx.Where("order_id = ?", o.ID()).Delete(&po.OrderItemPO{}).Error; err != nil {
			return err
		}

		// 保存新的订单项
		if len(itemPOs) > 0 {
			if err := tx.Create(&itemPOs).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// FindByID 根据ID查找订单
func (r *OrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	var orderPO po.OrderPO

	// 查询订单
	result := r.db.WithContext(ctx).First(&orderPO, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, result.Error
	}

	// 手动查询订单项（不使用GORM的Preload，保持聚合边界清晰）
	var itemPOs []po.OrderItemPO
	if err := r.db.WithContext(ctx).Where("order_id = ?", id).Find(&itemPOs).Error; err != nil {
		return nil, err
	}

	return orderPO.ToDomain(itemPOs), nil
}

// FindByUserID 根据用户ID查找订单列表
func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	var orderPOs []po.OrderPO

	// 查询订单
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orderPOs).Error; err != nil {
		return nil, err
	}

	// 批量查询订单项
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

// FindDeliveredOrdersByUserID 查找用户已送达的订单
func (r *OrderRepository) FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*order.Order, error) {
	var orderPOs []po.OrderPO

	// 查询已送达的订单
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, string(order.StatusDelivered)).
		Order("created_at DESC").
		Find(&orderPOs).Error; err != nil {
		return nil, err
	}

	// 批量查询订单项
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

// Remove 删除订单（逻辑删除：标记为已取消）
// DDD原则：推荐逻辑删除而非物理删除，保留业务历史
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

// 编译时检查接口实现
var _ order.Repository = (*OrderRepository)(nil)
