#!/bin/bash

###############################################################################
# MCP OpenAPI Service 生产环境服务脚本
#
# 用法:
#   ./service.sh start      # 启动服务
#   ./service.sh stop       # 停止服务
#   ./service.sh restart    # 重启服务
#   ./service.sh status     # 查看状态
#   ./service.sh logs       # 查看日志
#   ./service.sh logs -f    # 实时查看日志
#   ./service.sh reload     # 重载配置 (发送 SIGHUP)
#   ./service.sh health     # 健康检查
###############################################################################

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置项
SERVICE_NAME="mcp-for-swagger"
SERVICE_USER="${SERVICE_USER:-root}"
SERVICE_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="${SERVICE_DIR}/bin"
LOG_DIR="${SERVICE_DIR}/logs"

# PID 文件 - 直接放在服务脚本同级目录，固定文件名为 pid，文件内容为进程号
PID_FILE="${SERVICE_DIR}/pid"

# 检测二进制文件 - 自动检测当前平台
detect_binary() {
    ensure_dirs

    local current_os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local current_arch=$(uname -m)

    # 转换架构名称
    case "${current_arch}" in
        x86_64) current_arch="amd64" ;;
        aarch64|arm64) current_arch="arm64" ;;
    esac

    # 优先查找匹配当前平台的二进制文件
    local binary_name="mcp-for-swagger-${current_os}-${current_arch}"
    local binary_path="${BIN_DIR}/${binary_name}"

    if [[ -f "${binary_path}" ]]; then
        BINARY_NAME="${binary_name}"
        BINARY_PATH="${binary_path}"
        return 0
    fi

    # 回退到 linux-amd64
    binary_name="mcp-for-swagger-linux-amd64"
    binary_path="${BIN_DIR}/${binary_name}"

    if [[ -f "${binary_path}" ]]; then
        BINARY_NAME="${binary_name}"
        BINARY_PATH="${binary_path}"
        return 0
    fi

    return 1
}

# 二进制文件
BINARY_NAME=""
BINARY_PATH=""

# 配置文件
CONFIG_FILE="${SERVICE_DIR}/config.yaml"
ENV_FILE="${SERVICE_DIR}/.env"

# 健康检查端点
HEALTH_ENDPOINT="/health"
# 尝试从配置文件读取端口，否则使用默认值
HEALTH_PORT="${HEALTH_PORT:-}"
if [[ -z "${HEALTH_PORT}" ]] && [[ -f "${CONFIG_FILE}" ]]; then
    # 尝试从 YAML 配置中读取端口
    HEALTH_PORT=$(grep -E "^\s*port:" "${CONFIG_FILE}" 2>/dev/null | head -1 | awk '{print $2}' | tr -d '"' | tr -d "'")
fi
HEALTH_PORT="${HEALTH_PORT:-8000}"

# 启动超时 (秒)
STARTUP_TIMEOUT=30

# 打印信息
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示用法
usage() {
    cat << EOF
用法：$0 <命令> [选项]

命令:
    start       启动服务
    stop        停止服务
    restart     重启服务
    status      查看服务状态
    logs        查看日志 (支持 tail 命令的所有选项)
    reload      重载配置 (发送 SIGHUP 信号)
    health      健康检查
    check       检查配置和依赖

选项:
    -f, --follow    实时查看日志 (tail -f)
    -n <num>        查看最近 N 行日志
    -e <env>        指定环境 (dev/test/prod)

示例:
    $0 start                    # 启动服务
    $0 stop                     # 停止服务
    $0 restart                  # 重启服务
    $0 status                   # 查看状态
    $0 logs -n 100              # 查看最近 100 行日志
    $0 logs -f                  # 实时查看日志
    $0 health                   # 健康检查
    $0 reload                   # 重载配置

环境变量:
    SERVICE_USER    运行服务的用户 (默认：当前用户)
    HEALTH_PORT     健康检查端口 (默认：8000)
    CONFIG_PATH     配置文件路径
EOF
    exit 1
}

