package license

import (
	"fmt"
	"testing"
	"time"
)

// TestIntegration_DataSync_GetTableList 测试获取表列表
func TestIntegration_DataSync_GetTableList(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 获取表列表 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	fmt.Println("登录中...")
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	fmt.Println("登录成功!")

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 获取表列表
	fmt.Println("\n获取表列表...")
	tables, err := syncClient.GetTableList()
	if err != nil {
		fmt.Printf("获取表列表结果: %v\n", err)
	} else {
		fmt.Printf("表数量: %d\n", len(tables))
		for _, table := range tables {
			fmt.Printf("  表名: %s, 记录数: %d, 最后更新: %s\n",
				table.TableName, table.RecordCount, table.LastUpdated)
		}
	}

	fmt.Println("\n数据同步 - 获取表列表测试通过!")
}

// TestIntegration_DataSync_PushAndPullRecord 测试推送和拉取记录
func TestIntegration_DataSync_PushAndPullRecord(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 推送和拉取记录 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	fmt.Println("登录中...")
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 测试数据
	tableName := "test_configs"
	recordID := fmt.Sprintf("test_record_%d", time.Now().UnixNano())
	testData := map[string]interface{}{
		"key":   "test_key",
		"value": "test_value",
		"count": 42,
	}

	// 推送记录
	fmt.Printf("\n推送记录到表 %s...\n", tableName)
	result, err := syncClient.PushRecord(tableName, recordID, testData, 0)
	if err != nil {
		fmt.Printf("推送记录结果: %v\n", err)
	} else {
		fmt.Printf("推送成功! 状态: %s, 版本: %d\n", result.Status, result.Version)
	}

	// 拉取记录
	fmt.Printf("\n从表 %s 拉取记录...\n", tableName)
	records, serverTime, err := syncClient.PullTable(tableName, 0)
	if err != nil {
		fmt.Printf("拉取记录结果: %v\n", err)
	} else {
		fmt.Printf("拉取成功! 记录数: %d, 服务器时间: %d\n", len(records), serverTime)
		for i, record := range records {
			fmt.Printf("  记录 %d: ID=%s, 版本=%d\n", i+1, record.ID, record.Version)
		}
	}

	fmt.Println("\n数据同步 - 推送和拉取记录测试通过!")
}

// TestIntegration_DataSync_BatchPush 测试批量推送
func TestIntegration_DataSync_BatchPush(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 批量推送 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 准备批量数据
	tableName := "test_batch"
	records := []PushRecordItem{
		{
			RecordID: fmt.Sprintf("batch_1_%d", time.Now().UnixNano()),
			Data:     map[string]interface{}{"name": "Item 1", "value": 100},
			Version:  0,
		},
		{
			RecordID: fmt.Sprintf("batch_2_%d", time.Now().UnixNano()),
			Data:     map[string]interface{}{"name": "Item 2", "value": 200},
			Version:  0,
		},
		{
			RecordID: fmt.Sprintf("batch_3_%d", time.Now().UnixNano()),
			Data:     map[string]interface{}{"name": "Item 3", "value": 300},
			Version:  0,
		},
	}

	// 批量推送
	fmt.Printf("\n批量推送 %d 条记录到表 %s...\n", len(records), tableName)
	results, err := syncClient.PushRecordBatch(tableName, records)
	if err != nil {
		fmt.Printf("批量推送结果: %v\n", err)
	} else {
		fmt.Printf("批量推送成功! 结果数: %d\n", len(results))
		for _, r := range results {
			fmt.Printf("  记录 %s: 状态=%s, 版本=%d\n", r.RecordID, r.Status, r.Version)
		}
	}

	fmt.Println("\n数据同步 - 批量推送测试通过!")
}

