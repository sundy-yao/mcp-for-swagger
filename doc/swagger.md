# 工单系统开放 API - Swagger 文档

## 概述

工单系统开放 API 提供了基于 API Token 认证的接口，允许外部系统（如 AI 助手、钉钉机器人等）创建和查询工单。

**工单编号规则**: `渠道号@MD5(提交人 ID| 标题 | 类别)`

示例：
- 钉钉渠道：`ding@a1b2c3d4e5f6...`
- 企微渠道：`wx@1a2b3c4d5e6f...`
- API 渠道：`api@abc123def456...`

**渠道代码映射**：
- `dingtalk` / `钉钉` → `ding`
- `wechat` / `企微` / `微信` → `wx`
- `api` → `api`
- `web` → `web`
- `email` → `email`
- 其他 → `oth`

**工单编号生成方式**：

```bash
# 1. 构建哈希内容：提交人 ID| 标题 | 类别
# 2. 计算 MD5
# 3. 拼接：渠道号@MD5

# 示例（Linux/Mac）
CONTENT="user123|无法登录系统 | 技术支持"
HASH=$(echo -n "$CONTENT" | md5sum | cut -d' ' -f1)
TICKET_NO="ding@${HASH}"
echo $TICKET_NO
```

**防重机制**: 创建工单接口会检查工单编号是否已存在。由于工单编号是基于内容哈希生成的，相同内容的工单会产生相同的编号，从而天然防重。

## 认证方式

所有接口都需要在请求头中携带 API Token：

- **方式 1**: `X-Api-Token: your_token_here`
- **方式 2**: `Authorization: Bearer your_token_here`

## API 列表

---

### 1. 创建工单

**请求**

