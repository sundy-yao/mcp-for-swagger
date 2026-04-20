# 构建和服务脚本使用指南

## build.sh - 构建脚本

### 用法

```bash
# 默认构建 (当前平台)
./build.sh

# 开发环境构建
./build.sh dev

# 测试环境构建 (linux/amd64)
./build.sh test

# 生产环境构建 (带优化，运行测试)
./build.sh prod

# 构建所有平台
./build.sh all

# 清理构建文件
./build.sh clean

# 清理指定平台
./build.sh clean linux
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GOOS` | 当前系统 | 目标操作系统 |
| `GOARCH` | amd64 | 目标架构 |
| `OUTPUT_DIR` | ./bin | 输出目录 |

### 示例

```bash
# macOS ARM64 开发构建
GOOS=darwin GOARCH=arm64 ./build.sh dev

# Linux ARM64 生产构建
GOOS=linux GOARCH=arm64 ./build.sh prod

# 交叉编译到 Windows
GOOS=windows GOARCH=amd64 ./build.sh prod
```

### 构建输出

```
./bin/
├── mcp-for-swagger-darwin-arm64   # macOS ARM
├── mcp-for-swagger-darwin-amd64    # macOS Intel
├── mcp-for-swagger-linux-amd64     # Linux x86_64
├── mcp-for-swagger-linux-arm64     # Linux ARM
├── mcp-for-swagger-windows-amd64   # Windows x86_64
└── mcp-for-swagger-windows-arm64   # Windows ARM
```

---

## service.sh - 生产环境服务脚本

### 用法

```bash
# 启动服务
./service.sh start

# 停止服务
./service.sh stop

# 重启服务
./service.sh restart

# 查看状态
./service.sh status

# 查看日志
./service.sh logs
./service.sh logs -n 100      # 最近 100 行
./service.sh logs -f          # 实时查看

# 健康检查
./service.sh health

# 检查配置和依赖
./service.sh check

# 重载配置 (发送 SIGHUP)
./service.sh reload
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVICE_USER` | 当前用户 | 运行服务的用户 |
| `HEALTH_PORT` | 8000 | 健康检查端口 |
| `CONFIG_PATH` | config.yaml | 配置文件路径 |

### 示例

```bash
# 启动服务
./service.sh start

# 查看服务状态
./service.sh status

# 实时查看日志
./service.sh logs -f

# 健康检查
./service.sh health

# 检查配置
./service.sh check
```

### 输出示例

#### 状态检查
```
========================================
  mcp-for-swagger 状态
========================================

状态：运行中
PID: 12345


健康状态：健康

日志文件:
  stdout: ./logs/mcp-for-swagger.log
  stderr: ./logs/mcp-for-swagger.error.log
  大小：1.2M
```

#### 健康检查
```
[INFO] 执行健康检查...
[SUCCESS] 健康检查通过
响应：{"status":"ok","sessions":0}
```

#### 配置检查
```
========================================
  mcp-for-swagger 检查
========================================

[INFO] 检查二进制文件...
[SUCCESS] 二进制文件存在：./bin/mcp-for-swagger-darwin-arm64
[INFO] 检查配置文件...
[SUCCESS] 配置文件存在：./config.yaml
[INFO] 检查日志目录...
[SUCCESS] 日志目录存在：./logs
[INFO] 检查端口 8000...
[SUCCESS] 端口 8000 可用

[SUCCESS] 检查完成，无明显问题
```

---

## 典型使用流程

### 开发环境

```bash
# 1. 构建开发版本
./build.sh dev

# 2. 启动服务
./service.sh start

# 3. 查看状态
./service.sh status

# 4. 查看日志
./service.sh logs -f

# 5. 停止服务
./service.sh stop
```

### 生产环境部署

```bash
# 1. 构建生产版本 (包含测试)
./build.sh prod

# 2. 检查配置
./service.sh check

# 3. 启动服务
./service.sh start

# 4. 健康检查
./service.sh health

# 5. 查看日志
./service.sh logs -n 50
```

### 部署到服务器

```bash
# 1. 交叉编译到 Linux
GOOS=linux GOARCH=amd64 ./build.sh prod

# 2. 上传到服务器
scp bin/mcp-for-swagger-linux-amd64 user@server:/opt/mcp/bin/
scp config.yaml user@server:/opt/mcp/config.yaml
scp service.sh user@server:/opt/mcp/

# 3. 在服务器上
ssh user@server
cd /opt/mcp
./service.sh check
./service.sh start
./service.sh health
```

---

## 文件结构

```
project/
├── bin/                    # 编译输出目录
│   └── mcp-for-swagger-*
├── logs/                   # 日志目录
│   ├── mcp-for-swagger.log
│   └── mcp-for-swagger.error.log
├── .pid/                   # PID 文件目录
│   └── mcp-for-swagger.pid
├── build.sh                # 构建脚本
├── service.sh              # 服务脚本
├── config.yaml             # 配置文件
└── .env                    # 环境变量 (可选)
```

## 故障排除

### 服务无法启动

1. 检查日志：`./service.sh logs`
2. 检查配置：`./service.sh check`
3. 检查端口占用：`lsof -i:8000`

### 二进制文件不存在

运行构建脚本：
```bash
./build.sh dev    # 开发环境
./build.sh prod   # 生产环境
```

### 健康检查失败

1. 确认服务已启动：`./service.sh status`
2. 检查健康端点：`curl http://localhost:8000/health`
3. 查看错误日志：`tail -100 logs/mcp-for-swagger.error.log`