// TestIntegration_DataSync_DeleteRecord 测试删除记录
func TestIntegration_DataSync_DeleteRecord(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 删除记录 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 先创建一条记录
	tableName := "test_delete"
	recordID := fmt.Sprintf("delete_test_%d", time.Now().UnixNano())
	testData := map[string]interface{}{"name": "to_be_deleted"}

	fmt.Printf("\n先创建记录 %s...\n", recordID)
	_, err = syncClient.PushRecord(tableName, recordID, testData, 0)
	if err != nil {
		fmt.Printf("创建记录失败: %v\n", err)
	} else {
		fmt.Println("记录创建成功!")
	}

	// 删除记录
	fmt.Printf("\n删除记录 %s...\n", recordID)
	err = syncClient.DeleteRecord(tableName, recordID)
	if err != nil {
		fmt.Printf("删除记录结果: %v\n", err)
	} else {
		fmt.Println("删除成功!")
	}

	fmt.Println("\n数据同步 - 删除记录测试通过!")
}

// TestIntegration_DataSync_SyncTime 测试同步时间管理
func TestIntegration_DataSync_SyncTime(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 同步时间管理 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
	)
	defer client.Close()

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 测试设置和获取同步时间
	tableName := "test_table"
	testTime := time.Now().Unix()

	fmt.Printf("\n设置表 %s 的同步时间为 %d...\n", tableName, testTime)
	syncClient.SetLastSyncTime(tableName, testTime)

	gotTime := syncClient.GetLastSyncTime(tableName)
	fmt.Printf("获取到的同步时间: %d\n", gotTime)

	if gotTime != testTime {
		t.Errorf("同步时间不匹配: 期望 %d, 实际 %d", testTime, gotTime)
	}

	fmt.Println("\n数据同步 - 同步时间管理测试通过!")
}

// TestIntegration_DataSync_IncrementalSync 测试增量同步
func TestIntegration_DataSync_IncrementalSync(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 增量同步 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	tableName := "test_incremental"

	// 第一次全量拉取
	fmt.Println("\n第一次全量拉取...")
	records1, serverTime1, err := syncClient.PullTable(tableName, 0)
	if err != nil {
		fmt.Printf("全量拉取结果: %v\n", err)
	} else {
		fmt.Printf("全量拉取成功! 记录数: %d, 服务器时间: %d\n", len(records1), serverTime1)
	}

	// 推送新记录
	recordID := fmt.Sprintf("incr_%d", time.Now().UnixNano())
	fmt.Printf("\n推送新记录 %s...\n", recordID)
	_, err = syncClient.PushRecord(tableName, recordID, map[string]interface{}{"data": "new"}, 0)
	if err != nil {
		fmt.Printf("推送失败: %v\n", err)
	}

	// 增量拉取
	fmt.Printf("\n增量拉取 (since=%d)...\n", serverTime1)
	records2, serverTime2, err := syncClient.PullTable(tableName, serverTime1)
	if err != nil {
		fmt.Printf("增量拉取结果: %v\n", err)
	} else {
		fmt.Printf("增量拉取成功! 新记录数: %d, 服务器时间: %d\n", len(records2), serverTime2)
	}

	fmt.Println("\n数据同步 - 增量同步测试通过!")
}

// TestIntegration_DataSync_SyncTableFromServer 测试便捷同步方法
func TestIntegration_DataSync_SyncTableFromServer(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 便捷同步方法 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	tableName := "test_convenient"

	// 使用便捷方法同步
	fmt.Printf("\n使用 SyncTableFromServer 同步表 %s...\n", tableName)
	updates, deletes, serverTime, err := syncClient.SyncTableFromServer(tableName, 0)
	if err != nil {
		fmt.Printf("同步结果: %v\n", err)
	} else {
		fmt.Printf("同步成功!\n")
		fmt.Printf("  更新记录数: %d\n", len(updates))
		fmt.Printf("  删除记录数: %d\n", len(deletes))
		fmt.Printf("  服务器时间: %d\n", serverTime)
	}

	fmt.Println("\n数据同步 - 便捷同步方法测试通过!")
}

