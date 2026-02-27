# 最小服务示例

该示例演示一个不依赖 MySQL 的最小 DDD 风格服务。

```bash
go run ./examples/minimal-service/cmd/server
```

## 接口示例

```bash
curl http://localhost:8081/api/v1/health

curl -X POST http://localhost:8081/api/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"编写第一个任务"}'

curl http://localhost:8081/api/v1/tasks
```
