#!/bin/bash
# ============================================
# License Server 一键安装脚本 (HTTPS 版本)
# ============================================
# 功能：
#   - 检查系统环境
#   - 自动生成安全密钥
#   - 支持 HTTPS（自签名/Let's Encrypt）
#   - 配置 Docker 环境
#   - 启动所有服务
#   - 初始化管理员账号
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 横幅
print_banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                                                          ║"
    echo "║       License Server 一键安装脚本 (HTTPS)                ║"
    echo "║                                                          ║"
    echo "║           多应用授权管理平台                             ║"
    echo "║                                                          ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# 检查 root 权限
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 用户运行此脚本"
        log_info "使用: sudo ./install-https.sh"
        exit 1
    fi
}

# 检查系统要求
check_requirements() {
    log_info "检查系统要求..."

    # 检查操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        log_info "操作系统: $OS"
    fi

    # 检查内存
    TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
    if [ "$TOTAL_MEM" -lt 1024 ]; then
        log_warning "内存小于 1GB，可能影响性能"
    else
        log_success "内存: ${TOTAL_MEM}MB"
    fi

    # 检查磁盘空间
    FREE_DISK=$(df -m / | awk 'NR==2 {print $4}')
    if [ "$FREE_DISK" -lt 5120 ]; then
        log_warning "磁盘空间小于 5GB"
    else
        log_success "可用磁盘: ${FREE_DISK}MB"
    fi
}

# 安装 Docker
install_docker() {
    if command -v docker &> /dev/null; then
        log_success "Docker 已安装: $(docker --version)"
    else
        log_info "正在安装 Docker..."
        curl -fsSL https://get.docker.com | sh
        systemctl enable docker
        systemctl start docker
        log_success "Docker 安装完成"
    fi

    if docker compose version &> /dev/null; then
        log_success "Docker Compose 已安装"
    else
        log_info "正在安装 Docker Compose 插件..."
        apt-get update
        apt-get install -y docker-compose-plugin
        log_success "Docker Compose 安装完成"
    fi
}

# 安装 openssl
install_openssl() {
    if command -v openssl &> /dev/null; then
        log_success "OpenSSL 已安装"
    else
        log_info "正在安装 OpenSSL..."
        apt-get update && apt-get install -y openssl
        log_success "OpenSSL 安装完成"
    fi
}

# 生成随机密码
generate_password() {
    local length=${1:-16}
    openssl rand -base64 48 | tr -dc 'a-zA-Z0-9!@#$%^&*()_+' | head -c "$length"
}

# 生成随机密钥
generate_secret() {
    openssl rand -base64 32
}

# 获取服务器 IP
get_server_ip() {
    PUBLIC_IP=$(curl -s --max-time 5 https://api.ipify.org 2>/dev/null || \
                curl -s --max-time 5 https://ifconfig.me 2>/dev/null || \
                curl -s --max-time 5 https://icanhazip.com 2>/dev/null || \
                echo "")

    if [ -z "$PUBLIC_IP" ]; then
        PUBLIC_IP=$(hostname -I | awk '{print $1}')
    fi

    echo "$PUBLIC_IP"
}

