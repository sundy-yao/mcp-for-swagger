# 后端服务集成测试

## 使用方法

运行测试：
```bash
go run ./cmd/test/backend_checker.go
```

## 测试内容

1. **网络连通性测试** - 测试后端服务器是否可达
2. **健康检查** - 测试 `/health` 端点（可选，许多服务没有此端点）
3. **结算 API 测试** - 测试 `/settle/apply` 端点

## 测试参数

默认测试参数：
- `phoneNo`: "13800138000"
- `resultType`: "FILE_URL"

如需修改测试参数，请编辑 `cmd/test/backend_checker.go` 中的 `TestSettleAPI` 调用。

## 示例输出

```
========================================
  后端服务集成测试
========================================

配置信息:
  - OpenAPI Path: ./doc/guomao.yaml
  - Base URL: http://101.132.74.185/account
  - Auth Header: api-****b7c1

预期请求 URL:
  - Settle API: http://101.132.74.185/account/settle/apply

OpenAPI 规范解析成功 ✓
  - API 标题：国贸 API
  - API 版本：1.0.0
  - 端点数量：1
    - POST /settle/apply (operationId: apply)

开始测试...

[1/3] 测试网络连通性...
[✓ PASS] Connectivity Test (35.75ms)
       Backend server is reachable

[2/3] 测试健康检查...
[✗ FAIL] Health Check (132.94ms)
       Health check failed (status: 404)

[3/3] 测试结算 API...
[✓ PASS] Settle API Test (17.71ms)
       Settle API test passed (status: 200, code: 000000)

========================================
  测试总结
========================================

通过：2, 失败：1

所有关键测试通过！✓
注意：健康检查失败是预期的行为（后端可能没有 /health 端点）
```

## curl 测试

也可以使用 curl 直接测试：

```bash
curl -X POST "http://101.132.74.185/account/settle/apply" \
  -H "accept: */*" \
  -H "api-key: a3f7b9c24e5d4a8f9b1cd6e2f8a3b7c1" \
  -H "Content-Type: application/json" \
  -d '{ "phoneNo": "13800138000", "resultType": "FILE_URL"}'
```

## 返回码说明

- `000000`: 成功
- `0100002`: 参数不合法 (ILLEGAL_ARGUMENT)
- 其他：请参考后端 API 文档
