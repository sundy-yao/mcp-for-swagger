# MCP OpenAPI Service

[中文](#中文) | [English](#english)

---

## English

### Overview

A Golang-based MCP (Model Context Protocol) service that automatically converts backend RESTful OpenAPI interfaces into MCP tools through YAML configuration.

### Features

- Automatic parsing of OpenAPI 3.0 specifications
- Auto-register API endpoints as MCP tools
- SSE (Server-Sent Events) transport support
- Flexible YAML configuration
- Tool name filtering and exclusion
- Automatic input schema generation

### Quick Start

#### 1. Install Dependencies

```bash
go mod tidy
```

#### 2. Configure

Copy and edit the environment configuration:

```bash
cp .env.example .env.local
# Edit .env.local with your configuration
```

Edit `config.yaml`:

```yaml
mcp:
  host: "0.0.0.0"
  port: 8080
  name: "mcp-openapi-service"
  version: "1.0.0"
  transport: "sse"

openapi:
  path: "./doc/openapi.yaml"
  base_url: "${API_BASE_URL}"
  headers:
    - "X-Api-Token:${API_AUTH_TOKEN}"

tool_mapping:
  prefix: "api"
  exclude:
    - "healthCheck"
```

#### 3. Run

```bash
source .env.local && go run cmd/server/main.go
```

Or with environment variables:

```bash
MCP_PORT=9000 API_BASE_URL=http://api.example.com go run cmd/server/main.go
```

### Configuration

#### MCP Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| host | string | "0.0.0.0" | Server listen address |
| port | int | 8080 | Server port |
| name | string | "mcp-openapi-service" | Service name |
| version | string | "1.0.0" | Service version |
| transport | string | "sse" | Transport mode (sse/stdio) |

#### OpenAPI Settings

| Field | Type | Description |
|-------|------|-------------|
| path | string | OpenAPI YAML file path |
| url | string | OpenAPI URL (alternative to path) |
| base_url | string | Backend API base URL |
| headers | []string | HTTP headers (supports ${ENV} placeholders) |

#### Tool Mapping

| Field | Type | Description |
|-------|------|-------------|
| prefix | string | Tool name prefix |
| exclude | []string | Exclude operationId or path |
| include_tags | []string | Only include endpoints with specific tags |

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `/sse` | SSE connection endpoint |
| `/messages` | MCP message handler |
| `/health` | Health check |

### Example

Given this OpenAPI endpoint:

```yaml
/pets:
  get:
    operationId: "listPets"
    summary: "List all pets"
```

The generated MCP tool:

| Tool Name | Description |
|-----------|-------------|
| listPets | List all pets |

Call the tool:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "listPets",
    "arguments": {}
  }
}
```

### Development

#### Build

```bash
go build -o bin/mcp-server cmd/server/main.go
```

#### Test

```bash
# Unit tests
go test ./...

# Integration test
go run ./cmd/test/backend_checker.go
```

### Project Structure

```
mcp-for-swagger/
├── cmd/
│   ├── server/
│   │   └── main.go          # Main entry point
│   └── test/
│       └── backend_checker.go  # Backend integration test
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration loading
│   ├── httpclient/
│   │   └── client.go        # HTTP client
│   ├── logger/
│   │   └── logger.go        # Logging module
│   ├── mcp/
│   │   ├── server.go        # MCP server core
│   │   ├── types/
│   │   │   └── types.go     # MCP types
│   │   ├── registry/
│   │   │   └── registry.go  # Tool registry
│   │   └── transport/
│   │       └── sse.go       # SSE transport
│   ├── openapi/
│   │   └── parser.go        # OpenAPI parser
│   └── tools/
│       └── registrar.go     # Tool registrar
├── doc/
│   └── openapi.yaml         # OpenAPI specification
├── config.yaml              # Configuration file
├── .env.example             # Environment example
└── go.mod
```

### Architecture

```
1. Load config.yaml
        ↓
2. Parse doc/openapi.yaml
        ↓
3. Create httpclient (backend API URL + auth)
        ↓
4. Register tools (API endpoints → MCP tools)
        ↓
5. Start MCP SSE server
        ↓
