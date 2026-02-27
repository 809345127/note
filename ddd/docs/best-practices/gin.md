# Gin 最佳实践（评审基线）

## 强制规则

1. 中间件顺序：`request-id -> recovery -> logging -> auth(可选) -> rate-limit -> CORS`。
2. 控制器职责：参数绑定/校验、调用应用服务、映射响应。
3. 全部接口使用统一错误响应模型。
4. 健康检查必须包含 `/health`、`/health/live`、`/health/ready`。
5. 必须限制请求体大小并设置超时。

## 拒绝合并场景

1. 在控制器中实现业务逻辑。
2. 不同接口返回不一致的错误 JSON 结构。
3. 必填字段缺少验证标签。
4. 日志缺少请求 ID。

## 正向参考文件

- `api/router.go`
- `api/middleware/middleware.go`
- `api/response/response.go`
