#!/bin/bash
# ============================================
# License Server 一键安装脚本
# ============================================
# 功能：
#   - 检查系统环境
#   - 自动生成安全密钥
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
NC='\033[0m' # No Color

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 横幅
print_banner() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                                                          ║"
    echo "║           License Server 一键安装脚本                    ║"
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
        log_info "使用: sudo ./install.sh"
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
    else
        log_warning "无法检测操作系统"
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

    # 检查 Docker Compose
    if docker compose version &> /dev/null; then
        log_success "Docker Compose 已安装"
    else
        log_info "正在安装 Docker Compose 插件..."
        apt-get update
        apt-get install -y docker-compose-plugin
        log_success "Docker Compose 安装完成"
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
    # 尝试获取公网 IP
    PUBLIC_IP=$(curl -s --max-time 5 https://api.ipify.org 2>/dev/null || \
                curl -s --max-time 5 https://ifconfig.me 2>/dev/null || \
                curl -s --max-time 5 https://icanhazip.com 2>/dev/null || \
                echo "")

    # 如果获取不到公网 IP，使用内网 IP
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

    # 端口配置
    read -p "前端端口 [80]: " FRONTEND_PORT
    FRONTEND_PORT=${FRONTEND_PORT:-80}

    read -p "后端端口 [8080]: " BACKEND_PORT
    BACKEND_PORT=${BACKEND_PORT:-8080}

    # 管理员配置
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

# 创建 .env 文件
create_env_file() {
    log_info "创建环境配置文件..."

    cat > .env << EOF
# ============================================
# License Server 环境配置
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ============================================

# 服务器配置
SERVER_IP=${SERVER_IP}
BACKEND_PORT=${BACKEND_PORT}
FRONTEND_PORT=${FRONTEND_PORT}

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
TLS_ENABLED=false

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

    cat > config.docker.yaml << EOF
# License Server Docker 配置
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
    - "http://${SERVER_IP}:${FRONTEND_PORT}"
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
    mkdir -p certs

    chmod -R 755 storage logs certs

    log_success "目录创建完成"
}

# 修改前端 API 地址
update_frontend_config() {
    log_info "更新前端配置..."

    # 创建前端环境变量文件
    cat > admin/.env.production << EOF
VITE_API_URL=/api
EOF

    log_success "前端配置更新完成"
}

# 构建并启动服务
start_services() {
    log_info "构建 Docker 镜像（首次可能需要几分钟）..."

    docker compose build --no-cache

    log_info "启动服务..."
    docker compose up -d

    log_info "等待服务启动..."
    sleep 10

    # 检查服务状态
    if docker compose ps | grep -q "Up"; then
        log_success "服务启动成功"
    else
        log_error "服务启动失败，请检查日志: docker compose logs"
        exit 1
    fi
}

# 初始化管理员账号
init_admin() {
    log_info "初始化管理员账号..."

    # 等待数据库完全就绪
    log_info "等待数据库就绪..."
    sleep 10

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
        ufw allow ${FRONTEND_PORT}/tcp
        ufw allow ${BACKEND_PORT}/tcp
        log_success "UFW 防火墙规则已添加"
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-port=${FRONTEND_PORT}/tcp
        firewall-cmd --permanent --add-port=${BACKEND_PORT}/tcp
        firewall-cmd --reload
        log_success "Firewalld 防火墙规则已添加"
    else
        log_warning "未检测到防火墙，请手动配置"
    fi
}

# 保存凭据
save_credentials() {
    CREDENTIALS_FILE="credentials.txt"

    cat > "$CREDENTIALS_FILE" << EOF
╔══════════════════════════════════════════════════════════════════════════╗
║                    License Server 安装凭据                               ║
║                    生成时间: $(date '+%Y-%m-%d %H:%M:%S')                         ║
╚══════════════════════════════════════════════════════════════════════════╝

【重要提示】请妥善保管此文件，首次登录后请立即修改密码！

═══════════════════════════════════════════════════════════════════════════
                              访问地址
═══════════════════════════════════════════════════════════════════════════

前端管理后台: http://${SERVER_IP}:${FRONTEND_PORT}
后端 API 地址: http://${SERVER_IP}:${BACKEND_PORT}

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

查看服务状态:    docker compose ps
查看日志:        docker compose logs -f
重启服务:        docker compose restart
停止服务:        docker compose down
更新服务:        docker compose pull && docker compose up -d

═══════════════════════════════════════════════════════════════════════════
EOF

    chmod 600 "$CREDENTIALS_FILE"
    log_success "凭据已保存到 $CREDENTIALS_FILE"
}

# 打印完成信息
print_completion() {
    echo ""
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════════════════════════════╗"
    echo "║                                                                          ║"
    echo "║                    🎉 安装完成！                                         ║"
    echo "║                                                                          ║"
    echo "╚══════════════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    echo -e "  ${BLUE}前端管理后台:${NC} http://${SERVER_IP}:${FRONTEND_PORT}"
    echo -e "  ${BLUE}后端 API:${NC}     http://${SERVER_IP}:${BACKEND_PORT}"
    echo ""
    echo -e "  ${BLUE}管理员邮箱:${NC}   ${ADMIN_EMAIL}"
    echo -e "  ${BLUE}管理员密码:${NC}   ${ADMIN_PASSWORD}"
    echo ""
    echo -e "  ${YELLOW}【重要】所有凭据已保存到 credentials.txt，请妥善保管！${NC}"
    echo -e "  ${YELLOW}【重要】首次登录后请立即修改默认密码！${NC}"
    echo ""
}

# 主函数
main() {
    print_banner

    # 检查 root 权限
    check_root

    # 检查系统要求
    check_requirements

    # 安装 Docker
    install_docker

    # 交互式配置
    interactive_config

    # 创建配置文件
    create_env_file
    create_docker_config

    # 创建目录
    create_directories

    # 更新前端配置
    update_frontend_config

    # 构建并启动服务
    start_services

    # 初始化管理员
    init_admin

    # 配置防火墙
    configure_firewall

    # 保存凭据
    save_credentials

    # 打印完成信息
    print_completion
}

# 运行主函数
main "$@"