# 创建必要目录
ensure_dirs() {
    mkdir -p "${BIN_DIR}"
    mkdir -p "${LOG_DIR}"
    # PID 文件直接放在服务目录，不需要单独目录
}

# 检查二进制文件
check_binary() {
    if ! detect_binary; then
        error "二进制文件不存在"
        info "请先运行构建脚本：./build.sh prod"
        exit 1
    fi

    info "使用二进制文件：${BINARY_PATH}"

    if [[ ! -x "${BINARY_PATH}" ]]; then
        info "添加执行权限..."
        chmod +x "${BINARY_PATH}"
    fi
}

# 检查配置文件
check_config() {
    if [[ ! -f "${CONFIG_FILE}" ]]; then
        warn "配置文件不存在：${CONFIG_FILE}"
    fi
}

# PID 文件已固定为 ${SERVICE_DIR}/pid，不需要动态设置

# 查找 PID 文件
find_pid_file() {
    if [[ -f "${SERVICE_DIR}/pid" ]]; then
        echo "${SERVICE_DIR}/pid"
        return 0
    fi
    return 1
}

# 获取进程 ID
get_pid() {
    # 尝试从 PID 文件获取
    local pid_file="${SERVICE_DIR}/pid"
    if [[ -f "${pid_file}" ]]; then
        local pid=$(cat "${pid_file}")
        if ps -p "${pid}" > /dev/null 2>&1; then
            echo "${pid}"
            return 0
        else
            # 进程不存在，清理旧的 PID 文件
            rm -f "${pid_file}"
        fi
    fi

    # 尝试通过完整路径查找进程
    local pid=$(pgrep -af "${BINARY_PATH}" 2>/dev/null | head -1 | awk '{print $1}')
    if [[ -n "${pid}" ]]; then
        echo "${pid}"
        return 0
    fi

    return 1
}

# 检查服务状态
check_status() {
    local pid=$(get_pid)
    if [[ -n "${pid}" ]]; then
        echo "${pid}"
        return 0
    fi
    return 1
}

# 启动服务
start() {
    ensure_dirs
    check_binary

    info "检查服务状态..."
    if check_status > /dev/null 2>&1; then
        local pid=$(check_status)
        warn "服务已在运行 (PID: ${pid})"
        return 0
    fi

    info "启动 ${SERVICE_NAME}..."

    # 构建启动命令
    local cmd="${BINARY_PATH}"

    # 配置文件
    if [[ -f "${CONFIG_FILE}" ]]; then
        cmd="${cmd}"
    fi

    # 环境变量文件
    local env_args=""
    if [[ -f "${ENV_FILE}" ]]; then
        env_args="--env-file ${ENV_FILE}"
    fi

    # 启动进程
    # 注意：应用自身已将日志输出到 logs/app.log，无需重定向 stdout/stderr
    # 这里使用 /dev/null 避免重复日志
    cd "${SERVICE_DIR}"

    nohup "${BINARY_PATH}" \
        > /dev/null 2> /dev/null &

    local pid=$!
    # 写入 PID 文件（固定文件名为 pid）
    echo "${pid}" > "${SERVICE_DIR}/pid"

    info "等待服务启动..."

    # 等待启动
    local count=0
    while [[ ${count} -lt ${STARTUP_TIMEOUT} ]]; do
        if ps -p "${pid}" > /dev/null 2>&1; then
            if check_health_silent; then
                success "服务已启动 (PID: ${pid})"
                return 0
            fi
        else
            error "服务进程已退出"
            # 应用日志在 logs/app.log
            tail -50 "${LOG_DIR}/app.log"
            return 1
        fi

        sleep 1
        count=$((count + 1))
    done

    error "服务启动超时 (${STARTUP_TIMEOUT}秒)"
    return 1
}

