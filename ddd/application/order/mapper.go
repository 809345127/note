package order

import (
	"ddd/domain/order"
	"ddd/domain/shared"
)

func toItemRequests(items []OrderItemRequest) []order.ItemRequest {
	requests := make([]order.ItemRequest, len(items))
	for i, item := range items {
		requests[i] = order.ItemRequest{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   *shared.NewMoney(item.UnitPrice, item.Currency),
		}
	}
	return requests
}

func toOrderResponse(o *order.Order) *OrderResponse {
	items := make([]OrderItemResponse, len(o.Items()))
	for i, item := range o.Items() {
		items[i] = OrderItemResponse{
			ProductID:   item.ProductID(),
			ProductName: item.ProductName(),
			Quantity:    item.Quantity(),
			UnitPrice: MoneyResponse{
				Amount:   item.UnitPrice().Amount(),
				Currency: item.UnitPrice().Currency(),
			},
			Subtotal: MoneyResponse{
				Amount:   item.Subtotal().Amount(),
				Currency: item.Subtotal().Currency(),
			},
		}
	}

	return &OrderResponse{
		ID:     o.ID(),
		UserID: o.UserID(),
		Items:  items,
		TotalAmount: MoneyResponse{
			Amount:   o.TotalAmount().Amount(),
			Currency: o.TotalAmount().Currency(),
		},
		Status:    string(o.Status()),
		CreatedAt: o.CreatedAt(),
		UpdatedAt: o.UpdatedAt(),
	}
}
