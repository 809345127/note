# Go DDD 评审基线项目

面向生产实践的参考项目，聚焦：

- Go + Gin + GORM + MySQL
- DDD 分层架构
- 代码评审规范与反模式示例

本仓库用于团队内部工程规范参考，强调“可运行、可落地、可评审”。

## 包含内容

1. 主服务基线（`api/`、`application/`、`domain/`、`infrastructure/`）
2. 可选 Outbox Worker 运行时（`cmd/worker`）
3. 最佳实践与评审文档（`docs/best-practices/`、`docs/review/`）
4. 最小可运行示例（`examples/minimal-service/`）

## 快速开始

### 1）运行主服务

前置条件：

- 已准备 MySQL，并由外部 DDL 系统创建好表（`users`、`orders`、`order_items`、`outbox_events`）

```bash
go run main.go
```

关键配置：

- outbox 在学习基线中固定启用（无需配置）

### 2）运行 Outbox Worker（可选）

```bash
go run ./cmd/worker
```

仅当 `worker.enabled=true` 时，Worker 才会工作。

### 3）运行最小示例（无需 MySQL）

```bash
go run ./examples/minimal-service/cmd/server
```

接口：

- `GET /api/v1/health`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`

## 评审基线文档

- `docs/best-practices/ddd.md`
- `docs/best-practices/gin.md`
- `docs/best-practices/gorm-mysql.md`
- `docs/review/checklist.md`
- `docs/anti-patterns/anti-patterns.md`

## 本地质量检查

```bash
./scripts/check.sh
```

## 历史 DDD 理论资料

已迁移至 `docs/ddd-reference/`。