// TestIntegration_DataSync_GetSyncStatus 测试获取同步状态
func TestIntegration_DataSync_GetSyncStatus(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 获取同步状态 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 获取同步状态
	fmt.Println("\n获取同步状态...")
	status, err := syncClient.GetSyncStatus()
	if err != nil {
		fmt.Printf("获取同步状态结果: %v\n", err)
	} else {
		fmt.Printf("同步状态:\n")
		fmt.Printf("  最后同步时间: %d\n", status.LastSyncTime)
		fmt.Printf("  待处理变更: %d\n", status.PendingChanges)
		fmt.Printf("  服务器时间: %d\n", status.ServerTime)
		fmt.Printf("  表状态: %v\n", status.TableStatus)
	}

	fmt.Println("\n数据同步 - 获取同步状态测试通过!")
}

// TestIntegration_DataSync_PushAndGetChanges 测试推送和获取变更
func TestIntegration_DataSync_PushAndGetChanges(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 推送和获取变更 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 创建变更
	changes := []SyncChange{
		{
			Table:      "test_changes",
			RecordID:   fmt.Sprintf("change_%d", time.Now().UnixNano()),
			Operation:  "insert",
			Data:       map[string]interface{}{"name": "test change"},
			Version:    1,
			ChangeTime: time.Now().Unix(),
		},
	}

	// 推送变更
	fmt.Printf("\n推送 %d 个变更...\n", len(changes))
	results, err := syncClient.PushChanges(changes)
	if err != nil {
		fmt.Printf("推送变更结果: %v\n", err)
	} else {
		fmt.Printf("推送成功! 结果数: %d\n", len(results))
	}

	// 获取变更
	fmt.Println("\n获取变更...")
	gotChanges, serverTime, err := syncClient.GetChanges(0, []string{"test_changes"})
	if err != nil {
		fmt.Printf("获取变更结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 变更数: %d, 服务器时间: %d\n", len(gotChanges), serverTime)
	}

	fmt.Println("\n数据同步 - 推送和获取变更测试通过!")
}

// TestIntegration_DataSync_AutoSyncManager 测试自动同步管理器
func TestIntegration_DataSync_AutoSyncManager(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 自动同步管理器 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 创建自动同步管理器
	tables := []string{"test_auto_1", "test_auto_2"}
	manager := syncClient.NewAutoSyncManager(tables, 5*time.Second)

	// 设置回调
	pullCount := 0
	manager.OnPull(func(tableName string, records []SyncRecord, deletes []string) error {
		pullCount++
		fmt.Printf("  拉取回调: 表=%s, 记录数=%d, 删除数=%d\n", tableName, len(records), len(deletes))
		return nil
	})

	conflictCount := 0
	manager.OnConflict(func(tableName string, result SyncResult) error {
		conflictCount++
		fmt.Printf("  冲突回调: 表=%s, 记录=%s\n", tableName, result.RecordID)
		return nil
	})

	// 启动自动同步
	fmt.Println("\n启动自动同步...")
	manager.Start()

	// 等待一次同步完成
	time.Sleep(2 * time.Second)

	// 手动触发同步
	fmt.Println("\n手动触发同步...")
	manager.SyncNow()

	// 停止自动同步
	fmt.Println("\n停止自动同步...")
	manager.Stop()

	fmt.Printf("\n拉取回调次数: %d\n", pullCount)
	fmt.Printf("冲突回调次数: %d\n", conflictCount)

	fmt.Println("\n数据同步 - 自动同步管理器测试通过!")
}

// TestIntegration_DataSync_Configs 测试配置数据同步
func TestIntegration_DataSync_Configs(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 配置数据 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 保存配置
	configs := []ConfigData{
		{Key: "theme", Value: "dark", UpdatedAt: time.Now().Unix()},
		{Key: "language", Value: "zh-CN", UpdatedAt: time.Now().Unix()},
	}

	fmt.Printf("\n保存 %d 个配置...\n", len(configs))
	err = syncClient.SaveConfigs(configs)
	if err != nil {
		fmt.Printf("保存配置结果: %v\n", err)
	} else {
		fmt.Println("保存成功!")
	}

	// 获取配置
	fmt.Println("\n获取配置...")
	gotConfigs, serverTime, err := syncClient.GetConfigs(0)
	if err != nil {
		fmt.Printf("获取配置结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 配置数: %d, 服务器时间: %d\n", len(gotConfigs), serverTime)
		for _, cfg := range gotConfigs {
			fmt.Printf("  %s = %v\n", cfg.Key, cfg.Value)
		}
	}

	fmt.Println("\n数据同步 - 配置数据测试通过!")
}

