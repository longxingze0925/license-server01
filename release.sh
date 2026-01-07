#!/bin/bash
# ============================================
# License Server 版本发布脚本
# ============================================
# 功能：
#   - 创建版本标签
#   - 更新版本号
#   - 推送到 GitHub 触发 CI/CD
# ============================================
# 使用方法：
#   ./release.sh 1.0.0           # 发布 v1.0.0
#   ./release.sh 1.1.0 --push    # 发布并推送
#   ./release.sh patch           # 自动递增补丁版本
#   ./release.sh minor           # 自动递增次版本
#   ./release.sh major           # 自动递增主版本
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

# 版本文件
VERSION_FILE="VERSION"

# 获取当前版本
get_current_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "0.0.0"
    fi
}

# 解析版本号
parse_version() {
    local version=$1
    # 移除 v 前缀
    version=${version#v}

    IFS='.' read -r major minor patch <<< "$version"
    echo "$major $minor $patch"
}

# 递增版本号
increment_version() {
    local current=$1
    local type=$2

    read -r major minor patch <<< "$(parse_version "$current")"

    case $type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
    esac

    echo "$major.$minor.$patch"
}

# 验证版本号格式
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_error "无效的版本号格式: $version"
        log_info "正确格式: X.Y.Z (例如: 1.0.0)"
        exit 1
    fi
}

# 检查 Git 状态
check_git_status() {
    # 检查是否在 Git 仓库中
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "当前目录不是 Git 仓库"
        exit 1
    fi

    # 检查是否有未提交的修改
    if ! git diff --quiet || ! git diff --cached --quiet; then
        log_error "有未提交的修改，请先提交"
        git status --short
        exit 1
    fi

    # 检查是否有未推送的提交
    LOCAL=$(git rev-parse HEAD)
    REMOTE=$(git rev-parse @{u} 2>/dev/null || echo "")
    if [ -n "$REMOTE" ] && [ "$LOCAL" != "$REMOTE" ]; then
        log_warning "有未推送的提交"
    fi
}

# 更新版本文件
update_version_file() {
    local version=$1
    echo "$version" > "$VERSION_FILE"
    log_info "更新版本文件: $VERSION_FILE -> $version"
}

# 生成更新日志
generate_changelog() {
    local version=$1
    local prev_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

    echo ""
    echo "## v$version ($(date +%Y-%m-%d))"
    echo ""

    if [ -n "$prev_tag" ]; then
        echo "### 更新内容"
        echo ""
        git log "$prev_tag"..HEAD --pretty=format:"- %s" --no-merges
        echo ""
    else
        echo "### 初始版本"
        echo ""
        git log --pretty=format:"- %s" --no-merges | head -20
        echo ""
    fi
}

# 创建 Git 标签
create_tag() {
    local version=$1
    local tag="v$version"

    # 检查标签是否已存在
    if git tag -l | grep -q "^$tag$"; then
        log_error "标签 $tag 已存在"
        exit 1
    fi

    # 更新版本文件
    update_version_file "$version"

    # 提交版本更新
    git add "$VERSION_FILE"
    git commit -m "chore: release v$version"

    # 生成更新日志
    CHANGELOG=$(generate_changelog "$version")

    # 创建带注释的标签
    git tag -a "$tag" -m "Release $tag

$CHANGELOG"

    log_success "创建标签: $tag"
}

# 推送到远程
push_to_remote() {
    local tag=$1

    log_info "推送到远程仓库..."

    # 推送代码
    git push origin HEAD

    # 推送标签
    git push origin "$tag"

    log_success "推送完成"
    log_info "GitHub Actions 将自动构建和发布"
}

# 显示帮助
show_help() {
    echo "版本发布脚本"
    echo ""
    echo "用法: $0 <版本号|类型> [选项]"
    echo ""
    echo "版本号:"
    echo "  1.0.0        指定版本号"
    echo "  patch        自动递增补丁版本 (0.0.X)"
    echo "  minor        自动递增次版本 (0.X.0)"
    echo "  major        自动递增主版本 (X.0.0)"
    echo ""
    echo "选项:"
    echo "  --push, -p   创建后自动推送"
    echo "  --dry-run    仅显示将要执行的操作"
    echo "  --help, -h   显示帮助"
    echo ""
    echo "示例:"
    echo "  $0 1.0.0              # 发布 v1.0.0"
    echo "  $0 patch --push       # 递增补丁版本并推送"
    echo "  $0 minor              # 递增次版本"
    echo ""
}

# 主函数
main() {
    local version_arg=""
    local push=false
    local dry_run=false

    # 解析参数
    for arg in "$@"; do
        case $arg in
            --push|-p)
                push=true
                ;;
            --dry-run)
                dry_run=true
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                version_arg="$arg"
                ;;
        esac
    done

    # 检查版本参数
    if [ -z "$version_arg" ]; then
        log_error "请指定版本号"
        show_help
        exit 1
    fi

    # 获取当前版本
    current_version=$(get_current_version)
    log_info "当前版本: v$current_version"

    # 确定新版本
    case $version_arg in
        major|minor|patch)
            new_version=$(increment_version "$current_version" "$version_arg")
            ;;
        *)
            new_version=${version_arg#v}
            validate_version "$new_version"
            ;;
    esac

    log_info "新版本: v$new_version"

    # 干运行模式
    if [ "$dry_run" = true ]; then
        echo ""
        log_warning "干运行模式 - 不会执行任何操作"
        echo ""
        echo "将要执行的操作:"
        echo "  1. 更新 $VERSION_FILE -> $new_version"
        echo "  2. 提交: chore: release v$new_version"
        echo "  3. 创建标签: v$new_version"
        if [ "$push" = true ]; then
            echo "  4. 推送到远程仓库"
        fi
        echo ""
        generate_changelog "$new_version"
        exit 0
    fi

    # 检查 Git 状态
    check_git_status

    # 确认发布
    echo ""
    read -p "确认发布 v$new_version? [y/N] " confirm
    if [[ ! $confirm =~ ^[Yy]$ ]]; then
        log_warning "取消发布"
        exit 0
    fi

    # 创建标签
    create_tag "$new_version"

    # 推送
    if [ "$push" = true ]; then
        push_to_remote "v$new_version"
    else
        echo ""
        log_info "使用以下命令推送:"
        echo "  git push origin HEAD"
        echo "  git push origin v$new_version"
    fi

    echo ""
    log_success "发布完成: v$new_version"
}

main "$@"
