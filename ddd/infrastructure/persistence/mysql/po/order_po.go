package po

import (
	"time"

	"ddd-example/domain/order"
	"ddd-example/domain/shared"
)

// OrderPO 订单持久化对象
// 注意：只用于数据库映射，不包含任何业务逻辑
// 禁止在此定义 GORM 关联
type OrderPO struct {
	ID            string    `gorm:"primaryKey;size:64"`
	UserID        string    `gorm:"size:64;index;not null"` // 只存ID，不关联User
	Status        string    `gorm:"size:20;not null"`
	TotalAmount   int64     `gorm:"not null"`
	TotalCurrency string    `gorm:"size:3;not null"`
	Version       int       `gorm:"default:0"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (OrderPO) TableName() string {
	return "orders"
}

// OrderItemPO 订单项持久化对象
type OrderItemPO struct {
	ID               string `gorm:"primaryKey;size:128"`
	OrderID          string `gorm:"size:64;index;not null"` // 只存ID，不用GORM关联
	ProductID        string `gorm:"size:64;not null"`
	ProductName      string `gorm:"size:255;not null"`
	Quantity         int    `gorm:"not null"`
	UnitPrice        int64  `gorm:"not null"`
	UnitCurrency     string `gorm:"size:3;not null"`
	Subtotal         int64  `gorm:"not null"`
	SubtotalCurrency string `gorm:"size:3;not null"`
}

// TableName 指定表名
func (OrderItemPO) TableName() string {
	return "order_items"
}

// FromOrderDomain 将领域模型转换为持久化对象
func FromOrderDomain(o *order.Order) (*OrderPO, []OrderItemPO) {
	orderPO := &OrderPO{
		ID:            o.ID(),
		UserID:        o.UserID(),
		Status:        string(o.Status()),
		TotalAmount:   o.TotalAmount().Amount(),
		TotalCurrency: o.TotalAmount().Currency(),
		Version:       o.Version(),
		CreatedAt:     o.CreatedAt(),
		UpdatedAt:     o.UpdatedAt(),
	}

	items := o.Items()
	itemPOs := make([]OrderItemPO, len(items))
	for i, item := range items {
		itemPOs[i] = OrderItemPO{
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
	}

	return orderPO, itemPOs
}

// ToDomain 将持久化对象转换为领域模型
func (po *OrderPO) ToDomain(itemPOs []OrderItemPO) *order.Order {
	// 转换订单项
	items := make([]order.OrderItem, len(itemPOs))
	for i, itemPO := range itemPOs {
		items[i] = order.RebuildItemFromDTO(order.ItemReconstructionDTO{
			ID:          itemPO.ID,
			ProductID:   itemPO.ProductID,
			ProductName: itemPO.ProductName,
			Quantity:    itemPO.Quantity,
			UnitPrice:   *shared.NewMoney(itemPO.UnitPrice, itemPO.UnitCurrency),
			Subtotal:    *shared.NewMoney(itemPO.Subtotal, itemPO.SubtotalCurrency),
		})
	}

	return order.RebuildFromDTO(order.ReconstructionDTO{
		ID:          po.ID,
		UserID:      po.UserID,
		Items:       items,
		TotalAmount: *shared.NewMoney(po.TotalAmount, po.TotalCurrency),
		Status:      order.Status(po.Status),
		Version:     po.Version,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	})
}
