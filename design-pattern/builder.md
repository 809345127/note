# 建造者模式（Builder Pattern）

## 为什么要有这种模式

1. 一般，只用一个全参构造方法，参数太多了！
2. 然后我们想到了必填的用构造方法，其他的用 `set` 方法。
3. 但是还是有几个问题：
   1. 如果必填的太多，构造方法还是回到第一个问题。
   2. 不同参数之间有依赖关系，不好校验。
   3. 有的时候希望对象属性不可变。

## Java 实现模板

```java
public class ResourcePoolConfig {
  private String name;
  private int maxTotal;
  private int maxIdle;
  private int minIdle;

  private ResourcePoolConfig(Builder builder) {
    this.name = builder.name;
    this.maxTotal = builder.maxTotal;
    this.maxIdle = builder.maxIdle;
    this.minIdle = builder.minIdle;
  }
  //...省略getter方法...

  // 我们将Builder类设计成了ResourcePoolConfig的内部类。
  // 我们也可以将Builder类设计成独立的非内部类ResourcePoolConfigBuilder。
  public static class Builder {
    private static final int DEFAULT_MAX_TOTAL = 8;
    private static final int DEFAULT_MAX_IDLE = 8;
    private static final int DEFAULT_MIN_IDLE = 0;

    private String name;
    private int maxTotal = DEFAULT_MAX_TOTAL;
    private int maxIdle = DEFAULT_MAX_IDLE;
    private int minIdle = DEFAULT_MIN_IDLE;

    public ResourcePoolConfig build() {
      // 校验逻辑放到这里来做，包括必填项校验、依赖关系校验、约束条件校验等
      if (StringUtils.isBlank(name)) {
        throw new IllegalArgumentException("name should not be empty");
      }
      if (maxIdle > maxTotal) {
        throw new IllegalArgumentException("maxIdle should <= maxTotal");
      }
      if (minIdle > maxTotal || minIdle > maxIdle) {
        throw new IllegalArgumentException("minIdle should <= maxIdle && <= maxTotal");
      }

      return new ResourcePoolConfig(this);
    }

    public Builder setName(String name) {
      if (StringUtils.isBlank(name)) {
        throw new IllegalArgumentException("name should not be empty");
      }
      this.name = name;
      return this;
    }

    public Builder setMaxTotal(int maxTotal) {
      if (maxTotal <= 0) {
        throw new IllegalArgumentException("maxTotal should > 0");
      }
      this.maxTotal = maxTotal;
      return this;
    }

    public Builder setMaxIdle(int maxIdle) {
      if (maxIdle < 0) {
        throw new IllegalArgumentException("maxIdle should >= 0");
      }
      this.maxIdle = maxIdle;
      return this;
    }

    public Builder setMinIdle(int minIdle) {
      if (minIdle < 0) {
        throw new IllegalArgumentException("minIdle should >= 0");
      }
      this.minIdle = minIdle;
      return this;
    }
  }
}

// 这段代码会抛出 IllegalArgumentException，因为 minIdle > maxIdle
ResourcePoolConfig config = new ResourcePoolConfig.Builder()
        .setName("dbconnectionpool")
        .setMaxTotal(16)
        .setMaxIdle(10)
        .setMinIdle(12)
        .build();
```

## Go 实现模板

```go
package builder

import (
    "errors"
)

const (
    defaultMaxTotal = 8
    defaultMaxIdle  = 8
    defaultMinIdle  = 0
)

type ResourcePoolConfig struct {
    Name     string
    MaxTotal int
    MaxIdle  int
    MinIdle  int
}

type ResourcePoolConfigBuilder struct {
    name     string
    maxTotal int
    maxIdle  int
    minIdle  int
}

func NewResourcePoolConfigBuilder() *ResourcePoolConfigBuilder {
    return &ResourcePoolConfigBuilder{
        maxTotal: defaultMaxTotal,
        maxIdle:  defaultMaxIdle,
        minIdle:  defaultMinIdle,
    }
}

func (b *ResourcePoolConfigBuilder) Name(name string) *ResourcePoolConfigBuilder {
    b.name = name
    return b
}

func (b *ResourcePoolConfigBuilder) MaxTotal(maxTotal int) *ResourcePoolConfigBuilder {
    b.maxTotal = maxTotal
    return b
}

func (b *ResourcePoolConfigBuilder) MaxIdle(maxIdle int) *ResourcePoolConfigBuilder {
    b.maxIdle = maxIdle
    return b
}

func (b *ResourcePoolConfigBuilder) MinIdle(minIdle int) *ResourcePoolConfigBuilder {
    b.minIdle = minIdle
    return b
}

func (b *ResourcePoolConfigBuilder) Build() (*ResourcePoolConfig, error) {
    if b.name == "" {
        return nil, errors.New("name should not be empty")
    }
    if b.maxTotal <= 0 {
        return nil, errors.New("maxTotal should > 0")
    }
    if b.maxIdle < 0 {
        return nil, errors.New("maxIdle should >= 0")
    }
    if b.minIdle < 0 {
        return nil, errors.New("minIdle should >= 0")
    }
    if b.maxIdle > b.maxTotal {
        return nil, errors.New("maxIdle should <= maxTotal")
    }
    if b.minIdle > b.maxTotal || b.minIdle > b.maxIdle {
        return nil, errors.New("minIdle should <= maxIdle && <= maxTotal")
    }

    return &ResourcePoolConfig{
        Name:     b.name,
        MaxTotal: b.maxTotal,
        MaxIdle:  b.maxIdle,
        MinIdle:  b.minIdle,
    }, nil
}

// 使用示例
// config, err := NewResourcePoolConfigBuilder().
//     Name("dbconnectionpool").
//     MaxTotal(16).
//     MaxIdle(10).
//     MinIdle(12).
//     Build()
// if err != nil {
//     // handle error
// }
```