```http
POST /open/ticket/createTicket
Content-Type: application/json
X-Api-Token: your_token_here
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ticketNo | string | **是** | 工单编号（AI 客户端按规则生成：`渠道号@MD5(提交人 ID| 标题 | 类别)`） |
| title | string | 是 | 工单标题 (2-200 字符) |
| description | string | 是 | 工单描述 (至少 10 字符) |
| category | string | 是 | 工单类别（如：技术支持、Bug 反馈） |
| priority | string | 是 | 紧急程度 (low:低 medium:中 high:高 urgent:紧急) |
| submitterID | string | 否 | 提交人 ID（钉钉/企微 openid，不传则使用 API Token 的 UserID） |
| submitter | string | 否 | 提交人姓名 (默认："API 用户") |
| submitterPhone | string | 否 | 提交人手机号 |
| submitterSource | string | 否 | 提交来源 (默认："api"，可选：web/dingtalk/api/wechat/email) |
| attachment | string | 否 | 附件路径 |

**请求示例**

```json
{
  "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
  "title": "无法登录系统",
  "description": "用户反馈无法登录系统，提示账号或密码错误",
  "category": "技术支持",
  "priority": "high",
  "submitterID": "user_openid_123",
  "submitter": "张三",
  "submitterPhone": "13800138000",
  "submitterSource": "dingtalk"
}
```

**AI 调用流程**

由于工单编号是基于内容哈希生成的，相同内容会产生相同的编号，因此：

1. AI 客户端按照规则生成工单编号
2. 直接调用 `/open/ticket/createTicket` 创建工单
3. 后端会检查工单编号是否已存在
4. 如果已存在，返回已存在的工单信息；如果不存在，创建新工单

**防重机制说明**

系统会检查工单编号是否已存在。由于工单编号是基于 `提交人 ID + 标题 + 类别` 的 MD5 哈希生成的，相同内容的工单会产生相同的编号，从而天然防重。

---

### 2. 分页获取工单列表

**请求**

```http
GET /open/ticket/getTicketList?page=1&pageSize=10&status=0&priority=3
X-Api-Token: your_token_here
```

**请求参数 (Query Parameters)**

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| page | integer | 否 | 1 | 页码 |
| pageSize | integer | 否 | 10 | 每页数量 |
| ticketNo | string | 否 | - | 工单编号（精确匹配） |
| title | string | 否 | - | 工单标题（模糊搜索） |
| submitter | string | 否 | - | 提交人（模糊搜索） |
| category | string | 否 | - | 工单类别（精确匹配） |
| priority | string | 否 | - | 紧急程度 (low/medium/high/urgent) |
| status | string | 否 | - | 工单状态 (pending/processing/resolved/closed/rejected) |
| handler | string | 否 | - | 处理人（模糊搜索） |

**工单状态说明**

| 状态值 | 状态名称 | 说明 |
|--------|---------|------|
| pending | 待处理 | 新创建的工单 |
| processing | 处理中 | 已分配处理人 |
| resolved | 已解决 | 处理完成 |
| closed | 已关闭 | 用户确认关闭 |
| rejected | 已拒绝 | 拒绝处理 |

**紧急程度说明**

| 值 | 名称 | 说明 |
|--------|---------|------|
| low | 低 | 低优先级 |
| medium | 中 | 中等优先级 |
| high | 高 | 高优先级 |
| urgent | 紧急 | 紧急，需要立即处理 |

---

### 3. 查询工单详情（按 ID）

**请求**

```http
GET /open/ticket/findTicket?ID=1
X-Api-Token: your_token_here
```

**请求参数 (Query Parameters)**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ID | integer | 是 | 工单 ID |

---

### 4. 查询工单详情（按工单编号）

**请求**

```http
GET /open/ticket/getTicketByTicketNo?ticketNo=ding@a1b2c3d4e5f678901234567890123456
X-Api-Token: your_token_here
```

**请求参数 (Query Parameters)**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ticketNo | string | 是 | 工单编号 |

**响应示例 (成功)**

```json
{
  "code": 0,
  "msg": "查询成功",
  "data": {
    "id": 1,
    "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
    "title": "无法登录系统",
    "description": "用户反馈无法登录系统，提示账号或密码错误",
    "submitter": "张三",
    "submitterPhone": "13800138000",
    "submitterSource": "dingtalk",
    "category": "技术支持",
    "priority": "high",
    "status": "pending",
    "createdAt": "2026-04-16T10:00:00Z",
    "updatedAt": "2026-04-16T10:00:00Z"
  }
}
```

**响应示例 (工单不存在)**

```json
{
  "code": 404,
  "msg": "工单不存在",
  "data": null
}
```

---

### 5. 更新工单

**请求**

```http
PUT /open/ticket/updateTicket
Content-Type: application/json
X-Api-Token: your_token_here
```

**请求参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ticketNo | string | 是 | 工单编号（AI 客户端按规则生成：`渠道号@MD5(提交人 ID| 标题 | 类别)`） |
| title | string | 是 | 工单标题 (2-200 字符) |
| description | string | 是 | 工单描述 (至少 10 字符) |
| category | string | 是 | 工单类别 |
| priority | string | 是 | 紧急程度 (low/medium/high/urgent) |
| status | string | 是 | 工单状态 (pending/processing/resolved/closed/rejected) |
| handler | string | 否 | 处理人姓名 |
| handlerDept | string | 否 | 处理部门 |
| remark | string | 否 | 处理备注 |
| attachment | string | 否 | 附件路径 |
| satisfaction | string | 否 | 满意度 (very_dissatisfied/dissatisfied/neutral/satisfied/very_satisfied) |

**请求示例**

```json
{
  "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
  "title": "更新后的工单标题",
  "description": "更新后的工单描述",
  "category": "技术支持",
  "priority": "high",
  "status": "resolved",
  "handler": "李四",
  "handlerDept": "技术部",
  "remark": "处理备注",
  "satisfaction": "very_satisfied"
}
```

## 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误（验证失败） |
| 401 | 认证失败（Token 缺失、无效或过期） |
| 404 | 资源不存在（工单不存在） |
| 500 | 服务器内部错误 |

## cURL 完整示例

### 创建工单

```bash
# 1. 先计算工单编号（渠道号@MD5(提交人 ID| 标题 | 类别)）
# 示例：ding@a1b2c3d4e5f678901234567890123456

