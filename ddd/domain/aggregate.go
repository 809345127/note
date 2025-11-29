package domain

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

// 聚合根和实体的设计原则：

// 1. 聚合内的实体只能通过聚合根访问
//    ❌ 错误：order.Items[0].Quantity = 100  // 直接修改聚合内实体
//    ✅ 正确：order.UpdateItemQuantity(itemID, 100)  // 通过聚合根方法

// 2. 聚合根负责维护聚合的一致性边界
//    - 添加/删除聚合内实体必须通过聚合根方法
//    - 聚合根验证所有操作不违反业务规则

// 3. 聚合根的实现模式：
//    type Order struct {
//        id       string  // 聚合根ID
//        items    []OrderItem  // 聚合内实体（私有字段）
//        version  int  // 乐观锁版本
//        events   []DomainEvent  // 领域事件列表
//    }
//
//    // 聚合根方法示例
//    func (o *Order) AddItem(...) error {
//        // 1. 验证业务规则
//        if o.status != OrderStatusPending {
//            return errors.New("cannot modify confirmed order")
//        }
//        // 2. 执行修改
//        o.items = append(o.items, newItem)
//        // 3. 记录事件
//        o.recordEvent(NewOrderItemAddedEvent(...))
//        return nil
//    }

// 4. 仓储的职责：
//    - 只持久化聚合根（整个聚合一起保存）
//    - 保证聚合的一致性（原子性操作）
//    - 在保存后发布聚合根的事件：
//
//      func (r *OrderRepository) Save(order *Order) error {
//          // 1. 保存聚合根
//          if err := r.db.Save(order); err != nil {
//              return err
//          }
//          // 2. 获取事件
//          events := order.PullEvents()
//          // 3. 发布事件
//          for _, event := range events {
//              r.eventPublisher.Publish(event)
//          }
//          return nil
//      }
