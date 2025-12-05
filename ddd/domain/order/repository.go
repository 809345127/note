package order

import "context"

// Repository 订单仓储接口
type Repository interface {
	// NextIdentity 生成新的订单ID
	NextIdentity() string

	// Save 保存或更新订单聚合根
	// 如果order.Version() == 0表示新建，否则为更新
	// 仓储只负责持久化，事件由 UoW 收集并保存到 outbox 表
	Save(ctx context.Context, order *Order) error

	// FindByID 根据ID查找订单聚合根
	FindByID(ctx context.Context, id string) (*Order, error)

	// FindByUserID 查找用户的订单（受控查询）
	FindByUserID(ctx context.Context, userID string) ([]*Order, error)

	// FindDeliveredOrdersByUserID 查找用户已送达的订单（CQRS中的受控查询）
	FindDeliveredOrdersByUserID(ctx context.Context, userID string) ([]*Order, error)

	// Remove 逻辑删除订单聚合根
	Remove(ctx context.Context, id string) error
}

// QueryService 查询服务接口（CQRS模式中的Q端）
type QueryService interface {
	// SearchOrders 搜索订单
	SearchOrders(criteria SearchCriteria) ([]*Order, error)
}

// SearchCriteria 通用查询条件
type SearchCriteria struct {
	Filters   map[string]interface{}
	SortBy    string
	SortOrder string // ASC or DESC
	Page      int
	PageSize  int
}
