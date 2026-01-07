package main

import (
	"fmt"
	"log"
	"os"
	"time"

	license "license-server/sdk/go"
)

// 配置
const (
	ServerURL = "http://localhost:8080"
	AppKey    = "18563102289d9bab79daa5e77479eb9a" // 替换为你的 AppKey
)

func main() {
	fmt.Println("========================================")
	fmt.Println("       License SDK 完整功能演示")
	fmt.Println("========================================")

	// 选择演示模式
	fmt.Println("\n选择演示模式:")
	fmt.Println("  1. 授权码激活模式")
	fmt.Println("  2. 账号密码登录模式")
	fmt.Println("  3. 热更新功能")
	fmt.Println("  4. 数据同步功能")
	fmt.Println("  5. 安全脚本功能")
	fmt.Println("  6. 安全客户端功能")
	fmt.Println("  7. 完整工作流演示")
	fmt.Println("  0. 退出")

	var choice int
	fmt.Print("\n请输入选项 (0-7): ")
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		demoActivate()
	case 2:
		demoLogin()
	case 3:
		demoHotUpdate()
	case 4:
		demoDataSync()
	case 5:
		demoSecureScript()
	case 6:
		demoSecureClient()
	case 7:
		demoFullWorkflow()
	case 0:
		fmt.Println("退出")
		return
	default:
		fmt.Println("无效选项")
	}
}

// ==================== 1. 授权码激活模式 ====================
func demoActivate() {
	fmt.Println("\n========== 授权码激活模式 ==========")

	// 创建客户端
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithOfflineGraceDays(7),
		license.WithEncryptCache(true),
		license.WithTimeout(30*time.Second),
	)
	defer client.Close()

	fmt.Printf("服务器地址: %s\n", client.GetServerURL())
	fmt.Printf("机器码: %s\n", client.GetMachineID())

	// 获取授权码
	licenseKey := os.Getenv("LICENSE_KEY")
	if licenseKey == "" {
		fmt.Print("请输入授权码: ")
		fmt.Scanln(&licenseKey)
	}

	// 激活授权
	fmt.Println("\n正在激活...")
	result, err := client.Activate(licenseKey)
	if err != nil {
		log.Fatalf("激活失败: %v", err)
	}

	fmt.Println("\n✓ 激活成功!")
	fmt.Printf("  授权ID: %s\n", result.LicenseID)
	fmt.Printf("  设备ID: %s\n", result.DeviceID)
	fmt.Printf("  类型: %s\n", result.Type)
	fmt.Printf("  剩余天数: %d\n", result.RemainingDays)
	fmt.Printf("  功能列表: %v\n", result.Features)
	if result.ExpireAt != nil {
		fmt.Printf("  过期时间: %s\n", *result.ExpireAt)
	}

	// 验证授权状态
	fmt.Println("\n--- 验证授权状态 ---")
	if client.IsValid() {
		fmt.Println("✓ 授权有效")
	} else {
		fmt.Println("✗ 授权无效")
	}

	// 发送心跳
	fmt.Println("\n--- 发送心跳 ---")
	if client.Heartbeat() {
		fmt.Println("✓ 心跳成功")
	} else {
		fmt.Println("✗ 心跳失败")
	}

	// 检查功能权限
	fmt.Println("\n--- 功能权限检查 ---")
	features := client.GetFeatures()
	fmt.Printf("可用功能: %v\n", features)

	testFeatures := []string{"basic", "export", "print", "advanced"}
	for _, f := range testFeatures {
		if client.HasFeature(f) {
			fmt.Printf("  ✓ %s: 已授权\n", f)
		} else {
			fmt.Printf("  ✗ %s: 未授权\n", f)
		}
	}

	fmt.Println("\n授权码激活演示完成!")
}