curl -X POST http://localhost:8888/open/ticket/createTicket \
  -H "X-Api-Token: your_token_here" \
  -H "Content-Type: application/json" \
  -d '{
    "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
    "title": "测试工单",
    "description": "这是一个通过 API 创建的测试工单，用于验证接口功能",
    "category": "技术支持",
    "priority": "medium",
    "submitterID": "user_openid_123",
    "submitter": "AI 助手",
    "submitterPhone": "13800138000",
    "submitterSource": "dingtalk"
  }'
```

**响应示例（创建成功）**：
```json
{
  "code": 0,
  "msg": "创建成功，工单编号：ding@a1b2c3d4e5f678901234567890123456",
  "data": {
    "duplicate": false,
    "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
    "id": 1
  }
}
```

**响应示例（工单已存在）**：
```json
{
  "code": 0,
  "msg": "检测到相似工单，返回已存在的工单",
  "data": {
    "duplicate": true,
    "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
    "id": 1,
    "title": "测试工单",
    "status": "pending"
  }
}
```

### 查询工单列表

```bash
curl -X GET "http://localhost:8888/open/ticket/getTicketList?page=1&pageSize=10&status=pending" \
  -H "X-Api-Token: your_token_here"
```

### 查询工单详情（按 ID）

```bash
curl -X GET "http://localhost:8888/open/ticket/findTicket?ID=1" \
  -H "X-Api-Token: your_token_here"
```

### 查询工单详情（按工单编号）

```bash
# 使用 AI 生成的工单编号查询
curl -X GET "http://localhost:8888/open/ticket/getTicketByTicketNo?ticketNo=ding@a1b2c3d4e5f678901234567890123456" \
  -H "X-Api-Token: your_token_here"
```

### 更新工单

```bash
curl -X PUT http://localhost:8888/open/ticket/updateTicket \
  -H "X-Api-Token: your_token_here" \
  -H "Content-Type: application/json" \
  -d '{
    "ticketNo": "ding@a1b2c3d4e5f678901234567890123456",
    "title": "更新的工单标题",
    "description": "更新的工单描述",
    "category": "技术支持",
    "priority": "high",
    "status": "resolved",
    "remark": "已处理完成"
  }'
```

### 完整脚本示例（生成编号 + 创建工单）

```bash
#!/bin/bash

# 工单信息
SUBMITTER_ID="user_openid_123"
TITLE="无法登录系统"
CATEGORY="技术支持"
SOURCE="dingtalk"

# 1. 计算 MD5（需要安装 md5sum 或 openssl）
CONTENT="${SUBMITTER_ID}|${TITLE}|${CATEGORY}"
if command -v md5sum &> /dev/null; then
    HASH=$(echo -n "$CONTENT" | md5sum | cut -d' ' -f1)
else
    HASH=$(echo -n "$CONTENT" | openssl md5 | cut -d' ' -f2)
fi

# 2. 生成工单编号
TICKET_NO="ding@${HASH}"
echo "生成的工单编号：${TICKET_NO}"

# 3. 创建工单
curl -X POST http://localhost:8888/open/ticket/createTicket \
  -H "X-Api-Token: your_token_here" \
  -H "Content-Type: application/json" \
  -d "{
    \"ticketNo\": \"${TICKET_NO}\",
    \"title\": \"${TITLE}\",
    \"description\": \"用户反馈无法登录系统，提示账号或密码错误\",
    \"category\": \"${CATEGORY}\",
    \"priority\": \"medium\",
    \"submitterID\": \"${SUBMITTER_ID}\",
    \"submitter\": \"张三\",
    \"submitterPhone\": \"13800138000\",
    \"submitterSource\": \"${SOURCE}\"
  }"
```

## 安全建议

1. **Token 保管**: 不要在客户端代码中暴露 Token，应在服务端中转请求
2. **权限控制**: 为 API Token 分配最小必要权限
3. **定期轮换**: 建议定期更换 Token
4. **监控日志**: 开启操作日志记录，便于审计
5. **限流保护**: 建议在网关层面对 API 进行限流，防止滥用
