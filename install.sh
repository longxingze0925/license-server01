#!/bin/bash
# ============================================
# License Server 一键安装脚本（Bootstrap）
# ============================================
# 作用：
#   - 拉取仓库并进入目录
#   - 调用核心安装脚本 scripts/install-core.sh
# ============================================

set -e

REPO_URL_DEFAULT="https://github.com/longxingze0925/license-server01.git"
REPO_BRANCH_DEFAULT="main"
INSTALL_DIR_DEFAULT="/opt/license-server"

REPO_URL="${LS_REPO:-$REPO_URL_DEFAULT}"
REPO_BRANCH="${LS_BRANCH:-$REPO_BRANCH_DEFAULT}"
INSTALL_DIR="${LS_DIR:-$INSTALL_DIR_DEFAULT}"

GIT_TOKEN="${LS_GIT_TOKEN:-${GIT_TOKEN:-}}"
USE_SSH=false
NON_INTERACTIVE=false
SHOW_HELP=false

PASS_ARGS=()

log_info() { echo -e "[INFO] $1"; }
log_error() { echo -e "[ERROR] $1"; }

ensure_root() {
    if [ "$EUID" -eq 0 ]; then
        return 0
    fi
    if command -v sudo >/dev/null 2>&1; then
        log_info "检测到非 root 用户，尝试使用 sudo 重新执行..."
        exec sudo -E bash "$0" "$@"
    fi
    log_error "请使用 root 用户运行此脚本（或安装 sudo）"
    exit 1
}

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

is_true() {
    case "$1" in
        1|true|TRUE|yes|YES|y|Y|on|ON) return 0 ;;
        *) return 1 ;;
    esac
}

has_arg() {
    local key="$1"
    for arg in "${PASS_ARGS[@]}"; do
        if [ "$arg" = "$key" ] || [[ "$arg" == "$key="* ]]; then
            return 0
        fi
    done
    return 1
}

append_arg_if_set() {
    local key="$1"
    local value="$2"
    if [ -n "$value" ] && ! has_arg "$key"; then
        PASS_ARGS+=("$key" "$value")
    fi
}

apply_env_overrides() {
    local ssl_mode="${LS_SSL:-}"
    local domain="${LS_DOMAIN:-}"
    local email="${LS_EMAIL:-}"
    local server_ip="${LS_SERVER_IP:-}"
    local http_port="${LS_HTTP_PORT:-}"
    local https_port="${LS_HTTPS_PORT:-}"
    local backend_port="${LS_BACKEND_PORT:-}"
    local admin_email="${LS_ADMIN_EMAIL:-}"
    local admin_password="${LS_ADMIN_PASSWORD:-}"
    local cert_path="${LS_CERT:-}"
    local key_path="${LS_KEY:-}"
    local image_tag="${LS_IMAGE_TAG:-}"

    append_arg_if_set "--ssl" "$ssl_mode"
    append_arg_if_set "--domain" "$domain"
    append_arg_if_set "--email" "$email"
    append_arg_if_set "--server-ip" "$server_ip"
    append_arg_if_set "--http-port" "$http_port"
    append_arg_if_set "--https-port" "$https_port"
    append_arg_if_set "--backend-port" "$backend_port"
    append_arg_if_set "--admin-email" "$admin_email"
    append_arg_if_set "--admin-password" "$admin_password"
    append_arg_if_set "--cert" "$cert_path"
    append_arg_if_set "--key" "$key_path"
    append_arg_if_set "--image-tag" "$image_tag"

    if is_true "${LS_NGINX_PROXY:-}"; then
        if ! has_arg "--nginx-proxy"; then
            PASS_ARGS+=("--nginx-proxy")
        fi
    fi

    if is_true "${LS_BUILD:-}"; then
        if ! has_arg "--build" && ! has_arg "--no-build"; then
            PASS_ARGS+=("--build")
        fi
    fi

    if is_true "${LS_NON_INTERACTIVE:-}" || is_true "${LS_YES:-}"; then
        if ! has_arg "--non-interactive" && ! has_arg "-y" && ! has_arg "--yes"; then
            PASS_ARGS+=("--non-interactive")
            NON_INTERACTIVE=true
        fi
    fi

    if is_true "${LS_SSH:-}"; then
        USE_SSH=true
    fi
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
    ensure_root "$@"
    parse_args "$@"
    apply_env_overrides

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
