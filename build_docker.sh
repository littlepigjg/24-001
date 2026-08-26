#!/bin/bash
# =============================================================================
# Code Sandbox Docker 构建脚本
# =============================================================================
# 用途：构建 Code Sandbox 项目的 Docker 镜像
# 作者：Code Sandbox Team
# 日期：2024
# =============================================================================

set -euo pipefail

# ==================== 配置 ====================
IMAGE_NAME="codesandbox"
IMAGE_TAG="latest"
DOCKERFILE_PATH="./Dockerfile"
BUILD_CONTEXT="."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ==================== 函数 ====================
print_header() {
    echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║   Code Sandbox Docker Image Builder          ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
    echo ""
}

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装或未在 PATH 中找到"
        print_info "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    # 检查 Docker daemon 是否运行
    if ! docker info &> /dev/null; then
        print_error "Docker daemon 未运行"
        print_info "请启动 Docker 服务"
        exit 1
    fi
}

# 检查必要文件
check_files() {
    if [ ! -f "$DOCKERFILE_PATH" ]; then
        print_error "Dockerfile 不存在: $DOCKERFILE_PATH"
        exit 1
    fi

    if [ ! -f "go.mod" ]; then
        print_error "go.mod 文件不存在"
        exit 1
    fi

    if [ ! -d "cmd/server" ]; then
        print_error "cmd/server 目录不存在"
        exit 1
    fi
}

# 清理旧镜像
clean_old_images() {
    local force=${1:-false}
    if docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" &>/dev/null; then
        if [ "$force" = "true" ]; then
            print_warning "正在移除旧镜像 ${IMAGE_NAME}:${IMAGE_TAG} ..."
            docker rmi "${IMAGE_NAME}:${IMAGE_TAG}" 2>/dev/null || true
        else
            print_warning "镜像 ${IMAGE_NAME}:${IMAGE_TAG} 已存在"
            read -p "是否重新构建？(y/N): " confirm
            if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
                print_info "构建已取消"
                exit 0
            fi
            docker rmi "${IMAGE_NAME}:${IMAGE_TAG}" 2>/dev/null || true
        fi
    fi
}

# 构建镜像
build_image() {
    local no_cache=${1:-false}
    local platform=${2:-""}

    print_info "开始构建镜像: ${IMAGE_NAME}:${IMAGE_TAG}"
    print_info "Dockerfile: $DOCKERFILE_PATH"
    print_info "构建上下文: $BUILD_CONTEXT"

    local build_args=(
        -f "$DOCKERFILE_PATH"
        -t "${IMAGE_NAME}:${IMAGE_TAG}"
        --progress=plain
    )

    if [ "$no_cache" = "true" ]; then
        build_args+=(--no-cache)
    fi

    if [ -n "$platform" ]; then
        build_args+=(--platform "$platform")
    fi

    echo ""
    print_info "Docker 构建参数: ${build_args[*]}"
    echo ""

    docker build "${build_args[@]}" "$BUILD_CONTEXT"
    local build_status=$?

    if [ $build_status -ne 0 ]; then
        print_error "镜像构建失败 (退出码: $build_status)"
        exit 1
    fi

    echo ""
    print_info "镜像构建成功！"
}

# 显示镜像信息
show_image_info() {
    echo ""
    echo -e "${BLUE}────────────────────────────────────────────────${NC}"
    print_info "镜像名称: ${IMAGE_NAME}"
    print_info "镜像标签: ${IMAGE_TAG}"
    echo ""

    # 获取镜像大小
    local image_size=$(docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" --format='{{.Size}}' 2>/dev/null || echo "未知")
    if [ "$image_size" != "未知" ]; then
        size_mb=$((image_size / 1048576))
        size_gb=$(echo "scale=2; $image_size / 1073741824" | bc 2>/dev/null || echo "N/A")
        print_info "镜像大小: ${size_mb} MB (${size_gb} GB)"
    fi

    # 获取创建时间
    local created=$(docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" --format='{{.Created}}' 2>/dev/null || echo "未知")
    print_info "创建时间: $created"

    echo -e "${BLUE}────────────────────────────────────────────────${NC}"
    echo ""
}

# 运行测试容器
run_test_container() {
    print_info "运行测试容器验证镜像..."
    echo ""

    local container_name="codesandbox-test-$(date +%s)"

    # 运行容器并等待启动
    timeout 15 docker run --rm --name "$container_name" \
        -p 8080:8080 \
        "${IMAGE_NAME}:${IMAGE_TAG}" &

    local container_pid=$!

    # 等待容器启动
    print_info "等待容器启动..."
    sleep 5

    # 健康检查
    print_info "执行健康检查..."
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        print_info "健康检查通过 ✓"
    else
        print_warning "健康检查未通过（容器可能仍在启动中）"
    fi

    # 清理
    kill $container_pid 2>/dev/null || true
    docker stop "$container_name" 2>/dev/null || true

    echo ""
    print_info "测试完成"
}

# 清理临时容器
cleanup_test() {
    local containers=$(docker ps -a -q --filter "name=codesandbox-test" 2>/dev/null || true)
    if [ -n "$containers" ]; then
        print_info "清理测试容器..."
        docker rm -f $containers 2>/dev/null || true
    fi
}

# 显示使用方式
show_usage() {
    echo ""
    echo -e "${GREEN}使用方式:${NC}"
    echo ""
    echo "  构建镜像:"
    echo "    bash build_docker.sh"
    echo ""
    echo "  无缓存构建:"
    echo "    bash build_docker.sh --no-cache"
    echo ""
    echo "  构建后自动测试:"
    echo "    bash build_docker.sh --test"
    echo ""
    echo "  指定平台构建:"
    echo "    bash build_docker.sh --platform linux/amd64"
    echo "    bash build_docker.sh --platform linux/arm64"
    echo ""
    echo "  自定义镜像名和标签:"
    echo "    IMAGE_NAME=myapp IMAGE_TAG=v1.0 bash build_docker.sh"
    echo ""
    echo -e "${BLUE}运行容器:${NC}"
    echo ""
    echo "  docker run -d -p 8080:8080 \\"
    echo "    -v /path/to/data:/app/data \\"
    echo "    codesandbox:latest"
    echo ""
}

# ==================== 主程序 ====================
main() {
    print_header

    # 解析参数
    local no_cache="false"
    local do_test="false"
    local platform=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --no-cache)
                no_cache="true"
                shift
                ;;
            --test)
                do_test="true"
                shift
                ;;
            --platform)
                platform="$2"
                shift 2
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            *)
                print_error "未知参数: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # 支持环境变量覆盖
    if [ -n "${IMAGE_NAME_OVERRIDE:-}" ]; then
        IMAGE_NAME="$IMAGE_NAME_OVERRIDE"
    fi
    if [ -n "${IMAGE_TAG_OVERRIDE:-}" ]; then
        IMAGE_TAG="$IMAGE_TAG_OVERRIDE"
    fi

    print_info "目标镜像: ${IMAGE_NAME}:${IMAGE_TAG}"
    echo ""

    # 执行检查
    check_docker
    check_files

    # 清理旧镜像（如果存在）
    clean_old_images "true"

    # 执行构建
    trap cleanup_test EXIT

    build_image "$no_cache" "$platform"

    # 显示信息
    show_image_info

    # 运行测试（可选）
    if [ "$do_test" = "true" ]; then
        run_test_container
    fi

    echo ""
    print_info "全部完成！"
    echo ""
}

# 执行主程序
main "$@"
