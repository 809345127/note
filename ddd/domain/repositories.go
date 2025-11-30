package domain

import "context"

// UserRepository 用户仓储接口
// DDD原则：
// 1. 仓储只负责聚合根的持久化
// 2. 不应该暴露批量查询（如FindAll），这类操作应该放在查询服务中
// 3. 使用NextIdentity生成ID，而非在实体中直接生成（便于测试和ID策略调整）
// 4. 包含context.Context以支持超时、取消和事务
type UserRepository interface {
	// NextIdentity 生成新的用户ID（DDD推荐在仓储中生成ID）
	NextIdentity() string

	// Save 保存或更新用户聚合根（包括聚合内的所有实体）
	// 如果user.Version() == 0表示新建，否则为更新
	Save(ctx context.Context, user *User) error

	// FindByID 根据ID查找用户聚合根
	FindByID(ctx context.Context, id string) (*User, error)

	// FindByEmail 根据邮箱查找用户（业务唯一性约束）
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Remove 逻辑删除用户聚合根（DDD推荐逻辑删除而非物理删除）
	Remove(ctx context.Context, id string) error
}

// OrderRepository 订单仓储接口
type OrderRepository interface {
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
// DDD区分：命令（修改）和查询（读取）应该分离
// 仓储负责命令操作（加载聚合根，保存聚合根）
// 查询服务负责复杂查询，不受聚合边界限制
type UserQueryService interface {
	// SearchUsers 搜索用户（支持分页、排序）
	SearchUsers(criteria SearchCriteria) ([]*User, error)

	// CountUsers 统计用户数量
	CountUsers(criteria SearchCriteria) (int, error)
}

type OrderQueryService interface {
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
