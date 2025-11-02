# Spring Boot 监控核心组件（Actuator + Micrometer + Prometheus）笔记


## 一、核心组件关系图（组件职责与依赖）
```
┌─────────────────────────────────────────────────────────────────┐
│                        你的 Spring Boot 应用                     │
│                                                                 │
│  ┌─────────────────┐       ┌─────────────────┐                  │
│  │   业务代码      │       │  配置文件        │                  │
│  │ （含自定义指标） │       │ （management.*） │                  │
│  │ - Counter       │       │ - 暴露端点       │                  │
│  │ - Timer         │       │ - 健康检查详情   │                  │
│  └────────┬────────┘       └────────┬────────┘                  │
│           │                         │                           │
│           ▼                         ▼                           │
│  ┌─────────────────┐       ┌─────────────────┐                  │
│  │   Micrometer    │◄──────┤   Actuator      │                  │
│  │ （指标内核）    │       │ （端点门户）    │                  │
│  │ - 收集指标      │       │ - 暴露 3 个端点  │                  │
│  │ - 标准化指标    │       │   ① /health      │                  │
│  │ - 对接监控系统  │       │   ② /info        │                  │
│  └────────┬────────┘       │   ③ /prometheus  │                  │
│           │                └─────────────────┘                  │
└───────────┼─────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      外部监控系统                                │
│                                                                 │
│  ┌─────────────────┐       ┌─────────────────┐                  │
│  │   Prometheus    │       │   Grafana       │                  │
│  │ （指标存储）    │◄──────┤ （可视化）      │                  │
│  │ - 抓取 /prometheus│      │ - 展示指标图表  │                  │
│  │ - 时序存储      │       │ - 配置告警      │                  │
│  └─────────────────┘       └─────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
```


### 各组件核心作用与依赖关系表
| 组件                | 核心作用                                                                 | 依赖关系                     |
|---------------------|--------------------------------------------------------------------------|------------------------------|
| **业务代码**        | 通过 Micrometer API 定义自定义指标（如订单计数 `Counter`、接口耗时 `Timer`） | 依赖 Micrometer 核心库       |
| **配置文件**        | 配置 Actuator 端点暴露范围、健康检查详情显示（`management.*` 前缀配置）   | 仅作用于 Actuator            |
| **Micrometer**      | 1. 收集指标（内置：JVM/HTTP 请求；自定义：业务指标）<br>2. 标准化指标格式<br>3. 支持对接 Prometheus | Actuator 的 `/prometheus` 端点依赖它提供数据 |
| **Actuator**        | 1. 暴露 3 个核心端点（`/health`/`/info`/`/prometheus`）<br>2. 提供健康检查和应用基础信息 | 依赖 Micrometer（指标来源）；无需依赖 Prometheus |
| **Prometheus**      | 1. 定时抓取 `/actuator/prometheus` 端点的指标<br>2. 时序化存储指标数据     | 依赖 Actuator 暴露的端点     |
| **Grafana**         | 1. 对接 Prometheus 读取指标<br>2. 可视化展示（如 Counter 趋势图）<br>3. 配置告警 | 依赖 Prometheus（数据来源）  |


## 二、自定义 Counter 指标数据流程图（从产生到可视化）
以“订单创建计数 `Counter`”为例，完整数据流转路径如下：

### Step 1：指标产生（业务代码）
业务代码中通过 `MeterRegistry` 注册并触发 `Counter`：
```java
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import org.springframework.stereotype.Component;

@Component
public class OrderService {
    // 1. 注入 Micrometer 自动配置的 MeterRegistry
    private final Counter orderCreateCounter;

    // 2. 注册 "order.create" 计数器
    public OrderService(MeterRegistry registry) {
        this.orderCreateCounter = Counter.builder("order.create")
                .description("订单创建成功次数")
                .register(registry);
    }

    // 3. 业务逻辑触发计数（每次创建订单 +1）
    public void createOrder() {
        orderCreateCounter.increment(); // 指标产生
    }
}
```


### Step 2：指标收集（Micrometer）
- `MeterRegistry` 接收 `order.create` 计数器的增量数据，存储在内存中；
- 自动标准化指标格式（如为 Counter 补充 `total` 后缀，适配 Prometheus 规范）；
- 无需额外配置，Micrometer 自动管理指标生命周期。


### Step 3：端点暴露（Actuator）
Actuator 根据配置文件，将不同类型数据通过对应端点暴露：
| 端点                | 数据来源                  | 是否包含 `order.create` 指标 | 访问示例                     |
|---------------------|---------------------------|------------------------------|------------------------------|
| `/actuator/health`  | 内置/自定义 `HealthIndicator` | ❌ 不包含（仅健康状态）      | `http://localhost:8080/actuator/health` |
| `/actuator/info`    | 配置文件 `info.*`/自定义 `InfoContributor` | ❌ 不包含（仅基础信息） | `http://localhost:8080/actuator/info` |
| `/actuator/prometheus` | Micrometer 的 `MeterRegistry` | ✅ 包含（Prometheus 格式） | `http://localhost:8080/actuator/prometheus` |

**示例：`/actuator/prometheus` 端点返回的 `order.create` 数据**
```text
# HELP order_create_total 订单创建成功次数
# TYPE order_create_total counter
order_create_total 15.0  # 假设已创建 15 个订单
```


### Step 4：指标抓取（Prometheus）
1. 在 Prometheus 配置文件（`prometheus.yml`）中添加抓取规则：
   ```yaml
   scrape_configs:
     - job_name: "springboot-app"
       scrape_interval: 15s  # 每 15 秒抓取一次
       static_configs:
         - targets: ["localhost:8080"]  # 你的 Spring Boot 应用地址
   ```
2. Prometheus 定时访问 `http://localhost:8080/actuator/prometheus`，抓取指标数据；
3. 将数据按时间戳时序化存储（如记录每 15 秒的 `order_create_total` 数值）。


### Step 5：指标可视化（Grafana）
1. 在 Grafana 中添加“Prometheus”数据源，配置 Prometheus 地址；
2. 导入或自定义仪表盘（Dashboard），筛选指标 `order_create_total`；
3. 展示趋势图（如“近 1 小时订单创建次数变化”），并配置告警（如 5 分钟内订单数为 0 触发通知）。


## 三、核心记忆点（复习重点）
1. **依赖关系口诀**  
   Actuator 是“门户”（暴露端点），Micrometer 是“内核”（产指标），Prometheus 是“仓库”（存指标），Grafana 是“显示器”（展指标）。

2. **3 个端点分工**  
   - 健康看 `/health`（UP/DOWN 状态）；  
   - 信息看 `/info`（应用版本/描述）；  
   - 指标看 `/prometheus`（自定义 Counter/Timer + 内置指标）。

3. **数据流向简化版**  
   业务代码产生指标 → Micrometer 收集 → Actuator 暴露 → Prometheus 抓取 → Grafana 展示。
