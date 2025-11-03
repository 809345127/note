# Golang 工程师实战：Prometheus+Grafana 监控配置全指南

## 一、前置认知：搞懂核心逻辑

### 1. 为什么需要这套监控？

Golang 服务上线后，你需要实时掌握：

* 接口健康度：QPS、错误率、响应时间（用户体验核心）；
* 系统资源：Goroutine 数、内存占用、GC 频率（服务稳定性核心）；
* 多实例状态：K8s Pod 负载均衡、实例重启情况（分布式部署必备）。

### 2. 工具分工

| 工具         | 核心作用                | 关键交互                            |
| ---------- | ------------------- | ------------------------------- |
| Prometheus | 指标采集、存储、计算（时序数据库）   | 从 Golang 服务的`/metrics`接口抓数据     |
| Grafana    | 指标可视化（图表展示、告警配置）    | 发送 PromQL 到 Prometheus 查数据，渲染图表 |
| Golang 客户端 | 定义指标、暴露`/metrics`接口 | `client_golang`库，嵌入业务代码         |

### 3. 必学指标类型

| 类型        | 特点（用途）        | 实战示例                                  |
| --------- | ------------- | ------------------------------------- |
| Counter   | 只增不减（统计累计值）   | 接口请求总数`http_requests_total`           |
| Gauge     | 可增可减（统计瞬时值）   | 活跃请求数`http_active_requests`           |
| Histogram | 统计数值分布（分位数计算） | 接口响应时间`http_request_duration_seconds` |

## 二、环境准备（3 步到位）

### 1. 安装 Prometheus

