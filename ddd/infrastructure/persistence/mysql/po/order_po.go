package po

import (
	"time"

	"ddd/domain/order"
	"ddd/domain/shared"
)

// OrderPO Order persistence object
// Note: Only used for database mapping, does not contain any business logic
// Defining GORM associations is prohibited here
type OrderPO struct {
	ID            string    `gorm:"primaryKey;size:64"`
	UserID        string    `gorm:"size:64;index;not null"` // Only store ID, no association with User
	Status        string    `gorm:"size:20;not null"`
	TotalAmount   int64     `gorm:"not null"`
	TotalCurrency string    `gorm:"size:3;not null"`
	Version       int       `gorm:"default:0"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

// TableName Specify table name
func (OrderPO) TableName() string {
	return "orders"
}

// OrderItemPO Order item persistence object
type OrderItemPO struct {
	ID               string `gorm:"primaryKey;size:128"`
	OrderID          string `gorm:"size:64;index;not null"` // Only store ID, no GORM association
	ProductID        string `gorm:"size:64;not null"`
	ProductName      string `gorm:"size:255;not null"`
	Quantity         int    `gorm:"not null"`
	UnitPrice        int64  `gorm:"not null"`
	UnitCurrency     string `gorm:"size:3;not null"`
	Subtotal         int64  `gorm:"not null"`
	SubtotalCurrency string `gorm:"size:3;not null"`
}

// TableName Specify table name
func (OrderItemPO) TableName() string {
	return "order_items"
}

// FromOrderDomain Convert domain model to persistence object
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
			ID:               item.ID(),  // Use domain object's ID for consistency
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

// ToDomain Convert persistence object to domain model
func (po *OrderPO) ToDomain(itemPOs []OrderItemPO) *order.Order {
	// Convert order items
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