# 交互式配置
interactive_config() {
    log_info "开始配置..."
    echo ""

    # 获取服务器 IP
    DEFAULT_IP=$(get_server_ip)
    read -p "服务器 IP 地址 [$DEFAULT_IP]: " SERVER_IP
    SERVER_IP=${SERVER_IP:-$DEFAULT_IP}

    # SSL 证书选择
    echo ""
    echo "=========================================="
    echo "         选择 SSL 证书类型"
    echo "=========================================="
    echo ""
    echo "  1) 自签名证书（推荐用于 IP 地址部署）"
    echo "     - 无需域名"
    echo "     - 浏览器会显示安全警告"
    echo ""
    echo "  2) Let's Encrypt 证书（推荐用于域名部署）"
    echo "     - 需要有效域名指向此服务器"
    echo "     - 免费，自动续期"
    echo "     - 浏览器信任，无警告"
    echo ""
    echo "  3) 仅 HTTP（不推荐）"
    echo "     - 不启用 HTTPS"
    echo ""

    read -p "请选择 [1]: " SSL_CHOICE
    SSL_CHOICE=${SSL_CHOICE:-1}

    case $SSL_CHOICE in
        1)
            SSL_MODE="self-signed"
            log_info "将使用自签名证书"
            ;;
        2)
            SSL_MODE="letsencrypt"
            read -p "请输入域名: " DOMAIN
            read -p "请输入邮箱（用于证书到期提醒）: " SSL_EMAIL
            if [ -z "$DOMAIN" ]; then
                log_error "域名不能为空"
                exit 1
            fi
            log_info "将使用 Let's Encrypt 证书"
            ;;
        3)
            SSL_MODE="http"
            log_warning "将不启用 HTTPS（不推荐用于生产环境）"
            ;;
        *)
            SSL_MODE="self-signed"
            log_info "默认使用自签名证书"
            ;;
    esac

    # 端口配置
    echo ""
    if [ "$SSL_MODE" = "http" ]; then
        read -p "HTTP 端口 [80]: " HTTP_PORT
        HTTP_PORT=${HTTP_PORT:-80}
        HTTPS_PORT=""
    else
        read -p "HTTP 端口（用于重定向）[80]: " HTTP_PORT
        HTTP_PORT=${HTTP_PORT:-80}
        read -p "HTTPS 端口 [443]: " HTTPS_PORT
        HTTPS_PORT=${HTTPS_PORT:-443}
    fi

    read -p "后端端口 [8080]: " BACKEND_PORT
    BACKEND_PORT=${BACKEND_PORT:-8080}

    # 管理员配置
    echo ""
    read -p "管理员邮箱 [admin@example.com]: " ADMIN_EMAIL
    ADMIN_EMAIL=${ADMIN_EMAIL:-admin@example.com}

    # 自动生成密码
    log_info "正在生成安全密钥..."

    MYSQL_ROOT_PASSWORD=$(generate_password 20)
    MYSQL_PASSWORD=$(generate_password 16)
    REDIS_PASSWORD=$(generate_password 16)
    JWT_SECRET=$(generate_secret)
    ADMIN_PASSWORD=$(generate_password 12)

    log_success "安全密钥生成完成"
}

# 生成自签名证书
generate_self_signed_cert() {
    log_info "正在生成自签名 SSL 证书..."

    mkdir -p certs/ssl

    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout certs/ssl/server.key \
        -out certs/ssl/server.crt \
        -subj "/CN=${SERVER_IP}" \
        -addext "subjectAltName=DNS:${SERVER_IP},DNS:localhost,IP:${SERVER_IP},IP:127.0.0.1"

    chmod 600 certs/ssl/server.key
    chmod 644 certs/ssl/server.crt

    log_success "自签名证书生成完成"
}

# 申请 Let's Encrypt 证书
generate_letsencrypt_cert() {
    log_info "正在申请 Let's Encrypt 证书..."

    # 安装 certbot
    if ! command -v certbot &> /dev/null; then
        log_info "安装 certbot..."
        apt-get update
        apt-get install -y certbot
    fi

    # 创建目录
    mkdir -p certs/ssl certs/letsencrypt certs/certbot

    # 申请证书
    certbot certonly --standalone \
        -d "$DOMAIN" \
        --email "$SSL_EMAIL" \
        --agree-tos \
        --no-eff-email \
        --non-interactive

    if [ $? -eq 0 ]; then
        # 复制证书
        cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" certs/ssl/server.crt
        cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" certs/ssl/server.key
        chmod 600 certs/ssl/server.key
        chmod 644 certs/ssl/server.crt

        log_success "Let's Encrypt 证书申请成功"

        # 设置自动续期
        setup_auto_renew
    else
        log_error "Let's Encrypt 证书申请失败"
        log_info "回退到自签名证书..."
        SSL_MODE="self-signed"
        generate_self_signed_cert
    fi
}

# 设置自动续期
setup_auto_renew() {
    log_info "配置证书自动续期..."

    cat > /etc/cron.d/certbot-renew << EOF
# 每天凌晨 2 点检查并续期证书
0 2 * * * root certbot renew --quiet && cp /etc/letsencrypt/live/$DOMAIN/fullchain.pem $(pwd)/certs/ssl/server.crt && cp /etc/letsencrypt/live/$DOMAIN/privkey.pem $(pwd)/certs/ssl/server.key && docker compose -f $(pwd)/docker-compose.https.yml restart frontend
EOF

    log_success "自动续期已配置"
}

