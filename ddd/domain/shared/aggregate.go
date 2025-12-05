package shared

// AggregateRoot 聚合根接口
// 聚合根是DDD的核心概念，它是聚合的入口点，维护聚合的一致性边界
// 特性：
// 1. 有全局唯一标识
// 2. 维护聚合内部的不变量
// 3. 所有修改必须通过聚合根进行
// 4. 负责发布领域事件
type AggregateRoot interface {
	// ID 返回聚合根的全局唯一标识
	ID() string

	// Version 返回当前版本号，用于乐观锁并发控制
	Version() int

	// PullEvents 获取并清空聚合根记录的领域事件
	// 这是标准的领域事件模式：聚合根记录事件，存储库在保存后发布事件
	PullEvents() []DomainEvent
}

// IsAggregateRoot 类型标记函数
// 用于编译时检查某个类型是否实现了AggregateRoot接口
// 使用方法：var _ = IsAggregateRoot(&User{})
func IsAggregateRoot(agg AggregateRoot) AggregateRoot {
	return agg
}

// Entity 实体接口
// 实体与值对象的区别：
// 1. 实体有唯一标识（ID）
// 2. 实体的生命周期较长
// 3. 通过标识判断相等性（即使属性相同，ID不同就是不同的实体）
type Entity interface {
	ID() string
}

// ValueObject 值对象接口
// 值对象的特征：
// 1. 没有唯一标识
// 2. 不可变（immutable）
// 3. 通过属性值判断相等性
// 4. 通常用于描述性概念
// 注意：Go语言中没有完美的方式强制实现不可变性，需要通过约定和编码规范保证
type ValueObject interface {
	// Equals 比较两个值对象是否相等
	Equals(other interface{}) bool
}