// ==================== 2. 账号密码登录模式 ====================
func demoLogin() {
	fmt.Println("\n========== 账号密码登录模式 ==========")

	// 创建客户端
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithOfflineGraceDays(7),
		license.WithEncryptCache(true),
	)
	defer client.Close()

	fmt.Printf("服务器地址: %s\n", client.GetServerURL())
	fmt.Printf("机器码: %s\n", client.GetMachineID())

	// 获取账号密码
	email := os.Getenv("USER_EMAIL")
	password := os.Getenv("USER_PASSWORD")
	if email == "" {
		fmt.Print("请输入邮箱: ")
		fmt.Scanln(&email)
	}
	if password == "" {
		fmt.Print("请输入密码: ")
		fmt.Scanln(&password)
	}

	// 登录
	fmt.Println("\n正在登录...")
	result, err := client.Login(email, password)
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}

	fmt.Println("\n✓ 登录成功!")
	fmt.Printf("  订阅ID: %s\n", result.SubscriptionID)
	fmt.Printf("  设备ID: %s\n", result.DeviceID)
	fmt.Printf("  套餐类型: %s\n", result.PlanType)
	fmt.Printf("  剩余天数: %d\n", result.RemainingDays)
	fmt.Printf("  功能列表: %v\n", result.Features)
	fmt.Printf("  邮箱: %s\n", result.Email)
	if result.ExpireAt != nil {
		fmt.Printf("  过期时间: %s\n", *result.ExpireAt)
	}

	// 验证订阅状态
	fmt.Println("\n--- 验证订阅状态 ---")
	if client.SubscriptionVerify() {
		fmt.Println("✓ 订阅有效")
	} else {
		fmt.Println("✗ 订阅无效")
	}

	// 发送订阅心跳
	fmt.Println("\n--- 发送订阅心跳 ---")
	if client.SubscriptionHeartbeat() {
		fmt.Println("✓ 心跳成功")
	} else {
		fmt.Println("✗ 心跳失败")
	}

	// 检查功能权限
	fmt.Println("\n--- 功能权限检查 ---")
	features := client.GetFeatures()
	fmt.Printf("可用功能: %v\n", features)

	for _, f := range features {
		fmt.Printf("  ✓ %s: 已授权\n", f)
	}

	// 获取授权信息
	fmt.Println("\n--- 授权信息 ---")
	info := client.GetLicenseInfo()
	if info != nil {
		fmt.Printf("  有效状态: %v\n", info.Valid)
		fmt.Printf("  剩余天数: %d\n", info.RemainingDays)
		fmt.Printf("  套餐类型: %s\n", info.PlanType)
	}

	fmt.Println("\n账号密码登录演示完成!")
}

// ==================== 3. 热更新功能 ====================
func demoHotUpdate() {
	fmt.Println("\n========== 热更新功能 ==========")

	// 创建客户端并登录
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithSkipVerify(true),
	)
	defer client.Close()

	// 登录
	fmt.Println("登录中...")
	_, err := client.Login("test@test.com", "Test123456")
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	fmt.Println("✓ 登录成功")

	// 创建热更新管理器
	fmt.Println("\n--- 创建热更新管理器 ---")
	updateManager := license.NewHotUpdateManager(client, "1.0.0",
		license.WithUpdateDir("./updates"),
		license.WithBackupDir("./backups"),
		license.WithUpdateCallback(func(status license.HotUpdateStatus, progress float64, err error) {
			fmt.Printf("  [回调] 状态: %s, 进度: %.1f%%", status, progress*100)
			if err != nil {
				fmt.Printf(", 错误: %v", err)
			}
			fmt.Println()
		}),
	)

	fmt.Printf("当前版本: %s\n", updateManager.GetCurrentVersion())

	// 检查更新
	fmt.Println("\n--- 检查更新 ---")
	updateInfo, err := updateManager.CheckUpdate()
	if err != nil {
		fmt.Printf("检查更新失败: %v\n", err)
	} else {
		fmt.Printf("有更新: %v\n", updateInfo.HasUpdate)
		if updateInfo.HasUpdate {
			fmt.Printf("  目标版本: %s\n", updateInfo.ToVersion)
			fmt.Printf("  更新类型: %s\n", updateInfo.UpdateType)
			fmt.Printf("  强制更新: %v\n", updateInfo.ForceUpdate)
			fmt.Printf("  更新日志: %s\n", updateInfo.Changelog)

			// 下载更新（如果有）
			fmt.Println("\n--- 下载更新 ---")
			updateFile, err := updateManager.DownloadUpdate(updateInfo)
			if err != nil {
				fmt.Printf("下载失败: %v\n", err)
			} else {
				fmt.Printf("✓ 下载完成: %s\n", updateFile)

				// 应用更新
				fmt.Println("\n--- 应用更新 ---")
				err = updateManager.ApplyUpdate(updateInfo, updateFile, "./app")
				if err != nil {
					fmt.Printf("应用更新失败: %v\n", err)
				} else {
					fmt.Println("✓ 更新应用成功")
				}
			}
		}
	}

	// 获取更新历史
	fmt.Println("\n--- 更新历史 ---")
	history, err := updateManager.GetUpdateHistory()
	if err != nil {
		fmt.Printf("获取历史失败: %v\n", err)
	} else {
		fmt.Printf("历史记录数: %d\n", len(history))
		for i, record := range history {
			fmt.Printf("  %d. %v\n", i+1, record)
		}
	}

	fmt.Println("\n热更新功能演示完成!")
}

