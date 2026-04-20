#!/bin/bash

###############################################################################
# MCP OpenAPI Service 构建脚本
#
# 用法:
#   ./build.sh                    # 默认构建 (linux/amd64)
#   ./build.sh dev                # 开发环境构建
#   ./build.sh test               # 测试环境构建
#   ./build.sh prod               # 生产环境构建
#   ./build.sh all                # 构建所有平台
#   ./build.sh clean              # 清理构建文件
#   ./build.sh clean linux        # 清理指定平台
###############################################################################

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目信息
PROJECT_NAME="mcp-for-swagger"
MAIN_PACKAGE="./cmd/server"
OUTPUT_DIR="./bin"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 默认配置
ENV=${1:-dev}
GOOS=${GOOS:-linux}
GOARCH=${GOARCH:-amd64}

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
用法：$0 [选项]

选项:
    dev         开发环境构建 (当前平台)
    test        测试环境构建 (linux/amd64)
    prod        生产环境构建 (linux/amd64, 优化编译)
    all         构建所有平台 (linux/windows/darwin)
    clean       清理所有构建文件
    clean <os>  清理指定平台构建文件

环境变量:
    GOOS        目标操作系统 (默认：当前系统或 linux)
    GOARCH      目标架构 (默认：amd64)
    OUTPUT_DIR  输出目录 (默认：./bin)

示例:
    $0                      # 默认构建
    $0 prod                 # 生产环境构建
    $0 all                  # 构建所有平台
    GOOS=darwin $0 dev      # macOS 开发构建
    $0 clean linux          # 清理 linux 构建文件
EOF
    exit 1
}

# 清理函数
clean() {
    local target=${1:-all}

    if [[ "$target" == "all" ]]; then
        info "清理所有构建文件..."
        rm -rf "${OUTPUT_DIR}"
    else
        info "清理 ${target} 构建文件..."
        case "$target" in
            linux)
                rm -rf "${OUTPUT_DIR}/${PROJECT_NAME}-linux-"*
                ;;
            windows)
                rm -rf "${OUTPUT_DIR}/${PROJECT_NAME}-windows-"*
                ;;
            darwin)
                rm -rf "${OUTPUT_DIR}/${PROJECT_NAME}-darwin-"*
                ;;
            *)
                rm -rf "${OUTPUT_DIR}/${PROJECT_NAME}-${target}"*
                ;;
        esac
    fi

    success "清理完成"
}

# 单个平台构建
build() {
    local env=$1
    local os=$2
    local arch=$3

    local output_name="${PROJECT_NAME}-${os}-${arch}"
    local output_path="${OUTPUT_DIR}/${output_name}"

    if [[ "$os" == "windows" ]]; then
        output_path="${output_path}.exe"
    fi

    info "构建 ${env} 环境 (${os}/${arch})..."

    # 设置编译参数
    local ldflags="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

    if [[ "$env" == "prod" ]]; then
        ldflags="${ldflags} -s -w"  # 移除调试信息，减小文件体积
    fi

    # 创建输出目录
    mkdir -p "${OUTPUT_DIR}"

    # 编译
    CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build \
        -ldflags "${ldflags}" \
        -o "${output_path}" \
        "${MAIN_PACKAGE}"

    if [[ $? -eq 0 ]]; then
        local size=$(du -h "${output_path}" | cut -f1)
        success "构建完成：${output_path} (${size})"
    else
        error "构建失败"
        exit 1
    fi
}

# 构建所有平台
build_all() {
    local env=$1

    info "构建所有平台 (${env})..."

    # Linux
    build "${env}" "linux" "amd64"
    build "${env}" "linux" "arm64"

    # macOS
    build "${env}" "darwin" "amd64"
    build "${env}" "darwin" "arm64"

    # Windows
    build "${env}" "windows" "amd64"
    build "${env}" "windows" "arm64"

    success "所有平台构建完成"

    # 显示构建结果
    echo ""
    info "构建文件列表:"
    ls -lh "${OUTPUT_DIR}/"
}

# 检查 Go 环境
check_go() {
    if ! command -v go &> /dev/null; then
        error "Go 未安装，请先安装 Go"
        exit 1
    fi

    local go_version=$(go version)
    info "Go 版本：${go_version}"
}

# 检查依赖
check_deps() {
    info "检查 Go 模块依赖..."
    go mod download
    go mod verify
    success "依赖检查完成"
}

# 运行测试
run_tests() {
    local env=$1

    if [[ "$env" == "prod" ]]; then
        info "运行测试套件..."
        go test -race -cover ./... || {
            error "测试失败"
            exit 1
        }
        success "测试通过"
    else
        info "运行测试套件 (无竞争检测)..."
        go test ./... || {
            warn "部分测试失败，继续构建"
        }
    fi
}

# 主函数
main() {
    echo "========================================"
    echo "  MCP OpenAPI Service 构建脚本"
    echo "========================================"
    echo ""

    # 处理 clean 命令
    if [[ "$ENV" == "clean" ]]; then
        clean "${2:-all}"
        exit 0
    fi

    # 处理 all 命令
    if [[ "$ENV" == "all" ]]; then
        check_go
        check_deps
        build_all "prod"
        exit 0
    fi

    # 处理 help
    if [[ "$ENV" == "help" ]] || [[ "$ENV" == "-h" ]] || [[ "$ENV" == "--help" ]]; then
        usage
    fi

    # 检查 Go 环境
    check_go

    # 确定当前系统
    if [[ "${GOOS}" == "linux" ]] && [[ "$(uname)" == "Darwin" ]]; then
        # macOS 上构建 linux 目标需要特殊处理
        if [[ "$ENV" == "dev" ]]; then
            GOOS=darwin
        fi
    fi

    # 检查依赖
    check_deps

    # 运行测试 (生产环境)
    if [[ "$ENV" == "prod" ]]; then
        run_tests "${ENV}"
    fi

    # 构建
    case "$ENV" in
        dev)
            build "dev" "$(go env GOOS)" "$(go env GOARCH)"
            ;;
        test|prod)
            build "${ENV}" "${GOOS}" "${GOARCH}"
            ;;
        *)
            error "未知的环境：${ENV}"
            usage
            ;;
    esac

    echo ""
    info "构建信息:"
    echo "  环境：${ENV}"
    echo "  版本：${VERSION}"
    echo "  构建时间：${BUILD_TIME}"
    echo "  Git 提交：${GIT_COMMIT}"
    echo ""
}

# 执行主函数
main "$@"
