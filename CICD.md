# CI/CD 自动化指南

本指南介绍如何使用 GitHub 管理代码版本，并实现自动构建和部署。

## 目录

1. [工作流程概述](#工作流程概述)
2. [初始设置](#初始设置)
3. [日常开发流程](#日常开发流程)
4. [版本发布](#版本发布)
5. [自动部署配置](#自动部署配置)
6. [常见问题](#常见问题)

---

## 工作流程概述

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  本地开发    │────▶│  推送代码    │────▶│ GitHub Actions│────▶│  自动部署   │
│  修改代码    │     │  到 GitHub  │     │  自动构建    │     │  到服务器   │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

### 触发条件

| 事件 | 触发操作 |
|------|----------|
| 推送到 main 分支 | 自动构建 + 测试 + 部署 |
| 创建版本标签 (v*) | 构建 + 发布 Release |
| Pull Request | 仅构建和测试 |

---

## 初始设置

### 1. 创建 GitHub 仓库

```bash
# 在 GitHub 上创建新仓库，然后：

cd 用户管理系统

# 初始化 Git（如果还没有）
git init

# 添加远程仓库
git remote add origin https://github.com/你的用户名/license-server.git

# 添加所有文件
git add .

# 首次提交
git commit -m "feat: initial commit"

# 推送到 GitHub
git push -u origin main
```

### 2. 配置 GitHub Secrets

进入仓库 Settings → Secrets and variables → Actions，添加以下 Secrets：

| Secret 名称 | 说明 | 示例 |
|------------|------|------|
| `DEPLOY_HOST` | 服务器 IP | `123.45.67.89` |
| `DEPLOY_USER` | SSH 用户名 | `root` |
| `DEPLOY_KEY` | SSH 私钥 | (完整私钥内容) |
| `DEPLOY_PATH` | 部署路径 | `/opt/license-server` |

#### 生成 SSH 密钥对

```bash
# 在本地生成密钥对
ssh-keygen -t ed25519 -C "github-actions-deploy" -f deploy_key

# deploy_key 是私钥 → 添加到 GitHub Secrets (DEPLOY_KEY)
# deploy_key.pub 是公钥 → 添加到服务器 authorized_keys

# 在服务器上添加公钥
cat deploy_key.pub >> ~/.ssh/authorized_keys
```

### 3. 服务器准备

```bash
# SSH 登录服务器
ssh root@你的服务器

# 克隆仓库
git clone https://github.com/你的用户名/license-server.git /opt/license-server

# 进入目录
cd /opt/license-server

# 首次安装
chmod +x install-https.sh
./install-https.sh

# 设置更新脚本权限
chmod +x update.sh
```

---

## 日常开发流程

### 修改代码并推送

```bash
# 1. 修改代码
# ... 编辑文件 ...

# 2. 查看修改
git status
git diff

# 3. 添加修改
git add .

# 4. 提交
git commit -m "feat: 添加新功能"

# 5. 推送到 GitHub
git push origin main
```

### 提交信息规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

| 前缀 | 说明 | 示例 |
|------|------|------|
| `feat:` | 新功能 | `feat: 添加用户导出功能` |
| `fix:` | 修复 Bug | `fix: 修复登录验证问题` |
| `docs:` | 文档更新 | `docs: 更新 API 文档` |
| `style:` | 代码格式 | `style: 格式化代码` |
| `refactor:` | 重构 | `refactor: 重构授权模块` |
| `test:` | 测试 | `test: 添加单元测试` |
| `chore:` | 构建/工具 | `chore: 更新依赖` |

---

## 版本发布

### 使用发布脚本

```bash
# 发布指定版本
./release.sh 1.0.0 --push

# 自动递增版本
./release.sh patch --push    # 0.1.0 → 0.1.1
./release.sh minor --push    # 0.1.0 → 0.2.0
./release.sh major --push    # 0.1.0 → 1.0.0

# 预览（不实际执行）
./release.sh 1.0.0 --dry-run
```

### 手动创建版本

```bash
# 1. 更新版本文件
echo "1.0.0" > VERSION

# 2. 提交
git add VERSION
git commit -m "chore: release v1.0.0"

# 3. 创建标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 4. 推送
git push origin main
git push origin v1.0.0
```

### 版本发布后

GitHub Actions 会自动：
1. 运行测试
2. 构建 Docker 镜像
3. 创建 GitHub Release
4. 部署到服务器（如已配置）

---

## 自动部署配置

### 方式一：GitHub Actions 自动部署

已在 `.github/workflows/ci-cd.yml` 中配置，需要设置 Secrets。

推送到 main 分支后会自动部署。

### 方式二：Webhook 触发部署

1. 在服务器上设置 Webhook 接收端点
2. GitHub 仓库设置 Webhook
3. 收到推送事件时自动执行 `update.sh`

### 方式三：定时检查更新

```bash
# 添加 cron 任务，每小时检查更新
echo "0 * * * * root cd /opt/license-server && git fetch && git diff --quiet origin/main || ./update.sh" > /etc/cron.d/license-server-update
```

### 方式四：手动更新

```bash
# SSH 登录服务器
ssh root@你的服务器

# 执行更新
cd /opt/license-server
./update.sh
```

---

## 服务器更新命令

```bash
# 更新到最新版本
./update.sh

# 更新到指定版本
./update.sh v1.2.0

# 强制更新（丢弃本地修改）
./update.sh --force

# 回滚到上一版本
./update.sh rollback

# 查看服务状态
./update.sh status

# 清理旧镜像
./update.sh cleanup
```

---

## GitHub Actions 工作流说明

### 工作流文件

位置：`.github/workflows/ci-cd.yml`

### 工作流任务

```yaml
jobs:
  backend:      # 后端构建和测试
  frontend:     # 前端构建
  docker:       # 构建 Docker 镜像
  release:      # 创建 GitHub Release
  deploy:       # 部署到服务器
```

### 查看构建状态

1. 进入 GitHub 仓库
2. 点击 "Actions" 标签
3. 查看工作流运行状态

### 构建徽章

在 README 中添加：

```markdown
![CI/CD](https://github.com/你的用户名/license-server/actions/workflows/ci-cd.yml/badge.svg)
```

---

## 常见问题

### Q: 推送后没有触发构建？

检查：
1. 工作流文件路径是否正确：`.github/workflows/ci-cd.yml`
2. 分支名称是否匹配（main/master）
3. 查看 Actions 页面是否有错误

### Q: 构建失败怎么办？

1. 查看 Actions 页面的错误日志
2. 本地运行测试：`go test ./...`
3. 检查依赖是否正确

### Q: 自动部署失败？

检查：
1. SSH 密钥是否正确配置
2. 服务器路径是否存在
3. 服务器是否有执行权限

### Q: 如何回滚到旧版本？

```bash
# 方式一：使用更新脚本
./update.sh v1.0.0

# 方式二：Git 回滚
git checkout v1.0.0
docker compose -f docker-compose.https.yml up -d --build
```

### Q: 如何查看部署日志？

```bash
# 服务器上查看
docker compose logs -f

# GitHub Actions 日志
# 在 Actions 页面查看
```

---

## 文件清单

| 文件 | 说明 |
|------|------|
| `.github/workflows/ci-cd.yml` | GitHub Actions 工作流 |
| `.gitignore` | Git 忽略文件 |
| `VERSION` | 版本号文件 |
| `release.sh` | 版本发布脚本 |
| `update.sh` | 服务器更新脚本 |

---

## 快速参考

```bash
# 日常开发
git add . && git commit -m "feat: 功能描述" && git push

# 发布新版本
./release.sh patch --push

# 服务器更新
./update.sh

# 查看状态
./update.sh status
```