# 停止服务
stop() {
    info "停止 ${SERVICE_NAME}..."

    local pid_file="${SERVICE_DIR}/pid"
    local pid=""

    if [[ -f "${pid_file}" ]]; then
        pid=$(cat "${pid_file}")
    fi

    # 如果 PID 文件不存在或无效，尝试通过完整路径查找
    if [[ -z "${pid}" ]] || ! ps -p "${pid}" > /dev/null 2>&1; then
        # 需要先调用 detect_binary 获取 BINARY_PATH
        if detect_binary; then
            pid=$(pgrep -af "${BINARY_PATH}" 2>/dev/null | head -1 | awk '{print $1}')
        fi
    fi

    if [[ -z "${pid}" ]]; then
        warn "服务未运行"
        rm -f "${pid_file}"
        return 0
    fi

    info "发送 SIGTERM 信号到进程 ${pid}..."
    kill -15 "${pid}" 2>/dev/null || true

    # 等待进程退出
    local count=0
    while [[ ${count} -lt 30 ]]; do
        if ! ps -p "${pid}" > /dev/null 2>&1; then
            success "服务已停止"
            rm -f "${pid_file}"
            return 0
        fi

        sleep 1
        count=$((count + 1))
    done

    # 强制终止
    warn "进程未正常退出，发送 SIGKILL..."
    kill -9 "${pid}" 2>/dev/null || true
    rm -f "${pid_file}"

    success "服务已强制停止"
}

# 重启服务
restart() {
    info "重启 ${SERVICE_NAME}..."
    stop
    sleep 2
    start
}

# 重载配置
reload() {
    info "重载配置..."

    local pid=$(get_pid)
    if [[ -z "${pid}" ]]; then
        error "服务未运行"
        return 1
    fi

    kill -HUP "${pid}"
    success "配置重载信号已发送 (PID: ${pid})"
}

# 查看状态
status() {
    echo "========================================"
    echo "  ${SERVICE_NAME} 状态"
    echo "========================================"
    echo ""

    local pid_file="${SERVICE_DIR}/pid"
    local pid=""

    if [[ -f "${pid_file}" ]]; then
        pid=$(cat "${pid_file}")
        if ! ps -p "${pid}" > /dev/null 2>&1; then
            # 进程不存在，清理旧的 PID 文件
            rm -f "${pid_file}"
            pid=""
        fi
    fi

    # 如果还是没有 PID，尝试通过完整路径查找
    if [[ -z "${pid}" ]]; then
        # 需要先调用 detect_binary 获取 BINARY_PATH
        if detect_binary; then
            pid=$(pgrep -af "${BINARY_PATH}" 2>/dev/null | head -1 | awk '{print $1}')
        fi
    fi

    if [[ -n "${pid}" ]]; then
        echo -e "状态：${GREEN}运行中${NC}"
        echo "PID: ${pid}"
        echo "PID 文件：${pid_file}"

        # 获取进程详细信息
        if ps -p "${pid}" -o pid,ppid,user,%cpu,%mem,etime,cmd > /dev/null 2>&1; then
            echo ""
            echo "进程信息:"
            ps -p "${pid}" -o pid,ppid,user,%cpu,%mem,etime,cmd
        fi

        # 健康检查
        echo ""
        if check_health_silent; then
            echo -e "健康状态：${GREEN}健康${NC}"
        else
            echo -e "健康状态：${RED}不健康${NC}"
        fi
    else
        echo -e "状态：${YELLOW}已停止${NC}"
        echo "PID 文件：无"
    fi

    echo ""
    echo "日志文件:"
    echo "  应用日志：${LOG_DIR}/app.log"

    if [[ -f "${LOG_DIR}/app.log" ]]; then
        echo "  大小：$(du -h "${LOG_DIR}/app.log" | cut -f1)"
    fi

    echo ""
}

# 查看日志
logs() {
    local args="$@"
    local app_log="${LOG_DIR}/app.log"

    if [[ ! -f "${app_log}" ]]; then
        error "日志文件不存在：${app_log}"
        return 1
    fi

    if [[ -n "${args}" ]]; then
        tail ${args} "${app_log}"
    else
        tail "${app_log}"
    fi
}