# 创建 .env 文件
create_env_file() {
    log_info "创建环境配置文件..."

    cat > .env << EOF
# ============================================
# License Server 环境配置 (HTTPS)
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ============================================

# 服务器配置
SERVER_IP=${SERVER_IP}
DOMAIN=${DOMAIN:-}
SSL_MODE=${SSL_MODE}
BACKEND_PORT=${BACKEND_PORT}
HTTP_PORT=${HTTP_PORT}
HTTPS_PORT=${HTTPS_PORT}

# MySQL 配置
MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
MYSQL_DATABASE=license_server
MYSQL_USER=license_admin
MYSQL_PASSWORD=${MYSQL_PASSWORD}
MYSQL_PORT=3306

# Redis 配置
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_PORT=6379

# JWT 配置
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRE_HOURS=24

# 安全配置
SERVER_MODE=release
TLS_ENABLED=true

# 管理员配置
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}

# 前端配置
VITE_API_URL=/api
EOF

    chmod 600 .env
    log_success ".env 文件创建完成"
}

# 创建 Docker 配置文件
create_docker_config() {
    log_info "创建 Docker 配置文件..."

    # 确定访问地址
    if [ "$SSL_MODE" = "http" ]; then
        ACCESS_URL="http://${SERVER_IP}:${HTTP_PORT}"
    else
        ACCESS_URL="https://${DOMAIN:-$SERVER_IP}:${HTTPS_PORT}"
    fi

    cat > config.docker.yaml << EOF
# License Server Docker 配置 (HTTPS)
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"
  tls:
    enabled: false
    cert_file: "/app/certs/server.crt"
    key_file: "/app/certs/server.key"

database:
  driver: "mysql"
  host: "mysql"
  port: 3306
  username: "license_admin"
  password: "${MYSQL_PASSWORD}"
  database: "license_server"
  charset: "utf8mb4"
  max_idle_conns: 10
  max_open_conns: 100

redis:
  host: "redis"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0

jwt:
  secret: "${JWT_SECRET}"
  expire_hours: 24

rsa:
  key_size: 2048

storage:
  scripts_dir: "/app/storage/scripts"
  releases_dir: "/app/storage/releases"

log:
  level: "info"
  file: "/app/logs/app.log"
  max_size: 100
  max_backups: 5
  max_age: 30

email:
  enabled: false
  smtp_host: ""
  smtp_port: 587
  username: ""
  password: ""
  from: ""

security:
  max_login_attempts: 5
  login_lock_minutes: 15
  ip_max_attempts: 20
  ip_lock_minutes: 30
  password_min_length: 8
  password_require_num: true
  password_require_sym: true
  csrf_enabled: false
  csrf_token_expiry: 60
  csrf_cookie_name: "csrf_token"
  enable_security_headers: true
  allowed_origins:
    - "${ACCESS_URL}"
    - "https://${DOMAIN:-$SERVER_IP}"
    - "http://${SERVER_IP}"
    - "http://localhost:3000"
    - "http://127.0.0.1:3000"
EOF

    log_success "Docker 配置文件创建完成"
}

# 创建必要目录
create_directories() {
    log_info "创建必要目录..."

    mkdir -p storage/scripts
    mkdir -p storage/releases
    mkdir -p logs
    mkdir -p certs/ssl
    mkdir -p certs/letsencrypt
    mkdir -p certs/certbot

    chmod -R 755 storage logs certs

    log_success "目录创建完成"
}

# 生成 SSL 证书
generate_ssl_cert() {
    case $SSL_MODE in
        self-signed)
            generate_self_signed_cert
            ;;
        letsencrypt)
            generate_letsencrypt_cert
            ;;
        http)
            log_info "跳过 SSL 证书生成（HTTP 模式）"
            ;;
    esac
}

# 构建并启动服务
start_services() {
    log_info "构建 Docker 镜像（首次可能需要几分钟）..."

    if [ "$SSL_MODE" = "http" ]; then
        COMPOSE_FILE="docker-compose.yml"
    else
        COMPOSE_FILE="docker-compose.https.yml"
    fi

    docker compose -f "$COMPOSE_FILE" build --no-cache

    log_info "启动服务..."
    docker compose -f "$COMPOSE_FILE" up -d

    log_info "等待服务启动..."
    sleep 15

    if docker compose -f "$COMPOSE_FILE" ps | grep -q "Up"; then
        log_success "服务启动成功"
    else
        log_error "服务启动失败，请检查日志: docker compose -f $COMPOSE_FILE logs"
        exit 1
    fi
}

