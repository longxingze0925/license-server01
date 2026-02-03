#!/bin/bash
# ============================================
# License Server 自动更新脚本
# ============================================
# 功能：
#   - 从 GitHub 拉取最新代码
#   - 重新构建 Docker 镜像
#   - 平滑重启服务（零停机）
# ============================================
# 使用方法：
#   ./update.sh              # 更新到最新版本
#   ./update.sh v1.2.0       # 更新到指定版本
#   ./update.sh --force      # 强制更新（忽略本地修改）
# 环境变量：
#   GIT_TOKEN                # 私有仓库 Token（HTTPS）
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 私有仓库 Token 支持（HTTPS）
ORIGIN_URL=""
TOKEN_ACTIVE=false

restore_origin_url() {
    if [ "$TOKEN_ACTIVE" = true ] && [ -n "$ORIGIN_URL" ]; then
        git remote set-url origin "$ORIGIN_URL" >/dev/null 2>&1 || true
    fi
}

prepare_git_auth() {
    ORIGIN_URL=$(git remote get-url origin 2>/dev/null || echo "")
    if [ -n "$GIT_TOKEN" ] && [[ "$ORIGIN_URL" == https://github.com/* ]]; then
        local token_url="https://x-access-token:${GIT_TOKEN}@${ORIGIN_URL#https://}"
        git remote set-url origin "$token_url" >/dev/null 2>&1 || true
        TOKEN_ACTIVE=true
    fi
}

trap restore_origin_url EXIT

# 确定使用的 compose 文件
if [ -f "docker-compose.https.yml" ] && [ -f "certs/ssl/server.crt" ]; then
    COMPOSE_FILE="docker-compose.https.yml"
else
    COMPOSE_FILE="docker-compose.yml"
fi

# 解析参数
VERSION=""
FORCE=false
for arg in "$@"; do
    case $arg in
        --force|-f)
            FORCE=true
            ;;
        v*)
            VERSION="$arg"
            ;;
    esac
done

# 显示当前版本
show_current_version() {
    if [ -f "VERSION" ]; then
        echo "当前版本: $(cat VERSION)"
    fi
    echo "当前分支: $(git branch --show-current 2>/dev/null || echo 'N/A')"
    echo "最后提交: $(git log -1 --format='%h %s' 2>/dev/null || echo 'N/A')"
}

# 备份当前版本
backup_current() {
    log_info "备份当前配置..."

    BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"

    # 备份配置文件
    [ -f ".env" ] && cp .env "$BACKUP_DIR/"
    [ -f "config.docker.yaml" ] && cp config.docker.yaml "$BACKUP_DIR/"

    log_success "备份完成: $BACKUP_DIR"
}

# 拉取最新代码
pull_latest() {
    log_info "检查更新..."

    prepare_git_auth

    # 检查是否有本地修改
    if ! git diff --quiet 2>/dev/null; then
        if [ "$FORCE" = true ]; then
            log_warning "强制模式：丢弃本地修改"
            git checkout -- .
        else
            log_error "检测到本地修改，请先提交或使用 --force 强制更新"
            git status --short
            exit 1
        fi
    fi

    # 获取远程更新
    git fetch origin

    # 切换到指定版本或拉取最新
    if [ -n "$VERSION" ]; then
        log_info "切换到版本: $VERSION"
        git checkout "$VERSION"
    else
        CURRENT_BRANCH=$(git branch --show-current)
        log_info "更新分支: $CURRENT_BRANCH"
        git pull origin "$CURRENT_BRANCH"
    fi

    log_success "代码更新完成"
}

# 重新构建镜像
rebuild_images() {
    log_info "重新构建 Docker 镜像..."

    docker compose -f "$COMPOSE_FILE" build --no-cache

    log_success "镜像构建完成"
}

# 平滑重启服务
restart_services() {
    log_info "重启服务（零停机）..."

    # 使用 rolling update 策略
    # 先启动新容器，等待健康检查通过后再停止旧容器

    # 重启后端
    docker compose -f "$COMPOSE_FILE" up -d --no-deps --build backend

    # 等待后端健康
    log_info "等待后端服务就绪..."
    sleep 10

    # 检查后端健康状态
    for i in {1..30}; do
        if docker compose -f "$COMPOSE_FILE" exec -T backend wget --no-verbose --tries=1 --spider http://localhost:8080/api/health 2>/dev/null; then
            log_success "后端服务就绪"
            break
        fi
        sleep 2
    done

    # 重启前端
    docker compose -f "$COMPOSE_FILE" up -d --no-deps --build frontend

    log_info "等待前端服务就绪..."
    sleep 5

    log_success "服务重启完成"
}

# 检查服务状态
check_status() {
    log_info "检查服务状态..."

    docker compose -f "$COMPOSE_FILE" ps

    echo ""
    log_info "健康检查..."

    # 检查后端
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/health | grep -q "200"; then
        log_success "后端服务: 正常"
    else
        log_error "后端服务: 异常"
    fi

    # 检查前端
    if curl -s -o /dev/null -w "%{http_code}" http://localhost/ | grep -q "200\|301\|302"; then
        log_success "前端服务: 正常"
    else
        log_error "前端服务: 异常"
    fi
}

# 回滚到上一版本
rollback() {
    log_warning "回滚到上一版本..."

    git checkout HEAD~1
    rebuild_images
    restart_services

    log_success "回滚完成"
}

# 清理旧镜像
cleanup() {
    log_info "清理旧镜像..."

    docker image prune -f
    docker builder prune -f

    log_success "清理完成"
}

# 主函数
main() {
    echo ""
    echo "=========================================="
    echo "      License Server 自动更新"
    echo "=========================================="
    echo ""

    show_current_version
    echo ""

    # 备份
    backup_current

    # 拉取代码
    pull_latest

    # 重新构建
    rebuild_images

    # 重启服务
    restart_services

    # 检查状态
    check_status

    # 清理
    cleanup

    echo ""
    log_success "更新完成！"
    echo ""
    show_current_version
}

# 处理命令
case "${1:-update}" in
    update|"")
        main
        ;;
    rollback)
        rollback
        ;;
    status)
        check_status
        ;;
    cleanup)
        cleanup
        ;;
    *)
        main
        ;;
esac