6. Handle MCP requests (tools/list, tools/call)
```

### License

MIT

---

## 中文

### 项目概述

基于 Golang 的 MCP (Model Context Protocol) 服务，通过 YAML 配置自动将后端 RESTful OpenAPI 接口转换为 MCP 工具。

### 功能特性

- 自动解析 OpenAPI 3.0 规范文件
- 自动将 API 端点注册为 MCP 工具
- 支持 SSE (Server-Sent Events) 传输
- 灵活的 YAML 配置
- 支持工具名过滤和排除
- 自动构建工具输入 Schema

### 快速开始

#### 1. 安装依赖

```bash
go mod tidy
```

#### 2. 配置

复制并编辑环境配置：

```bash
cp .env.example .env.local
# 编辑 .env.local 填入配置
```

编辑 `config.yaml`：

```yaml
mcp:
  host: "0.0.0.0"
  port: 8080
  name: "mcp-openapi-service"
  version: "1.0.0"
  transport: "sse"

openapi:
  path: "./doc/openapi.yaml"
  base_url: "${API_BASE_URL}"
  headers:
    - "X-Api-Token:${API_AUTH_TOKEN}"

tool_mapping:
  prefix: "api"
  exclude:
    - "healthCheck"
```

#### 3. 运行

```bash
source .env.local && go run cmd/server/main.go
```

或使用环境变量：

```bash
MCP_PORT=9000 API_BASE_URL=http://api.example.com go run cmd/server/main.go
```

### 配置说明

#### MCP 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| host | string | "0.0.0.0" | 服务监听地址 |
| port | int | 8080 | 服务端口 |
| name | string | "mcp-openapi-service" | 服务名称 |
| version | string | "1.0.0" | 服务版本 |
| transport | string | "sse" | 传输方式 (sse/stdio) |

#### OpenAPI 配置

| 字段 | 类型 | 说明 |
|------|------|------|
| path | string | OpenAPI YAML 文件路径 |
| url | string | OpenAPI URL (与 path 二选一) |
| base_url | string | 后端 API 基础 URL |
| headers | []string | HTTP headers (支持 ${ENV} 占位符) |

#### 工具映射

| 字段 | 类型 | 说明 |
|------|------|------|
| prefix | string | 工具名前缀 |
| exclude | []string | 排除的 operationId 或 path |
| include_tags | []string | 只包含指定 tag 的端点 |

### 服务端点

| 端点 | 说明 |
|------|------|
| `/sse` | SSE 连接端点 |
| `/messages` | MCP 消息处理端点 |
| `/health` | 健康检查端点 |

### 示例

OpenAPI 端点：

```yaml
/pets:
  get:
    operationId: "listPets"
    summary: "List all pets"
```

生成的 MCP 工具：

| 工具名 | 描述 |
|--------|------|
| listPets | List all pets |

调用工具：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "listPets",
    "arguments": {}
  }
}
```

### 开发

#### 构建

```bash
go build -o bin/mcp-server cmd/server/main.go
```

#### 测试

```bash
# 单元测试
go test ./...

# 集成测试
go run ./cmd/test/backend_checker.go
```

### 项目结构

```
mcp-for-swagger/
├── cmd/
│   ├── server/
│   │   └── main.go          # 主程序入口
│   └── test/
│       └── backend_checker.go  # 后端集成测试
├── internal/
│   ├── config/
│   │   └── config.go        # 配置加载
│   ├── httpclient/
│   │   └── client.go        # HTTP 客户端
│   ├── logger/
│   │   └── logger.go        # 日志模块
│   ├── mcp/
│   │   ├── server.go        # MCP 服务器核心
│   │   ├── types/
│   │   │   └── types.go     # MCP 类型
│   │   ├── registry/
│   │   │   └── registry.go  # 工具注册表
│   │   └── transport/
│   │       └── sse.go       # SSE 传输层
│   ├── openapi/
│   │   └── parser.go        # OpenAPI 解析器
│   └── tools/
│       └── registrar.go     # 工具注册器
├── doc/
│   └── openapi.yaml         # OpenAPI 规范文件
├── config.yaml              # 配置文件
├── .env.example             # 环境变量示例
└── go.mod
```

### 架构说明

```
1. 加载 config.yaml
        ↓
2. 解析 doc/openapi.yaml
        ↓
3. 创建 httpclient (后端 API 地址 + 认证)
        ↓
4. 注册工具 (API 端点 → MCP 工具)
        ↓
5. 启动 MCP SSE 服务器
        ↓
6. 处理 MCP 请求 (tools/list, tools/call)
```

### 许可证

MIT