# 初始化管理员账号
init_admin() {
    log_info "初始化管理员账号..."

    # 等待数据库完全就绪
    log_info "等待数据库就绪..."
    sleep 10

    # 确定 compose 文件
    if [ "$SSL_MODE" = "http" ]; then
        COMPOSE_FILE="docker-compose.yml"
    else
        COMPOSE_FILE="docker-compose.https.yml"
    fi

    # 使用 Python 生成 bcrypt 密码哈希
    log_info "生成密码哈希..."
    PASSWORD_HASH=$(docker run --rm python:3-alpine sh -c "pip install -q bcrypt && python -c \"import bcrypt; print(bcrypt.hashpw(b'${ADMIN_PASSWORD}', bcrypt.gensalt(10)).decode())\"" 2>/dev/null)

    if [ -z "$PASSWORD_HASH" ]; then
        log_error "无法生成密码哈希"
        return 1
    fi

    log_info "创建租户和管理员账号..."

    # 通过 MySQL 容器创建租户和管理员（使用 utf8mb4 字符集）
    docker exec license-mysql mysql -u root -p"${MYSQL_ROOT_PASSWORD}" --default-character-set=utf8mb4 license_server -e "
    -- 检查是否已存在租户
    SET @tenant_exists = (SELECT COUNT(*) FROM tenants WHERE slug = 'default');

    -- 如果不存在则创建租户
    SET @tenant_id = UUID();
    INSERT INTO tenants (id, name, slug, plan, status, created_at, updated_at)
    SELECT @tenant_id, '默认团队', 'default', 'professional', 'active', NOW(), NOW()
    WHERE @tenant_exists = 0;

    -- 获取租户 ID（无论是新建还是已存在）
    SET @final_tenant_id = (SELECT id FROM tenants WHERE slug = 'default' LIMIT 1);

    -- 检查管理员是否已存在
    SET @admin_exists = (SELECT COUNT(*) FROM team_members WHERE email = '${ADMIN_EMAIL}');

    -- 如果不存在则创建管理员
    INSERT INTO team_members (id, tenant_id, email, password, name, role, status, created_at, updated_at, email_verified)
    SELECT UUID(), @final_tenant_id, '${ADMIN_EMAIL}', '${PASSWORD_HASH}', '管理员', 'owner', 'active', NOW(), NOW(), 1
    WHERE @admin_exists = 0;
    " 2>/dev/null

    if [ $? -eq 0 ]; then
        log_success "管理员账号初始化完成"
    else
        log_warning "管理员账号可能已存在或创建失败，请检查"
    fi
}

# 配置防火墙
configure_firewall() {
    log_info "配置防火墙..."

    if command -v ufw &> /dev/null; then
        ufw allow ${HTTP_PORT}/tcp
        [ -n "$HTTPS_PORT" ] && ufw allow ${HTTPS_PORT}/tcp
        log_success "UFW 防火墙规则已添加"
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-port=${HTTP_PORT}/tcp
        [ -n "$HTTPS_PORT" ] && firewall-cmd --permanent --add-port=${HTTPS_PORT}/tcp
        firewall-cmd --reload
        log_success "Firewalld 防火墙规则已添加"
    else
        log_warning "未检测到防火墙，请手动配置"
    fi
}