// ==================== 4. 数据同步功能 ====================
func demoDataSync() {
	fmt.Println("\n========== 数据同步功能 ==========")

	// 创建客户端并登录
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithSkipVerify(true),
	)
	defer client.Close()

	// 登录
	fmt.Println("登录中...")
	_, err := client.Login("test@test.com", "Test123456")
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	fmt.Println("✓ 登录成功")

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 获取表列表
	fmt.Println("\n--- 获取表列表 ---")
	tables, err := syncClient.GetTableList()
	if err != nil {
		fmt.Printf("获取表列表失败: %v\n", err)
	} else {
		fmt.Printf("表数量: %d\n", len(tables))
		for _, table := range tables {
			fmt.Printf("  - %s (记录数: %d)\n", table.TableName, table.RecordCount)
		}
	}

	// 推送数据示例
	fmt.Println("\n--- 推送数据 ---")
	tableName := "demo_data"
	recordID := fmt.Sprintf("demo_%d", time.Now().Unix())
	testData := map[string]interface{}{
		"name":       "演示数据",
		"value":      123,
		"created_at": time.Now().Format(time.RFC3339),
	}

	result, err := syncClient.PushRecord(tableName, recordID, testData, 0)
	if err != nil {
		fmt.Printf("推送失败: %v\n", err)
	} else {
		fmt.Printf("✓ 推送成功! 状态: %s, 版本: %d\n", result.Status, result.Version)
	}

	// 拉取数据
	fmt.Println("\n--- 拉取数据 ---")
	records, serverTime, err := syncClient.PullTable(tableName, 0)
	if err != nil {
		fmt.Printf("拉取失败: %v\n", err)
	} else {
		fmt.Printf("✓ 拉取成功! 记录数: %d, 服务器时间: %d\n", len(records), serverTime)
		for _, r := range records {
			fmt.Printf("  - ID: %s, 版本: %d\n", r.ID, r.Version)
		}
	}

	// 批量推送
	fmt.Println("\n--- 批量推送 ---")
	batchRecords := []license.PushRecordItem{
		{RecordID: fmt.Sprintf("batch_1_%d", time.Now().Unix()), Data: map[string]interface{}{"name": "批量1"}, Version: 0},
		{RecordID: fmt.Sprintf("batch_2_%d", time.Now().Unix()), Data: map[string]interface{}{"name": "批量2"}, Version: 0},
	}
	batchResults, err := syncClient.PushRecordBatch("demo_batch", batchRecords)
	if err != nil {
		fmt.Printf("批量推送失败: %v\n", err)
	} else {
		fmt.Printf("✓ 批量推送成功! 结果数: %d\n", len(batchResults))
	}

	// 获取同步状态
	fmt.Println("\n--- 同步状态 ---")
	status, err := syncClient.GetSyncStatus()
	if err != nil {
		fmt.Printf("获取状态失败: %v\n", err)
	} else {
		fmt.Printf("服务器时间: %d\n", status.ServerTime)
		fmt.Printf("待处理变更: %d\n", status.PendingChanges)
	}

	// 自动同步管理器示例
	fmt.Println("\n--- 自动同步管理器 ---")
	autoSync := syncClient.NewAutoSyncManager([]string{"demo_data", "demo_batch"}, 30*time.Second)
	autoSync.OnPull(func(tableName string, records []license.SyncRecord, deletes []string) error {
		fmt.Printf("  [自动同步] 表: %s, 更新: %d, 删除: %d\n", tableName, len(records), len(deletes))
		return nil
	})
	autoSync.Start()
	fmt.Println("自动同步已启动 (30秒间隔)")

	// 手动触发一次同步
	autoSync.SyncNow()

	// 停止自动同步
	time.Sleep(1 * time.Second)
	autoSync.Stop()
	fmt.Println("自动同步已停止")

	fmt.Println("\n数据同步功能演示完成!")
}

