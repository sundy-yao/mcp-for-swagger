# MCP OpenAPI Service - Development Guidelines

## Project Overview

A Golang-based MCP (Model Context Protocol) service that automatically converts backend RESTful OpenAPI interfaces into MCP services through YAML configuration.

**Repository**: `github.com/sundy-yao/mcp-for-swagger`

## 快速开始（新员工必读）

### 1. Environment Setup

```bash
# required: Go 1.26+
go version

# Clone repository
git clone git@github.com:sundy-yao/mcp-for-swagger.git
cd mcp-for-swagger

# Install dependencies
go mod tidy
```

### 2. Local Development Configuration

```bash
# Copy environment configuration template
cp .env.example .env.local

# Edit .env.local with actual configuration values
# vi .env.local

# Load environment variables and run
source .env.local && go run cmd/server/main.go
```

### 3. Configuration Files

| File | Purpose | Committed |
|------|---------|-----------|
| `config.yaml` | Main configuration template (no sensitive info) | ✅ Yes |
| `.env.example` | Environment variables example | ✅ Yes |
| `.env.local` | Local environment config (sensitive info) | ❌ No |
| `config.local.yaml` | Local config overrides | ❌ No |

### 4. Common Commands

```bash
# Run in development mode
go run cmd/server/main.go

# Build for production
go build -o bin/mcp-server cmd/server/main.go

# Code formatting
gofmt -w ./...

# Code linting
go vet ./...

# Run tests
go test ./... -v

# List dependencies
go list -m all
```

## Project Structure

```
mcp-for-swagger/
├── cmd/
│   ├── server/
│   │   └── main.go          # Main entry point
│   └── test/
│       └── backend_checker.go # Backend integration test tool
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration loading and parsing
│   ├── httpclient/
│   │   └── client.go        # HTTP client
│   ├── logger/
│   │   └── logger.go        # Logging module
│   ├── mcp/
│   │   ├── server.go        # MCP server core
│   │   ├── types/
│   │   │   └── types.go     # MCP type definitions
│   │   ├── registry/
│   │   │   └── registry.go  # Tool registry
│   │   └── transport/
│   │       └── sse.go       # SSE transport layer
│   ├── openapi/
│   │   └── parser.go        # OpenAPI parser
│   └── tools/
│       └── registrar.go     # Tool registrar
├── doc/
│   └── openapi.yaml         # OpenAPI specification file
├── config.yaml              # Main configuration file
├── .env.example             # Environment variables example
├── .env.local               # Local environment config (gitignore)
└── go.mod
```

## Development Guidelines

### Code Style

- **Go Version**: Go 1.26+
- **Formatting**: Must use `gofmt` for code formatting
- **Naming**:
  - Exported types: Start with uppercase (e.g., `Client`, `Request`)
  - Private types: Start with lowercase (e.g., `clientConfig`)
  - Interfaces: `-er` suffix (e.g., `Parser`, `Handler`)
- **Package names**: Use lowercase, avoid underscores

### Error Handling

```go
// ✅ Correct: Error wrapping with context
if err != nil {
    return fmt.Errorf("failed to parse config: %w", err)
}

// ✅ Correct: Logging errors
logger.Error("Failed to load config: %v", err)

// ❌ Wrong: Ignoring errors
doSomething()  // No err check

// ❌ Wrong: Returning err without context
return err
```

### Logging

```go
// Log levels: DEBUG, INFO, WARN, ERROR

logger.Debug("Debug info: %v", data)     // Development debugging
logger.Info("Server started on %s:%d", host, port) // Key workflow
logger.Warn("Missing config, using default")       // Recoverable issue
logger.Error("Database connection failed: %v", err) // Critical error
```

### Comments

```go
// Package comment: Before package declaration
// Package config provides configuration loading and management
package config

// Type comment: Exported types must have comments
// Settings configuration struct
type Settings struct { ... }

// Function comment: Exported functions must have comments
// LoadConfig loads configuration from YAML file
func LoadConfig(path string) (*Settings, error) { ... }
```

### Git Commit Convention

```bash
# Format: <type>(<scope>): <subject>
# type: feat, fix, docs, style, refactor, test, chore

# Examples
git commit -m "feat(config): add environment variable placeholder support"
git commit -m "fix(mcp): fix SSE connection leak"
git commit -m "docs: update development guidelines"
```

## Architecture

### Core Components

| Component | Package | Responsibility |
|-----------|---------|----------------|
| config | `internal/config` | Configuration loading, supports YAML and env vars |
| openapi | `internal/openapi` | Parse OpenAPI specifications |
| tools | `internal/tools` | Register API endpoints as MCP tools |
| httpclient | `internal/httpclient` | Encapsulate HTTP requests |
| mcp | `internal/mcp` | MCP protocol server |

### Data Flow

```
1. Load config.yaml
        ↓
2. Parse doc/openapi.yaml
        ↓
3. Create httpclient (configure backend API URL and auth)
        ↓
4. Register tools (API endpoints → MCP tools)
        ↓
5. Start MCP SSE server
        ↓
6. Handle MCP requests (tools/list, tools/call)
```

### Configuration Priority

```
Environment variables > config.local.yaml > config.yaml > Defaults
```

## Adding New MCP Tools/Endpoints

### Step 1: Update OpenAPI Specification

Edit `doc/openapi.yaml` to add new API endpoints:

```yaml
paths:
  /api/v1/orders/{id}:
    get:
      operationId: getOrderById
      summary: Get order by ID
      tags:
        - orders
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
```

### Step 2: Configure Tool Mapping (Optional)

Edit `config.yaml` to configure tool prefix or filtering:

```yaml
tool_mapping:
  prefix: "order"          # Tool name becomes order_getOrderById
  exclude:                 # Exclude endpoints
    - "healthCheck"
  include_tags:            # Only include specific tags
    - "orders"
```

### Step 3: Verify

```bash
# Run backend test tool
go run cmd/test/backend_checker.go

# Start service and check tool list
go run cmd/server/main.go
```

## Debugging Tips

### 1. Enable Debug Logging

```bash
# Set in config.yaml
log:
  level: "DEBUG"
```

### 2. Check SSE Connection

Visit `http://localhost:8080/sse` to view SSE connection status

### 3. Health Check

```bash
curl http://localhost:8080/health
```

## FAQ

### Q: How to add new authentication methods?

**A**: Add authentication logic in `internal/httpclient/client.go`, or configure in `config.yaml` headers:

```yaml
openapi:
  headers:
    - "Authorization: Bearer ${API_TOKEN}"
    - "X-Custom-Header: value"
```

### Q: How to exclude specific API endpoints?

**A**: Add operationId or path to `tool_mapping.exclude` in `config.yaml`:

```yaml
tool_mapping:
  exclude:
    - "internalHealthCheck"  # operationId
    - "/debug/vars"          # path
```

### Q: How to connect to internal network API during local development?

**A**: Use `.env.local` to configure internal network address:

```bash
# .env.local
API_BASE_URL=http://api.internal.company.com:8080
API_AUTH_TOKEN=your-token
```

## Security Notes

1. **Sensitive Info**: Never commit tokens, passwords, or other sensitive information to git
2. **Config Files**: Use `.env.local` or `config.local.yaml` for local configuration
3. **Code Review**: Check for hardcoded credentials before committing

## Further Reading

- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [OpenAPI 3.0 Specification](https://swagger.io/specification/)
- [Go Official Documentation](https://go.dev/doc/)