* **下载**：[官网](https://prometheus.io/download/)下载对应系统版本（如 Linux-amd64）；
* **启动**：

```bash
# 解压
tar -zxvf prometheus-2.52.0.linux-amd64.tar.gz
cd prometheus-2.52.0.linux-amd64

# 直接启动（默认端口9090）
./prometheus --config.file=prometheus.yml
```

* **验证**：访问`http://localhost:9090`，看到 Prometheus UI 即成功。

### 2. 安装 Grafana

* **Linux 快速安装**：

```bash
# 安装依赖
sudo apt-get install -y adduser libfontconfig1

# 下载deb包
wget https://dl.grafana.com/enterprise/release/grafana-enterprise_10.4.0_amd64.deb

# 安装并启动
sudo dpkg -i grafana-enterprise_10.4.0_amd64.deb
sudo systemctl start grafana-server
```

* **验证**：访问`http://localhost:3000`，默认账号密码`admin/admin`（首次登录需修改）。

### 3. Golang 项目引入依赖

在你的 Golang 项目根目录执行，引入 Prometheus 客户端：

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

## 三、Step 1：Golang 代码集成指标（核心实战）

目标：在业务代码中**定义指标→更新指标→暴露 /metrics 接口**，让 Prometheus 能抓到数据。

### 1. 封装指标定义（可复用代码）

创建`internal/metrics/metrics.go`，统一管理指标：

```go
package metrics

import (
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// 1. 定义全局指标（根据业务扩展）
var (
    // Counter：接口请求总数（按路径、方法拆分）
    HTTPRequestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total", // 指标名规范：小写+下划线
            Help: "Total number of HTTP requests",
        },
        []string{"path", "method"}, // 标签：用于维度拆分（必加，方便后续聚合）
    )
    
    // Gauge：当前活跃请求数
    HTTPActiveRequests = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "http_active_requests",
            Help: "Number of active HTTP requests",
        },
    )
    
    // Histogram：接口响应时间（单位：秒，默认桶覆盖常见场景）
    HTTPRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "Duration of HTTP requests in seconds",
            Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, ..., 10]秒
        },
        []string{"path", "method"},
    )
)

// 2. 注册指标（程序启动时执行）
func Init() {
    // 注册自定义指标
    prometheus.MustRegister(HTTPRequestTotal)
    prometheus.MustRegister(HTTPActiveRequests)
    prometheus.MustRegister(HTTPRequestDuration)
    
    // Go运行时指标（可选，推荐：Goroutine数、内存、GC等）
    prometheus.MustRegister(prometheus.NewGoCollector())
}

// 3. 指标中间件（自动更新HTTP请求指标，业务路由直接复用）
func HTTPMetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 瞬时指标：活跃请求数+1
        HTTPActiveRequests.Inc()
        
        // 记录开始时间（用于计算响应时间）
        start := time.Now()
        
        // 延迟执行：请求结束后更新指标
        defer func() {
            HTTPActiveRequests.Dec() // 活跃请求数-1
            duration := time.Since(start).Seconds()
            
            // 累计指标：请求总数+1（按路径、方法拆分）
            HTTPRequestTotal.WithLabelValues(r.URL.Path, r.Method).Inc()
            
            // 分布指标：记录响应时间
            HTTPRequestDuration.WithLabelValues(r.URL.Path, r.Method).Observe(duration)
        }()
        
        // 执行业务逻辑
        next.ServeHTTP(w, r)
    })
}

// 4. 暴露/metrics接口（单独路由，方便Prometheus抓取）
func RegisterMetricsHandler(mux *http.ServeMux) {
    // 配置超时（避免抓取阻塞业务，推荐10秒）
    mux.Handle("/metrics", promhttp.HandlerFor(
        prometheus.DefaultGatherer,
        promhttp.HandlerOpts{Timeout: 10 * time.Second},
    ))
}
```

### 2. 业务代码集成

在你的`main.go`中引入指标模块，串联业务逻辑：

```go
package main

import (
    "fmt"
    "net/http"
    "time"
    
    "your-project/internal/metrics" // 替换为你的实际包路径
)

// 示例业务接口
func helloHandler(w http.ResponseWriter, r *http.Request) {
    time.Sleep(100 * time.Millisecond) // 模拟业务耗时
    w.Write([]byte("Hello, Prometheus!"))
}

func main() {
    // 1. 初始化指标（必须在路由前执行）
    metrics.Init()
    
    // 2. 创建路由，应用指标中间件
    mux := http.NewServeMux()
    
    // 业务路由：加指标中间件
    mux.Handle("/hello", metrics.HTTPMetricsMiddleware(http.HandlerFunc(helloHandler)))
    
    // 暴露/metrics接口
    metrics.RegisterMetricsHandler(mux)
    
    // 3. 启动服务（端口自定义，如8080）
    fmt.Println("Server running on :8080")
    http.ListenAndServe(":8080", mux)
}
```

### 3. 验证指标暴露

启动 Golang 服务后，访问`http://localhost:8080/metrics`，能看到类似以下内容即成功：

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/hello"} 5  # 访问5次/hello后的结果

# HELP http_active_requests Number of active HTTP requests
# TYPE http_active_requests gauge
http_active_requests 0
```

## 四、Step 2：配置 Prometheus 抓取指标

Prometheus 需要知道 "抓哪个服务的哪个接口"，分**非 K8s**和**K8s**两种场景（重点覆盖 K8s，因为生产常用）。

### 场景 1：非 K8s 环境（静态配置）

编辑 Prometheus 配置文件`prometheus.yml`，添加抓取任务：

```yaml
global:
  scrape_interval: 15s  # 全局抓取间隔（默认15秒，高频指标可缩至10秒）
  scrape_timeout: 10s   # 单次抓取超时（必须小于scrape_interval）

scrape_configs:
  # 任务名：自定义（如golang-app）
  - job_name: 'golang-app'
    static_configs:
      # 抓取目标：Golang服务的IP:端口（多个用逗号分隔）
      - targets: ['localhost:8080']
```

* **重启 Prometheus**：`pkill prometheus && ./prometheus --config.file=prometheus.yml`
* **验证抓取**：访问 Prometheus UI → `Status` → `Targets`，看到`golang-app`状态为`UP`即成功。

### 场景 2：K8s 环境（动态服务发现，生产推荐）

K8s 中 Pod 会漂移，需用 "动态发现" 替代静态 IP，推荐用**ServiceMonitor**（基于 Prometheus Operator）。

#### 步骤 1：部署 Prometheus Operator（Helm 一键安装）

```bash
# 添加Helm仓库
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

# 安装（命名空间monitoring，自动创建）
helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
```

#### 步骤 2：Golang 服务的 K8s 配置（Pod+Service）

创建`golang-app-k8s.yaml`，关键是给 Service 加标签，方便 ServiceMonitor 匹配：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: golang-app-service
  namespace: default
  labels:
    app: golang-app  # 关键标签：ServiceMonitor会通过这个匹配
spec:
  selector:
    app: golang-app  # 匹配Pod的标签
  ports:
  - name: metrics  # 端口名：ServiceMonitor需指定
    port: 8080      # Service暴露的端口
    targetPort: 8080 # Pod的容器端口（和Golang服务端口一致）

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: golang-app-deploy
  namespace: default
spec:
  replicas: 3  # 3个Pod实例（模拟分布式部署）
  selector:
    matchLabels:
      app: golang-app
  template:
    metadata:
      labels:
        app: golang-app
    spec:
      containers:
      - name: golang-app
        image: your-golang-image:v1  # 替换为你的镜像
        ports:
        - containerPort: 8080
```

* **部署**：`kubectl apply -f golang-app-k8s.yaml`

#### 步骤 3：创建 ServiceMonitor（让 Prometheus 自动发现）

创建`golang-app-servicemonitor.yaml`：

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: golang-app-servicemonitor
  namespace: monitoring  # 必须和Prometheus Operator同命名空间
  labels:
    release: prometheus  # 匹配Prometheus Operator的release标签
spec:
  selector:
    matchLabels:
      app: golang-app  # 匹配Service的标签（和步骤2一致）
  namespaceSelector:
    matchNames: [default]  # 只在default命名空间找Service
  endpoints:
  - port: metrics  # 匹配Service的port名（步骤2中的metrics）
    interval: 10s  # 抓取间隔（覆盖全局配置）
```

* **部署**：`kubectl apply -f golang-app-servicemonitor.yaml`
* **验证**：Prometheus UI → `Status` → `ServiceDiscovery`，能看到`golang-app-servicemonitor`即成功。

## 五、Step 3：Grafana 配置可视化（实战场景）

目标：用 Grafana 做 3 个核心面板，覆盖 80% 监控需求。

### 1. 连接 Prometheus 数据源

* 登录 Grafana → 左侧`Configuration`（齿轮图标）→ `Data Sources` → `Add data source`；
* 搜索`Prometheus` → 输入 Prometheus 地址（非 K8s：`http://localhost:9090`；K8s：`http://prometheus-kube-prometheus-prometheus.monitoring:9090`）；
* 点击`Save & test`，显示 "Data source is working" 即成功。

### 2. 实战面板 1：接口 QPS 监控（折线图）

#### 配置步骤：

1. 左侧`+` → `Dashboard` → `Add visualization`；
2. 数据源选 Prometheus，输入 PromQL：

```promql
# 按接口拆分QPS（5分钟窗口平滑）
rate(http_requests_total[5m])
```

3. **图表类型**：选`Line chart`；
4. **图例优化**：`Legend format`输入`{{method}} {{path}}`（显示 "方法 + 路径"）；
5. **标题**：输入 "接口 QPS 监控"，点击`Apply`。

#### 效果：

* X 轴：时间；Y 轴：QPS（每秒请求数）；
* 多条彩色线，每条对应一个接口（如`GET /hello`），直观看到流量波动。

### 3. 实战面板 2：接口响应时间 P95（折线图）

#### 配置步骤：

1. 新增可视化，输入 PromQL：

```promql
# 95%分位响应时间（转毫秒，更易读）
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path, method)) * 1000
```

2. **图表类型**：`Line chart`；
3. **Y 轴单位**：选`Milliseconds (ms)`；
4. **图例**：`{{method}} {{path}} (P95)`；
5. **标题**："接口响应时间 P95 监控"。

#### 效果：

* 每条线对应一个接口的 P95 响应时间（95% 请求耗时≤该值）；
* 若`GET /hello`的 P95 从 50ms 涨到 500ms，说明接口性能下降。

### 4. 实战面板 3：K8s Pod 负载（柱状图）

#### 配置步骤（K8s 环境专属）：

1. 新增可视化，输入 PromQL：

```promql
# 按Pod拆分总请求数（看各实例负载均衡情况）
sum(http_requests_total) by (__meta_kubernetes_pod_name, path, method)
```

2. **图表类型**：`Bar chart`；
3. **X 轴**：选`__meta_kubernetes_pod_name`（显示 Pod 名）；
4. **标题**："K8s Pod 接口请求总量"。

#### 效果：

* 每个 Pod 对应一根柱子，高度代表该 Pod 的累计请求数，快速判断是否负载不均（如某 Pod 请求数是其他的 2 倍）。

## 六、实战技巧与避坑指南

### 1. 指标命名规范

* 格式：`{业务}_{指标}_{类型}`（如`http_requests_total`）；
* 小写 + 下划线，避免特殊字符（如`httpRequestTotal`不推荐）。

### 2. 标签设计避坑

* 避免高基数标签：如用户 ID、订单号（会导致指标爆炸，Prometheus 存储崩溃）；
* 必加标签：`path`、`method`（接口维度），K8s 环境自动加`__meta_kubernetes_pod_name`（实例维度）。

### 3. K8s Pod 重启不影响整体指标

* 按`__meta_kubernetes_service_name`聚合（如`sum(rate(...)) by (__meta_kubernetes_service_name)`）；
* Pod 重启后，Prometheus 自动发现新 Pod，`rate()`函数处理 Counter 重置，整体指标准确。

### 4. 数据保留与性能优化

* Prometheus 配置数据保留：`prometheus.yml`中加`storage_retention: 15d`（保留 15 天，避免磁盘占满）；
* 非核心指标抓取间隔：K8s ServiceMonitor 中`interval`设为 30s，减少 Prometheus 压力。

## 七、常见问题排查

### 1. Prometheus 抓不到指标？

* 检查 Golang 服务：`curl http://localhost:8080/metrics`是否有数据；
* 检查网络：Prometheus 服务器能否 ping 通 Golang 服务（K8s 环境看 Pod 是否在同一个命名空间，Service 是否正常）；
* 检查 Prometheus 配置：`Status` → `Configuration`，看`targets`是否正确。

### 2. Grafana 没数据？

* 检查 PromQL：复制 PromQL 到 Prometheus UI 的`Graph`页，看是否有结果（排除 PromQL 错误）；
* 检查数据源：确认 Grafana 的 Prometheus 地址正确，且网络可达。

### 3. K8s ServiceMonitor 不生效？

* 检查标签匹配：ServiceMonitor 的`selector.matchLabels`是否和 Service 的标签一致；
* 检查命名空间：ServiceMonitor 必须和 Prometheus Operator 在同一命名空间（如`monitoring`）；
* 查看事件：`kubectl describe servicemonitor golang-app-servicemonitor -n monitoring`，看是否有错误。

> （注：文档部分内容可能由 AI 生成）