// ==================== 5. 安全脚本功能 ====================
func demoSecureScript() {
	fmt.Println("\n========== 安全脚本功能 ==========")

	// 创建客户端并登录
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithSkipVerify(true),
	)
	defer client.Close()

	// 登录
	fmt.Println("登录中...")
	_, err := client.Login("test@test.com", "Test123456")
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	fmt.Println("✓ 登录成功")

	// 创建安全脚本管理器
	fmt.Println("\n--- 创建安全脚本管理器 ---")
	scriptManager := license.NewSecureScriptManager(client,
		license.WithAppSecret("your-app-secret"),
		license.WithExecuteCallback(func(scriptID string, status string, err error) {
			fmt.Printf("  [回调] 脚本: %s, 状态: %s", scriptID, status)
			if err != nil {
				fmt.Printf(", 错误: %v", err)
			}
			fmt.Println()
		}),
	)
	fmt.Println("✓ 管理器创建成功")

	// 获取脚本版本列表
	fmt.Println("\n--- 获取脚本版本列表 ---")
	versions, err := scriptManager.GetScriptVersions()
	if err != nil {
		fmt.Printf("获取版本列表失败: %v\n", err)
	} else {
		fmt.Printf("可用脚本数: %d\n", len(versions))
		for _, v := range versions {
			fmt.Printf("  - %s (v%s): %s\n", v.ScriptID, v.Version, v.Name)
		}
	}

	// 获取并执行脚本（如果有可用脚本）
	if len(versions) > 0 {
		scriptID := versions[0].ScriptID
		fmt.Printf("\n--- 获取脚本: %s ---\n", scriptID)

		script, err := scriptManager.FetchScript(scriptID)
		if err != nil {
			fmt.Printf("获取脚本失败: %v\n", err)
		} else {
			fmt.Printf("✓ 获取成功!\n")
			fmt.Printf("  版本: %s\n", script.Version)
			fmt.Printf("  内容长度: %d 字节\n", len(script.Content))
			fmt.Printf("  过期时间: %s\n", script.ExpiresAt.Format(time.RFC3339))

			// 执行脚本
			fmt.Println("\n--- 执行脚本 ---")
			result, err := scriptManager.ExecuteScript(scriptID, nil,
				func(content []byte, args map[string]interface{}) (string, error) {
					// 这里是你的脚本执行逻辑
					fmt.Printf("  执行脚本内容 (%d 字节)...\n", len(content))
					return "执行成功", nil
				})
			if err != nil {
				fmt.Printf("执行失败: %v\n", err)
			} else {
				fmt.Printf("✓ 执行结果: %s\n", result)
			}
		}
	} else {
		fmt.Println("\n没有可用的脚本")
	}

	// 缓存管理
	fmt.Println("\n--- 缓存管理 ---")
	scriptManager.ClearCache()
	fmt.Println("✓ 缓存已清理")

	fmt.Println("\n安全脚本功能演示完成!")
}

// ==================== 6. 安全客户端功能 ====================
func demoSecureClient() {
	fmt.Println("\n========== 安全客户端功能 ==========")

	// 创建基础客户端
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithSkipVerify(true),
	)
	defer client.Close()

	// 登录
	fmt.Println("登录中...")
	_, err := client.Login("test@test.com", "Test123456")
	if err != nil {
		log.Fatalf("登录失败: %v", err)
	}
	fmt.Println("✓ 登录成功")

	// 创建安全客户端
	fmt.Println("\n--- 创建安全客户端 ---")
	secureClient := license.NewSecureClient(client)
	defer secureClient.Close()

	fmt.Printf("安全令牌: %s\n", secureClient.GetSecurityToken())

	// 安全验证
	fmt.Println("\n--- 安全验证 ---")
	if secureClient.IsValid() {
		fmt.Println("✓ 安全验证通过")
	} else {
		fmt.Println("✗ 安全验证失败")
	}

	// 创建强化安全客户端
	fmt.Println("\n--- 创建强化安全客户端 ---")
	hardenedClient := license.NewHardenedSecureClient(client)
	defer hardenedClient.Close()

	// 获取安全状态
	fmt.Println("\n--- 安全状态 ---")
	status := hardenedClient.GetSecurityStatus()
	fmt.Printf("安全状态: %v\n", status)

	// 分布式验证
	fmt.Println("\n--- 分布式验证 ---")
	distResult := hardenedClient.IsValidDistributed()
	if distResult != nil {
		fmt.Println("分布式验证令牌已生成")
		for i := 0; i < 4; i++ {
			verified := hardenedClient.VerifyDistributedToken(distResult, i)
			if verified {
				fmt.Printf("  ✓ Token %d: 验证通过\n", i)
			} else {
				fmt.Printf("  ✗ Token %d: 验证失败\n", i)
			}
		}
	}

	fmt.Println("\n安全客户端功能演示完成!")
}

