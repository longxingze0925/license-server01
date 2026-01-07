@echo off
chcp 65001 >nul
echo ========================================
echo   自签名 SSL 证书生成脚本
echo   适用于 IP 地址部署（无需域名备案）
echo ========================================
echo.

:: 设置你的服务器 IP 地址
set /p SERVER_IP="请输入服务器 IP 地址: "

:: 创建证书目录
if not exist "..\certs" mkdir "..\certs"

:: 创建 OpenSSL 配置文件
echo [req] > cert.conf
echo default_bits = 2048 >> cert.conf
echo prompt = no >> cert.conf
echo default_md = sha256 >> cert.conf
echo distinguished_name = dn >> cert.conf
echo x509_extensions = v3_req >> cert.conf
echo. >> cert.conf
echo [dn] >> cert.conf
echo CN = %SERVER_IP% >> cert.conf
echo. >> cert.conf
echo [v3_req] >> cert.conf
echo subjectAltName = @alt_names >> cert.conf
echo. >> cert.conf
echo [alt_names] >> cert.conf
echo IP.1 = %SERVER_IP% >> cert.conf
echo IP.2 = 127.0.0.1 >> cert.conf

:: 生成证书
echo.
echo 正在生成证书...
openssl req -x509 -nodes -days 365 -newkey rsa:2048 ^
  -keyout ..\certs\server.key ^
  -out ..\certs\server.crt ^
  -config cert.conf

:: 清理临时文件
del cert.conf

echo.
echo ========================================
echo   证书生成完成！
echo   证书位置: certs\server.crt
echo   私钥位置: certs\server.key
echo   有效期: 365 天
echo ========================================
echo.
echo 注意事项:
echo 1. 客户端需要信任此证书或跳过验证
echo 2. 建议每年更新证书
echo 3. 请妥善保管 server.key 私钥文件
echo.
pause
