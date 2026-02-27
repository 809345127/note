package shared

// AggregateRoot 表示聚合根。
type AggregateRoot interface {
	ID() string
	Version() int
	PullEvents() []DomainEvent
}

// Entity 表示实体。
type Entity interface {
	ID() string
}

// ValueObject 表示值对象。
type ValueObject interface {
	Equals(other interface{}) bool
}