# 健康检查 (静默)
check_health_silent() {
    if command -v curl > /dev/null 2>&1; then
        curl -sf "http://localhost:${HEALTH_PORT}${HEALTH_ENDPOINT}" > /dev/null 2>&1
        return $?
    elif command -v wget > /dev/null 2>&1; then
        wget -q --spider "http://localhost:${HEALTH_PORT}${HEALTH_ENDPOINT}" 2>/dev/null
        return $?
    else
        return 1
    fi
}

# 健康检查
health() {
    info "执行健康检查..."

    local pid=$(get_pid)
    if [[ -z "${pid}" ]]; then
        error "服务未运行"
        return 1
    fi

    local response
    if command -v curl > /dev/null 2>&1; then
        response=$(curl -s "http://localhost:${HEALTH_PORT}${HEALTH_ENDPOINT}")
        local status=$?
    elif command -v wget > /dev/null 2>&1; then
        response=$(wget -qO- "http://localhost:${HEALTH_PORT}${HEALTH_ENDPOINT}")
        local status=$?
    else
        error "需要 curl 或 wget 命令"
        return 1
    fi

    if [[ ${status} -eq 0 ]]; then
        success "健康检查通过"
        echo "响应：${response}"
        return 0
    else
        error "健康检查失败"
        return 1
    fi
}

# 检查配置和依赖
check() {
    echo "========================================"
    echo "  ${SERVICE_NAME} 检查"
    echo "========================================"
    echo ""

    local has_error=0

    # 检查二进制文件
    info "检查二进制文件..."
    if detect_binary; then
        success "二进制文件存在：${BINARY_PATH}"
        "${BINARY_PATH}" --version 2>/dev/null || true
    else
        error "二进制文件不存在"
        info "请运行构建脚本：./build.sh prod 或 ./build.sh dev"
        has_error=1
    fi

    # 检查配置文件
    info "检查配置文件..."
    if [[ -f "${CONFIG_FILE}" ]]; then
        success "配置文件存在：${CONFIG_FILE}"
    else
        warn "配置文件不存在：${CONFIG_FILE}"
    fi

    # 检查环境变量文件
    info "检查环境变量文件..."
    if [[ -f "${ENV_FILE}" ]]; then
        success "环境变量文件存在：${ENV_FILE}"
    else
        warn "环境变量文件不存在：${ENV_FILE}"
    fi

    # 检查日志目录
    info "检查日志目录..."
    if [[ -d "${LOG_DIR}" ]]; then
        success "日志目录存在：${LOG_DIR}"
    else
        warn "日志目录不存在：${LOG_DIR}"
    fi

    # 检查端口占用
    info "检查端口 ${HEALTH_PORT}..."
    if command -v lsof > /dev/null 2>&1; then
        local port_pid=$(lsof -ti:${HEALTH_PORT} 2>/dev/null | head -1)
        if [[ -n "${port_pid}" ]]; then
            warn "端口 ${HEALTH_PORT} 被占用 (PID: ${port_pid})"
        else
            success "端口 ${HEALTH_PORT} 可用"
        fi
    fi

    echo ""
    if [[ ${has_error} -eq 0 ]]; then
        success "检查完成，无明显问题"
    else
        error "检查完成，发现问题"
        return 1
    fi
}

# 主函数
main() {
    local command="${1:-}"

    case "${command}" in
        start)
            start
            ;;
        stop)
            stop
            ;;
        restart)
            restart
            ;;
        status)
            status
            ;;
        logs)
                logs "${@:2}"
            ;;
        reload)
            reload
            ;;
        health)
            health
            ;;
        check)
            check
            ;;
        -h|--help|help)
            usage
            ;;
        "")
            usage
            ;;
        *)
            error "未知命令：${command}"
            usage
            ;;
    esac
}

# 执行主函数
main "$@"