// ==================== 7. 完整工作流演示 ====================
func demoFullWorkflow() {
	fmt.Println("\n========== 完整工作流演示 ==========")

	// 步骤 1: 创建客户端
	fmt.Println("\n[步骤 1] 创建客户端")
	client := license.NewClient(ServerURL, AppKey,
		license.WithAppVersion("1.0.0"),
		license.WithOfflineGraceDays(7),
		license.WithEncryptCache(true),
		license.WithTimeout(30*time.Second),
	)
	defer client.Close()

	fmt.Printf("  服务器: %s\n", client.GetServerURL())
	fmt.Printf("  机器码: %s\n", client.GetMachineID())
	fmt.Println("  ✓ 客户端创建成功")

	// 步骤 2: 登录
	fmt.Println("\n[步骤 2] 账号登录")
	loginResult, err := client.Login("test@test.com", "Test123456")
	if err != nil {
		log.Fatalf("  ✗ 登录失败: %v", err)
	}
	fmt.Printf("  订阅ID: %s\n", loginResult.SubscriptionID)
	fmt.Printf("  套餐: %s\n", loginResult.PlanType)
	fmt.Printf("  剩余天数: %d\n", loginResult.RemainingDays)
	fmt.Println("  ✓ 登录成功")

	// 步骤 3: 验证授权
	fmt.Println("\n[步骤 3] 验证授权")
	if client.IsValid() {
		fmt.Println("  ✓ 本地授权有效")
	}
	if client.SubscriptionVerify() {
		fmt.Println("  ✓ 服务器验证通过")
	}

	// 步骤 4: 功能检查
	fmt.Println("\n[步骤 4] 功能权限")
	features := client.GetFeatures()
	fmt.Printf("  可用功能: %v\n", features)
	for _, f := range features {
		fmt.Printf("  ✓ %s\n", f)
	}

	// 步骤 5: 发送心跳
	fmt.Println("\n[步骤 5] 发送心跳")
	if client.SubscriptionHeartbeat() {
		fmt.Println("  ✓ 心跳成功")
	}

	// 步骤 6: 数据同步
	fmt.Println("\n[步骤 6] 数据同步")
	syncClient := client.NewDataSyncClient()
	tables, _ := syncClient.GetTableList()
	fmt.Printf("  同步表数量: %d\n", len(tables))
	fmt.Println("  ✓ 数据同步就绪")

	// 步骤 7: 热更新检查
	fmt.Println("\n[步骤 7] 热更新检查")
	updateManager := license.NewHotUpdateManager(client, "1.0.0")
	updateInfo, err := updateManager.CheckUpdate()
	if err != nil {
		fmt.Printf("  检查失败: %v\n", err)
	} else {
		if updateInfo.HasUpdate {
			fmt.Printf("  ✓ 发现新版本: %s\n", updateInfo.ToVersion)
		} else {
			fmt.Println("  ✓ 已是最新版本")
		}
	}

	// 步骤 8: 安全脚本
	fmt.Println("\n[步骤 8] 安全脚本")
	scriptManager := license.NewSecureScriptManager(client,
		license.WithAppSecret("your-app-secret"),
	)
	versions, _ := scriptManager.GetScriptVersions()
	fmt.Printf("  可用脚本数: %d\n", len(versions))
	fmt.Println("  ✓ 安全脚本就绪")

	// 步骤 9: 安全客户端
	fmt.Println("\n[步骤 9] 安全客户端")
	secureClient := license.NewSecureClient(client)
	defer secureClient.Close()
	if secureClient.IsValid() {
		fmt.Println("  ✓ 安全验证通过")
	}

	// 步骤 10: 模拟业务运行
	fmt.Println("\n[步骤 10] 模拟业务运行")
	fmt.Println("  应用程序正在运行...")
	fmt.Println("  (按 Ctrl+C 退出)")

	// 启动心跳定时器
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// 模拟运行 5 秒
	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			if client.SubscriptionHeartbeat() {
				fmt.Println("  [心跳] 成功")
			}
		case <-timeout:
			fmt.Println("\n========================================")
			fmt.Println("       完整工作流演示完成!")
			fmt.Println("========================================")
			return
		}
	}
}
