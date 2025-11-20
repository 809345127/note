package domain

// UserRepository 用户仓储接口
type UserRepository interface {
	Save(user *User) error
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindAll() ([]*User, error)
	Delete(id string) error
}

// OrderRepository 订单仓储接口
type OrderRepository interface {
	Save(order *Order) error
	FindByID(id string) (*Order, error)
	FindByUserID(userID string) ([]*Order, error)
	FindByUserIDAndStatus(userID string, status OrderStatus) ([]*Order, error)
	FindAll() ([]*Order, error)
	Delete(id string) error
}

// DomainEventPublisher 领域事件发布器接口
type DomainEventPublisher interface {
	Publish(event DomainEvent) error
	Subscribe(eventName string, handler func(DomainEvent)) error
}