// TestIntegration_DataSync_Workflows 测试工作流数据同步
func TestIntegration_DataSync_Workflows(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 工作流数据 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 保存工作流
	workflowID := fmt.Sprintf("wf_%d", time.Now().UnixNano())
	workflows := []WorkflowData{
		{
			ID:        workflowID,
			Name:      "测试工作流",
			Config:    map[string]interface{}{"steps": 3, "auto": true},
			Enabled:   true,
			UpdatedAt: time.Now().Unix(),
		},
	}

	fmt.Printf("\n保存工作流 %s...\n", workflowID)
	err = syncClient.SaveWorkflows(workflows)
	if err != nil {
		fmt.Printf("保存工作流结果: %v\n", err)
	} else {
		fmt.Println("保存成功!")
	}

	// 获取工作流
	fmt.Println("\n获取工作流...")
	gotWorkflows, serverTime, err := syncClient.GetWorkflows(0)
	if err != nil {
		fmt.Printf("获取工作流结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 工作流数: %d, 服务器时间: %d\n", len(gotWorkflows), serverTime)
	}

	// 删除工作流
	fmt.Printf("\n删除工作流 %s...\n", workflowID)
	err = syncClient.DeleteWorkflow(workflowID)
	if err != nil {
		fmt.Printf("删除工作流结果: %v\n", err)
	} else {
		fmt.Println("删除成功!")
	}

	fmt.Println("\n数据同步 - 工作流数据测试通过!")
}

// TestIntegration_DataSync_Materials 测试素材数据同步
func TestIntegration_DataSync_Materials(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 素材数据 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 保存素材
	materials := []MaterialData{
		{
			ID:        fmt.Sprintf("mat_%d", time.Now().UnixNano()),
			Type:      "text",
			Content:   "这是测试素材内容",
			Tags:      "test,demo",
			UpdatedAt: time.Now().Unix(),
		},
	}

	fmt.Printf("\n保存 %d 个素材...\n", len(materials))
	err = syncClient.SaveMaterials(materials)
	if err != nil {
		fmt.Printf("保存素材结果: %v\n", err)
	} else {
		fmt.Println("保存成功!")
	}

	// 获取素材
	fmt.Println("\n获取素材...")
	gotMaterials, serverTime, err := syncClient.GetMaterials(0)
	if err != nil {
		fmt.Printf("获取素材结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 素材数: %d, 服务器时间: %d\n", len(gotMaterials), serverTime)
	}

	fmt.Println("\n数据同步 - 素材数据测试通过!")
}

// TestIntegration_DataSync_Posts 测试帖子数据同步
func TestIntegration_DataSync_Posts(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 帖子数据 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 保存帖子
	postID := fmt.Sprintf("post_%d", time.Now().UnixNano())
	posts := []PostData{
		{
			ID:        postID,
			Content:   "这是测试帖子内容",
			Status:    "draft",
			GroupID:   "default",
			UpdatedAt: time.Now().Unix(),
		},
	}

	fmt.Printf("\n保存帖子 %s...\n", postID)
	err = syncClient.SavePosts(posts)
	if err != nil {
		fmt.Printf("保存帖子结果: %v\n", err)
	} else {
		fmt.Println("保存成功!")
	}

	// 获取帖子
	fmt.Println("\n获取帖子...")
	gotPosts, serverTime, err := syncClient.GetPosts(0, "")
	if err != nil {
		fmt.Printf("获取帖子结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 帖子数: %d, 服务器时间: %d\n", len(gotPosts), serverTime)
	}

	// 更新帖子状态
	fmt.Printf("\n更新帖子 %s 状态为 published...\n", postID)
	err = syncClient.UpdatePostStatus(postID, "published")
	if err != nil {
		fmt.Printf("更新状态结果: %v\n", err)
	} else {
		fmt.Println("更新成功!")
	}

	// 获取帖子分组
	fmt.Println("\n获取帖子分组...")
	groups, err := syncClient.GetPostGroups()
	if err != nil {
		fmt.Printf("获取分组结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 分组数: %d\n", len(groups))
		for _, g := range groups {
			fmt.Printf("  分组: %s (%s), 数量: %d\n", g.Name, g.ID, g.Count)
		}
	}

	fmt.Println("\n数据同步 - 帖子数据测试通过!")
}

