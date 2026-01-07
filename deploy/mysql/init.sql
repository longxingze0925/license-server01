-- License Server 数据库初始化脚本
-- 创建数据库和用户权限

-- 确保使用 UTF8MB4 字符集
ALTER DATABASE license_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 授予应用用户权限
GRANT ALL PRIVILEGES ON license_server.* TO 'license_admin'@'%';
FLUSH PRIVILEGES;
