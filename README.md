# License Server - 授权管理平台

多应用授权管理平台，支持用户管理、组织管理、授权码管理、设备绑定、脚本下发等功能。

## 功能特性

- **用户管理**: 注册、登录、JWT 认证
- **组织管理**: 多组织支持，成员角色管理
- **应用管理**: 多应用支持，独立 RSA 密钥对
- **授权管理**: 授权码生成、激活、续费、吊销、暂停
- **设备管理**: 设备绑定、数量限制、黑名单
- **心跳验证**: 定期验证，防止破解
- **脚本下发**: 加密脚本，按需下发
- **版本更新**: 软件版本管理，强制更新

## 技术栈

- Go 1.21+
- Gin (Web 框架)
- GORM (ORM)
- MySQL (数据库)
- JWT (认证)
- RSA (签名加密)

## 服务器一键安装（Docker，推荐）

> 私有仓库需要 GitHub Token 或已配置 SSH Key。以下为 Token 方式示例。

### HTTPS（Let's Encrypt，域名）

```bash
export GIT_TOKEN=YOUR_TOKEN
curl -H "Authorization: token $GIT_TOKEN" -fsSL \
  https://raw.githubusercontent.com/longxingze0925/license-server01/main/install.sh | \
  bash -s -- --repo https://github.com/longxingze0925/license-server01.git \
  --branch main --git-token "$GIT_TOKEN" \
  --ssl letsencrypt --domain example.com --email admin@example.com -y
```

### HTTPS（自定义证书）

```bash
export GIT_TOKEN=YOUR_TOKEN
curl -H "Authorization: token $GIT_TOKEN" -fsSL \
  https://raw.githubusercontent.com/longxingze0925/license-server01/main/install.sh | \
  bash -s -- --repo https://github.com/longxingze0925/license-server01.git \
  --branch main --git-token "$GIT_TOKEN" \
  --ssl custom --cert /path/to/fullchain.crt --key /path/to/private.key -y
```

### 更新

```bash
cd /opt/license-server
export GIT_TOKEN=YOUR_TOKEN
./update.sh
```

## 快速开始

### 1. 配置数据库

创建 MySQL 数据库：

```sql
CREATE DATABASE license_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 修改配置

编辑 `config.yaml`，配置数据库连接信息：

```yaml
database:
  host: "localhost"
  port: 3306
  username: "root"
  password: "your_password"
  database: "license_server"
```

### 3. 数据库迁移

```bash
go run ./cmd/main.go -migrate
```

### 4. 启动服务

```bash
go run ./cmd/main.go
```

服务将在 `http://localhost:8080` 启动。

## API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/register | 用户注册 |
| POST | /api/auth/login | 用户登录 |

### 客户端接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/client/auth/activate | 激活授权码 |
| POST | /api/client/auth/verify | 验证授权 |
| POST | /api/client/auth/heartbeat | 心跳 |
| POST | /api/client/auth/deactivate | 解绑设备 |
| GET | /api/client/scripts/version | 获取脚本版本 |
| GET | /api/client/scripts/:filename | 下载脚本 |
| GET | /api/client/releases/latest | 获取最新版本 |

### 管理接口（需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/admin/apps | 创建应用 |
| GET | /api/admin/apps | 获取应用列表 |
| GET | /api/admin/apps/:id | 获取应用详情 |
| PUT | /api/admin/apps/:id | 更新应用 |
| DELETE | /api/admin/apps/:id | 删除应用 |
| POST | /api/admin/licenses | 创建授权码 |
| GET | /api/admin/licenses | 获取授权列表 |
| POST | /api/admin/licenses/:id/renew | 续费 |
| POST | /api/admin/licenses/:id/revoke | 吊销 |
| POST | /api/admin/licenses/:id/suspend | 暂停 |
| POST | /api/admin/licenses/:id/resume | 恢复 |

## 客户端集成

### 激活授权码

```json
POST /api/client/auth/activate
{
  "app_key": "应用Key",
  "license_key": "XXXX-XXXX-XXXX-XXXX",
  "machine_id": "设备指纹",
  "device_info": {
    "name": "设备名称",
    "os": "Windows",
    "os_version": "10",
    "app_version": "1.0.0"
  }
}
```

### 验证授权

```json
POST /api/client/auth/verify
{
  "app_key": "应用Key",
  "machine_id": "设备指纹"
}
```

### 心跳

```json
POST /api/client/auth/heartbeat
{
  "app_key": "应用Key",
  "machine_id": "设备指纹",
  "app_version": "1.0.0"
}
```

## 目录结构

```
license-server/
├── cmd/
│   └── main.go              # 入口文件
├── internal/
│   ├── config/              # 配置管理
│   ├── handler/             # HTTP 处理器
│   ├── middleware/          # 中间件
│   ├── model/               # 数据模型
│   └── pkg/                 # 工具包
│       ├── crypto/          # 加密工具
│       ├── response/        # 响应封装
│       └── utils/           # 通用工具
├── storage/
│   ├── scripts/             # 脚本存储
│   └── releases/            # 发布包存储
├── config.yaml              # 配置文件
└── README.md
```

## 安全说明

1. **RSA 签名**: 每个应用独立密钥对，响应数据签名验证
2. **设备绑定**: 硬件指纹绑定，限制设备数量
3. **心跳验证**: 定期验证，防止离线破解
4. **脚本保护**: 脚本加密存储，授权验证后下发
