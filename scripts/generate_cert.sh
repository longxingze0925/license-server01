#!/bin/bash
# 自签名 SSL 证书生成脚本
# 适用于 IP 地址部署（无需域名备案）

echo "========================================"
echo "  自签名 SSL 证书生成脚本"
echo "  适用于 IP 地址部署（无需域名备案）"
echo "========================================"
echo

# 获取服务器 IP
read -p "请输入服务器 IP 地址: " SERVER_IP

# 创建证书目录
mkdir -p ../certs

# 生成证书
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ../certs/server.key \
  -out ../certs/server.crt \
  -subj "/CN=${SERVER_IP}" \
  -addext "subjectAltName=IP:${SERVER_IP},IP:127.0.0.1"

# 设置权限
chmod 600 ../certs/server.key
chmod 644 ../certs/server.crt

echo
echo "========================================"
echo "  证书生成完成！"
echo "  证书位置: certs/server.crt"
echo "  私钥位置: certs/server.key"
echo "  有效期: 365 天"
echo "========================================"
echo
echo "注意事项:"
echo "1. 客户端需要信任此证书或跳过验证"
echo "2. 建议每年更新证书"
echo "3. 请妥善保管 server.key 私钥文件"