// TestIntegration_DataSync_CommentScripts 测试评论话术同步
func TestIntegration_DataSync_CommentScripts(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 评论话术 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 先登录
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 保存评论话术
	scripts := []CommentScriptData{
		{
			ID:        fmt.Sprintf("script_%d", time.Now().UnixNano()),
			Content:   "这是一条测试评论话术",
			Category:  "general",
			UpdatedAt: time.Now().Unix(),
		},
	}

	fmt.Printf("\n保存 %d 条评论话术...\n", len(scripts))
	err = syncClient.SaveCommentScripts(scripts)
	if err != nil {
		fmt.Printf("保存评论话术结果: %v\n", err)
	} else {
		fmt.Println("保存成功!")
	}

	// 获取评论话术
	fmt.Println("\n获取评论话术...")
	gotScripts, serverTime, err := syncClient.GetCommentScripts(0, "")
	if err != nil {
		fmt.Printf("获取评论话术结果: %v\n", err)
	} else {
		fmt.Printf("获取成功! 话术数: %d, 服务器时间: %d\n", len(gotScripts), serverTime)
	}

	fmt.Println("\n数据同步 - 评论话术测试通过!")
}

// TestIntegration_DataSync_FullWorkflow 测试完整数据同步工作流
func TestIntegration_DataSync_FullWorkflow(t *testing.T) {
	fmt.Println("\n========== 集成测试: 数据同步 - 完整工作流 ==========")

	client := NewClient(IntegrationServerURL, IntegrationAppKey,
		WithAppVersion("1.0.0"),
		WithSkipVerify(true),
		WithTimeout(30*time.Second),
	)
	defer client.Close()

	// 步骤1: 登录
	fmt.Println("\n--- 步骤1: 登录 ---")
	_, err := client.Login(TestEmail, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	fmt.Println("登录成功!")

	// 创建数据同步客户端
	syncClient := client.NewDataSyncClient()

	// 步骤2: 获取同步状态
	fmt.Println("\n--- 步骤2: 获取同步状态 ---")
	status, err := syncClient.GetSyncStatus()
	if err != nil {
		fmt.Printf("获取状态失败: %v\n", err)
	} else {
		fmt.Printf("服务器时间: %d\n", status.ServerTime)
	}

	// 步骤3: 推送本地数据
	fmt.Println("\n--- 步骤3: 推送本地数据 ---")
	tableName := "test_workflow"
	localRecords := []map[string]interface{}{
		{"id": "local_1", "name": "Local Item 1"},
		{"id": "local_2", "name": "Local Item 2"},
	}
	results, err := syncClient.SyncTableToServer(tableName, localRecords, "id")
	if err != nil {
		fmt.Printf("推送失败: %v\n", err)
	} else {
		fmt.Printf("推送成功! 结果数: %d\n", len(results))
	}

	// 步骤4: 从服务器拉取数据
	fmt.Println("\n--- 步骤4: 从服务器拉取数据 ---")
	updates, deletes, serverTime, err := syncClient.SyncTableFromServer(tableName, 0)
	if err != nil {
		fmt.Printf("拉取失败: %v\n", err)
	} else {
		fmt.Printf("拉取成功! 更新: %d, 删除: %d\n", len(updates), len(deletes))
		syncClient.SetLastSyncTime(tableName, serverTime)
	}

	// 步骤5: 验证同步时间
	fmt.Println("\n--- 步骤5: 验证同步时间 ---")
	lastSync := syncClient.GetLastSyncTime(tableName)
	fmt.Printf("最后同步时间: %d\n", lastSync)

	fmt.Println("\n========== 数据同步完整工作流测试通过! ==========")
}