# 保存凭据
save_credentials() {
    CREDENTIALS_FILE="credentials.txt"

    # 确定访问地址
    if [ "$SSL_MODE" = "http" ]; then
        FRONTEND_URL="http://${SERVER_IP}:${HTTP_PORT}"
        BACKEND_URL="http://${SERVER_IP}:${BACKEND_PORT}"
    else
        FRONTEND_URL="https://${DOMAIN:-$SERVER_IP}:${HTTPS_PORT}"
        BACKEND_URL="http://${SERVER_IP}:${BACKEND_PORT}"
    fi

    cat > "$CREDENTIALS_FILE" << EOF
╔══════════════════════════════════════════════════════════════════════════╗
║                    License Server 安装凭据 (HTTPS)                       ║
║                    生成时间: $(date '+%Y-%m-%d %H:%M:%S')                         ║
╚══════════════════════════════════════════════════════════════════════════╝

【重要提示】请妥善保管此文件，首次登录后请立即修改密码！

═══════════════════════════════════════════════════════════════════════════
                              SSL 配置
═══════════════════════════════════════════════════════════════════════════

SSL 模式:     ${SSL_MODE}
$([ "$SSL_MODE" = "letsencrypt" ] && echo "域名:         ${DOMAIN}")
$([ "$SSL_MODE" = "self-signed" ] && echo "注意:         自签名证书，浏览器会显示安全警告")

═══════════════════════════════════════════════════════════════════════════
                              访问地址
═══════════════════════════════════════════════════════════════════════════

前端管理后台: ${FRONTEND_URL}
后端 API 地址: ${BACKEND_URL}

═══════════════════════════════════════════════════════════════════════════
                            管理员账号
═══════════════════════════════════════════════════════════════════════════

邮箱: ${ADMIN_EMAIL}
密码: ${ADMIN_PASSWORD}

═══════════════════════════════════════════════════════════════════════════
                            数据库信息
═══════════════════════════════════════════════════════════════════════════

MySQL Root 密码: ${MYSQL_ROOT_PASSWORD}
MySQL 用户名:    license_admin
MySQL 密码:      ${MYSQL_PASSWORD}
MySQL 数据库:    license_server

Redis 密码:      ${REDIS_PASSWORD}

═══════════════════════════════════════════════════════════════════════════
                              JWT 密钥
═══════════════════════════════════════════════════════════════════════════

${JWT_SECRET}

═══════════════════════════════════════════════════════════════════════════
                            常用命令
═══════════════════════════════════════════════════════════════════════════

# 使用的 compose 文件
COMPOSE_FILE=$([ "$SSL_MODE" = "http" ] && echo "docker-compose.yml" || echo "docker-compose.https.yml")

查看服务状态:    docker compose -f \$COMPOSE_FILE ps
查看日志:        docker compose -f \$COMPOSE_FILE logs -f
重启服务:        docker compose -f \$COMPOSE_FILE restart
停止服务:        docker compose -f \$COMPOSE_FILE down
更新服务:        docker compose -f \$COMPOSE_FILE pull && docker compose -f \$COMPOSE_FILE up -d

# SSL 证书管理
查看证书状态:    ./ssl-manager.sh status
续期证书:        ./ssl-manager.sh renew

═══════════════════════════════════════════════════════════════════════════
EOF

    chmod 600 "$CREDENTIALS_FILE"
    log_success "凭据已保存到 $CREDENTIALS_FILE"
}

# 打印完成信息
print_completion() {
    # 确定访问地址
    if [ "$SSL_MODE" = "http" ]; then
        FRONTEND_URL="http://${SERVER_IP}:${HTTP_PORT}"
    else
        FRONTEND_URL="https://${DOMAIN:-$SERVER_IP}:${HTTPS_PORT}"
    fi

    echo ""
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════════════════╗"
    echo "║                                                                          ║"
    echo "║                    🎉 安装完成！                                         ║"
    echo "║                                                                          ║"
    echo "╚══════════════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    echo -e "  ${BLUE}SSL 模式:${NC}     ${SSL_MODE}"
    echo -e "  ${BLUE}前端管理后台:${NC} ${FRONTEND_URL}"
    echo ""
    echo -e "  ${BLUE}管理员邮箱:${NC}   ${ADMIN_EMAIL}"
    echo -e "  ${BLUE}管理员密码:${NC}   ${ADMIN_PASSWORD}"
    echo ""

    if [ "$SSL_MODE" = "self-signed" ]; then
        echo -e "  ${YELLOW}【注意】使用自签名证书，浏览器会显示安全警告${NC}"
        echo -e "  ${YELLOW}        点击「高级」->「继续访问」即可${NC}"
        echo ""
    fi

    echo -e "  ${YELLOW}【重要】所有凭据已保存到 credentials.txt，请妥善保管！${NC}"
    echo -e "  ${YELLOW}【重要】首次登录后请立即修改默认密码！${NC}"
    echo ""
}

# 主函数
main() {
    print_banner
    check_root
    check_requirements
    install_docker
    install_openssl
    interactive_config
    create_directories
    create_env_file
    create_docker_config
    generate_ssl_cert
    start_services
    init_admin
    configure_firewall
    save_credentials
    print_completion
}

main "$@"
