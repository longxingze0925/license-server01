#!/bin/bash
# ============================================
# License Server 一键安装脚本（Bootstrap）
# ============================================
# 作用：
#   - 拉取仓库并进入目录
#   - 调用核心安装脚本 scripts/install-core.sh
# ============================================

set -e

REPO_URL_DEFAULT="https://github.com/longxingze0925/license-server.git"
REPO_BRANCH_DEFAULT="main"
INSTALL_DIR_DEFAULT="/opt/license-server"

REPO_URL="$REPO_URL_DEFAULT"
REPO_BRANCH="$REPO_BRANCH_DEFAULT"
INSTALL_DIR="$INSTALL_DIR_DEFAULT"

GIT_TOKEN="${GIT_TOKEN:-}"
USE_SSH=false
NON_INTERACTIVE=false
SHOW_HELP=false

PASS_ARGS=()

log_info() { echo -e "[INFO] $1"; }
log_error() { echo -e "[ERROR] $1"; }

usage() {
    cat <<'EOF'
用法:
  ./install.sh [选项]

Bootstrap 选项:
  --repo <url>        Git 仓库地址
  --branch <name>     分支或标签
  --dir <path>        安装目录（默认: /opt/license-server）
  --git-token <token> 私有仓库 Token（HTTPS）
  --ssh               使用 SSH 克隆
  -y, --non-interactive  非交互模式
  -h, --help          显示帮助

说明:
- 其他安装参数会透传给 scripts/install-core.sh
- 私有仓库建议使用 SSH 或 Token
EOF
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --repo)
                REPO_URL="$2"; shift 2 ;;
            --branch)
                REPO_BRANCH="$2"; shift 2 ;;
            --dir)
                INSTALL_DIR="$2"; shift 2 ;;
            --git-token)
                GIT_TOKEN="$2"; shift 2 ;;
            --ssh)
                USE_SSH=true; shift ;;
            -y|--non-interactive)
                NON_INTERACTIVE=true; PASS_ARGS+=("$1"); shift ;;
            -h|--help)
                SHOW_HELP=true; PASS_ARGS+=("$1"); shift ;;
            *)
                PASS_ARGS+=("$1"); shift ;;
        esac
    done
}

normalize_repo_url() {
    if [[ "$REPO_URL" != *.git ]]; then
        REPO_URL="${REPO_URL}.git"
    fi
}

https_to_ssh() {
    local url="$1"
    local host_path=${url#https://}
    echo "git@${host_path/\//:}"
}

clone_repo() {
    local target_dir="$1"

    normalize_repo_url

    local clone_url="$REPO_URL"
    local clean_url="$REPO_URL"

    if [ "$USE_SSH" = true ]; then
        clone_url=$(https_to_ssh "$REPO_URL")
        clean_url="$clone_url"
    elif [ -n "$GIT_TOKEN" ]; then
        if [[ "$REPO_URL" == https://github.com/* ]]; then
            clone_url="https://x-access-token:${GIT_TOKEN}@${REPO_URL#https://}"
        fi
    fi

    mkdir -p "$target_dir"
    if [ -n "$(ls -A "$target_dir" 2>/dev/null)" ]; then
        log_error "安装目录非空: $target_dir"
        exit 1
    fi

    log_info "克隆仓库: $REPO_URL (branch: $REPO_BRANCH)"
    git clone -b "$REPO_BRANCH" "$clone_url" "$target_dir"

    if [ -n "$GIT_TOKEN" ] && [ "$USE_SSH" = false ]; then
        (cd "$target_dir" && git remote set-url origin "$clean_url")
    fi
}

ensure_repo_dir() {
    if [ -f "scripts/install-core.sh" ] && [ -f "docker-compose.yml" ]; then
        return 0
    fi

    if [ -d "$INSTALL_DIR/.git" ]; then
        cd "$INSTALL_DIR"
        return 0
    fi

    if [ "$NON_INTERACTIVE" = true ] && [ "$USE_SSH" = false ] && [ -z "$GIT_TOKEN" ]; then
        log_error "私有仓库需要 --git-token 或 --ssh"
        exit 1
    fi

    if [ "$NON_INTERACTIVE" = false ] && [ "$USE_SSH" = false ] && [ -z "$GIT_TOKEN" ]; then
        echo ""
        echo "仓库为私有仓库，请选择认证方式:"
        echo "  1) HTTPS Token"
        echo "  2) SSH（需已配置 SSH Key）"
        read -p "请选择 [1]: " auth_choice
        auth_choice=${auth_choice:-1}
        if [ "$auth_choice" = "2" ]; then
            USE_SSH=true
        else
            read -p "请输入 GitHub Token: " GIT_TOKEN
            if [ -z "$GIT_TOKEN" ]; then
                log_error "Token 不能为空"
                exit 1
            fi
        fi
    fi

    clone_repo "$INSTALL_DIR"
    cd "$INSTALL_DIR"
}

main() {
    parse_args "$@"

    if [ "$SHOW_HELP" = true ]; then
        if [ -f "scripts/install-core.sh" ]; then
            exec bash scripts/install-core.sh --help
        fi
        usage
        exit 0
    fi

    ensure_repo_dir

    if [ -n "$GIT_TOKEN" ]; then
        export GIT_TOKEN
    fi

    exec bash scripts/install-core.sh "${PASS_ARGS[@]}"
}

main "$@"
