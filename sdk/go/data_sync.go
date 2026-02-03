// Package license 数据同步功能
// 支持将本地 SQLite 数据库的表数据同步到云端服务器
package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// SyncRecord 同步记录
type SyncRecord struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	Version   int64                  `json:"version"`
	IsDeleted bool                   `json:"is_deleted"`
	UpdatedAt int64                  `json:"updated_at"`
}

// SyncResult 同步结果
type SyncResult struct {
	RecordID      string `json:"record_id"`
	Status        string `json:"status"` // success, conflict, error
	Version       int64  `json:"version"`
	ServerVersion int64  `json:"server_version,omitempty"`
}

// TableInfo 表信息
type TableInfo struct {
	TableName   string `json:"table_name"`
	RecordCount int64  `json:"record_count"`
	LastUpdated string `json:"last_updated"`
}

// 数据类型常量
const (
	DataTypeScripts            = "scripts"              // 话术管理
	DataTypeDanmakuGroups      = "danmaku_groups"       // 互动规则
	DataTypeAIConfig           = "ai_config"            // AI配置
	DataTypeRandomWordAIConfig = "random_word_ai_config" // 随机词AI配置
)

// BackupData 备份数据
type BackupData struct {
	ID         string `json:"id"`
	DataType   string `json:"data_type"`
	DataJSON   string `json:"data_json"`
	Version    int    `json:"version"`
	DeviceName string `json:"device_name"`
	MachineID  string `json:"machine_id"`
	IsCurrent  bool   `json:"is_current"`
	DataSize   int64  `json:"data_size"`
	ItemCount  int    `json:"item_count"`
	Checksum   string `json:"checksum"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// DataSyncClient 数据同步客户端
type DataSyncClient struct {
	client       *Client
	lastSyncTime map[string]int64 // 每个表的最后同步时间
}

// NewDataSyncClient 创建数据同步客户端
func (c *Client) NewDataSyncClient() *DataSyncClient {
	return &DataSyncClient{
		client:       c,
		lastSyncTime: make(map[string]int64),
	}
}

// GetTableList 获取服务器上的所有表名
func (d *DataSyncClient) GetTableList() ([]TableInfo, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/tables?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    []TableInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}
	return result.Data, nil
}

// PullTable 从服务器拉取指定表的数据
// tableName: 表名
// since: 增量同步时间戳（0表示全量）
func (d *DataSyncClient) PullTable(tableName string, since int64) ([]SyncRecord, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	params.Set("table", tableName)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/table?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Table      string       `json:"table"`
			Records    []SyncRecord `json:"records"`
			Count      int          `json:"count"`
			ServerTime int64        `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	// 更新最后同步时间
	d.lastSyncTime[tableName] = result.Data.ServerTime

	return result.Data.Records, result.Data.ServerTime, nil
}

// PullAllTables 从服务器拉取所有表的数据
// since: 增量同步时间戳（0表示全量）
func (d *DataSyncClient) PullAllTables(since int64) (map[string][]SyncRecord, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/tables/all?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Tables     map[string][]SyncRecord `json:"tables"`
			ServerTime int64                   `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Tables, result.Data.ServerTime, nil
}

// PushRecord 推送单条记录到服务器
func (d *DataSyncClient) PushRecord(tableName, recordID string, data map[string]interface{}, version int64) (*SyncResult, error) {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"table":      tableName,
		"record_id":  recordID,
		"data":       data,
		"version":    version,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/table",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status        string `json:"status"`
			Version       int64  `json:"version"`
			ServerVersion int64  `json:"server_version,omitempty"`
			ServerData    string `json:"server_data,omitempty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &SyncResult{
		RecordID:      recordID,
		Status:        result.Data.Status,
		Version:       result.Data.Version,
		ServerVersion: result.Data.ServerVersion,
	}, nil
}

// PushRecordBatch 批量推送记录到服务器
type PushRecordItem struct {
	RecordID string                 `json:"record_id"`
	Data     map[string]interface{} `json:"data"`
	Version  int64                  `json:"version"`
	Deleted  bool                   `json:"deleted"`
}

func (d *DataSyncClient) PushRecordBatch(tableName string, records []PushRecordItem) ([]SyncResult, error) {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"table":      tableName,
		"records":    records,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/table/batch",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Table      string       `json:"table"`
			Results    []SyncResult `json:"results"`
			Count      int          `json:"count"`
			ServerTime int64        `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Results, nil
}

// DeleteRecord 删除服务器上的记录
func (d *DataSyncClient) DeleteRecord(tableName, recordID string) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"table":      tableName,
		"record_id":  recordID,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("DELETE", d.client.serverURL+"/api/client/sync/table", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// GetLastSyncTime 获取指定表的最后同步时间
func (d *DataSyncClient) GetLastSyncTime(tableName string) int64 {
	return d.lastSyncTime[tableName]
}

// SetLastSyncTime 设置指定表的最后同步时间
func (d *DataSyncClient) SetLastSyncTime(tableName string, t int64) {
	d.lastSyncTime[tableName] = t
}

// ==================== 便捷同步方法 ====================

// SyncTableToServer 将本地表数据同步到服务器
// 传入表名和记录列表，自动处理推送
func (d *DataSyncClient) SyncTableToServer(tableName string, records []map[string]interface{}, idField string) ([]SyncResult, error) {
	items := make([]PushRecordItem, 0, len(records))
	for _, record := range records {
		recordID := ""
		if id, ok := record[idField]; ok {
			recordID = fmt.Sprintf("%v", id)
		}
		if recordID == "" {
			continue
		}
		items = append(items, PushRecordItem{
			RecordID: recordID,
			Data:     record,
			Version:  0, // 不检查版本冲突
			Deleted:  false,
		})
	}

	if len(items) == 0 {
		return nil, nil
	}

	return d.PushRecordBatch(tableName, items)
}

// SyncTableFromServer 从服务器同步表数据到本地
// 返回需要更新/插入的记录和需要删除的记录ID
func (d *DataSyncClient) SyncTableFromServer(tableName string, since int64) (updates []SyncRecord, deletes []string, serverTime int64, err error) {
	records, serverTime, err := d.PullTable(tableName, since)
	if err != nil {
		return nil, nil, 0, err
	}

	updates = make([]SyncRecord, 0)
	deletes = make([]string, 0)

	for _, r := range records {
		if r.IsDeleted {
			deletes = append(deletes, r.ID)
		} else {
			updates = append(updates, r)
		}
	}

	return updates, deletes, serverTime, nil
}

// ==================== SQLite 辅助方法 ====================

// SQLiteRecord 从 SQLite 查询结果转换的记录
type SQLiteRecord struct {
	ID   interface{}            `json:"id"`
	Data map[string]interface{} `json:"data"`
}

// ConvertSQLiteRows 将 SQLite 查询结果转换为同步记录格式
// 需要传入列名列表和行数据
func ConvertSQLiteRows(columns []string, rows [][]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		record := make(map[string]interface{})
		for i, col := range columns {
			if i < len(row) {
				record[col] = row[i]
			}
		}
		result = append(result, record)
	}
	return result
}

// ApplySyncRecordToMap 将同步记录转换为 map（用于插入/更新本地数据库）
func ApplySyncRecordToMap(record SyncRecord) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range record.Data {
		result[k] = v
	}
	return result
}

// ==================== 自动同步管理器 ====================

// AutoSyncManager 自动同步管理器
type AutoSyncManager struct {
	syncClient   *DataSyncClient
	tables       []string
	interval     time.Duration
	stopChan     chan struct{}
	onPull       func(tableName string, records []SyncRecord, deletes []string) error
	onConflict   func(tableName string, result SyncResult) error
	lastSyncTime map[string]int64
}

// NewAutoSyncManager 创建自动同步管理器
func (d *DataSyncClient) NewAutoSyncManager(tables []string, interval time.Duration) *AutoSyncManager {
	return &AutoSyncManager{
		syncClient:   d,
		tables:       tables,
		interval:     interval,
		stopChan:     make(chan struct{}),
		lastSyncTime: make(map[string]int64),
	}
}

// OnPull 设置拉取数据回调
func (m *AutoSyncManager) OnPull(fn func(tableName string, records []SyncRecord, deletes []string) error) {
	m.onPull = fn
}

// OnConflict 设置冲突处理回调
func (m *AutoSyncManager) OnConflict(fn func(tableName string, result SyncResult) error) {
	m.onConflict = fn
}

// Start 启动自动同步
func (m *AutoSyncManager) Start() {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		// 立即执行一次同步
		m.syncAll()

		for {
			select {
			case <-ticker.C:
				m.syncAll()
			case <-m.stopChan:
				return
			}
		}
	}()
}

// Stop 停止自动同步
func (m *AutoSyncManager) Stop() {
	close(m.stopChan)
}

// syncAll 同步所有表
func (m *AutoSyncManager) syncAll() {
	for _, tableName := range m.tables {
		since := m.lastSyncTime[tableName]
		updates, deletes, serverTime, err := m.syncClient.SyncTableFromServer(tableName, since)
		if err != nil {
			continue
		}

		if m.onPull != nil && (len(updates) > 0 || len(deletes) > 0) {
			if err := m.onPull(tableName, updates, deletes); err != nil {
				continue
			}
		}

		m.lastSyncTime[tableName] = serverTime
	}
}

// SyncNow 立即同步
func (m *AutoSyncManager) SyncNow() {
	m.syncAll()
}

// ==================== 高级数据同步功能 ====================

// SyncChange 同步变更记录
type SyncChange struct {
	ID         string                 `json:"id"`
	Table      string                 `json:"table"`
	RecordID   string                 `json:"record_id"`
	Operation  string                 `json:"operation"` // insert, update, delete
	Data       map[string]interface{} `json:"data"`
	Version    int64                  `json:"version"`
	ChangeTime int64                  `json:"change_time"`
}

// SyncStatus 同步状态
type SyncStatus struct {
	LastSyncTime   int64            `json:"last_sync_time"`
	PendingChanges int              `json:"pending_changes"`
	TableStatus    map[string]int64 `json:"table_status"` // 每个表的最后同步时间
	ServerTime     int64            `json:"server_time"`
}

// ConflictInfo 冲突信息
type ConflictInfo struct {
	Table         string                 `json:"table"`
	RecordID      string                 `json:"record_id"`
	LocalData     map[string]interface{} `json:"local_data"`
	ServerData    map[string]interface{} `json:"server_data"`
	LocalVersion  int64                  `json:"local_version"`
	ServerVersion int64                  `json:"server_version"`
	ConflictTime  int64                  `json:"conflict_time"`
}

// ConflictResolution 冲突解决策略
type ConflictResolution string

const (
	// UseLocal 使用本地数据
	UseLocal ConflictResolution = "use_local"
	// UseServer 使用服务器数据
	UseServer ConflictResolution = "use_server"
	// Merge 合并数据
	Merge ConflictResolution = "merge"
)

// PushChanges 推送客户端变更到服务端（Push）
// changes: 变更列表
func (d *DataSyncClient) PushChanges(changes []SyncChange) ([]SyncResult, error) {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"changes":    changes,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/push",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Results    []SyncResult `json:"results"`
			ServerTime int64        `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Results, nil
}

// GetChanges 获取服务端变更（Pull）
// since: 从指定时间戳开始获取变更
// tables: 指定要获取的表（为空则获取所有表）
func (d *DataSyncClient) GetChanges(since int64, tables []string) ([]SyncChange, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}
	for _, table := range tables {
		params.Add("tables", table)
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/changes?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Changes    []SyncChange `json:"changes"`
			ServerTime int64        `json:"server_time"`
			HasMore    bool         `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Changes, result.Data.ServerTime, nil
}

// GetSyncStatus 获取同步状态
func (d *DataSyncClient) GetSyncStatus() (*SyncStatus, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/status?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    SyncStatus `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &result.Data, nil
}

// ResolveConflict 解决数据冲突
// resolution: 解决策略 (use_local, use_server, merge)
// mergedData: 当策略为 merge 时，提供合并后的数据
func (d *DataSyncClient) ResolveConflict(tableName, recordID string, resolution ConflictResolution, mergedData map[string]interface{}) (*SyncResult, error) {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"table":      tableName,
		"record_id":  recordID,
		"resolution": string(resolution),
	}
	if mergedData != nil {
		reqBody["merged_data"] = mergedData
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/conflict/resolve",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return &SyncResult{
		RecordID: recordID,
		Status:   result.Data.Status,
		Version:  result.Data.Version,
	}, nil
}

// ==================== 分类数据同步功能 ====================

// ConfigData 配置数据
type ConfigData struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	UpdatedAt int64       `json:"updated_at"`
}

// WorkflowData 工作流数据
type WorkflowData struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
	UpdatedAt int64                  `json:"updated_at"`
}

// MaterialData 素材数据
type MaterialData struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Tags      string `json:"tags"`
	UpdatedAt int64  `json:"updated_at"`
}

// PostData 帖子数据
type PostData struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	GroupID   string `json:"group_id"`
	UpdatedAt int64  `json:"updated_at"`
}

// CommentScriptData 评论话术数据
type CommentScriptData struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	UpdatedAt int64  `json:"updated_at"`
}

// VoiceConfigData TTS声音配置数据
type VoiceConfigData struct {
	ID           int64   `json:"id"`
	Role         string  `json:"role"`           // 角色标识
	Name         string  `json:"name"`           // 配置名称
	GPTPath      string  `json:"gpt_path"`       // GPT模型路径
	SoVITSPath   string  `json:"sovits_path"`    // SoVITS模型路径
	RefAudioPath string  `json:"ref_audio_path"` // 参考音频路径
	RefText      string  `json:"ref_text"`       // 参考文本
	Language     string  `json:"language"`       // 语言
	SpeedFactor  float64 `json:"speed_factor"`   // 语速因子
	TTSVersion   int     `json:"tts_version"`    // TTS版本: 1=v1, 2=v2, 3=v3, 4=v4, 5=v2Pro, 6=v2ProPlus
	Enabled      bool    `json:"enabled"`        // 是否启用
	UpdatedAt    int64   `json:"updated_at"`
}

// GetConfigs 获取配置数据
func (d *DataSyncClient) GetConfigs(since int64) ([]ConfigData, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/configs?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Configs    []ConfigData `json:"configs"`
			ServerTime int64        `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Configs, result.Data.ServerTime, nil
}

// SaveConfigs 保存配置数据
func (d *DataSyncClient) SaveConfigs(configs []ConfigData) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"configs":    configs,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/configs",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// GetWorkflows 获取工作流数据
func (d *DataSyncClient) GetWorkflows(since int64) ([]WorkflowData, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/workflows?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Workflows  []WorkflowData `json:"workflows"`
			ServerTime int64          `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Workflows, result.Data.ServerTime, nil
}

// SaveWorkflows 保存工作流数据
func (d *DataSyncClient) SaveWorkflows(workflows []WorkflowData) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"workflows":  workflows,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/workflows",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// DeleteWorkflow 删除工作流
func (d *DataSyncClient) DeleteWorkflow(workflowID string) error {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)

	req, _ := http.NewRequest("DELETE",
		d.client.serverURL+"/api/client/sync/workflows/"+workflowID+"?"+params.Encode(), nil)

	resp, err := d.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// GetMaterials 获取素材数据
func (d *DataSyncClient) GetMaterials(since int64) ([]MaterialData, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/materials?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Materials  []MaterialData `json:"materials"`
			ServerTime int64          `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Materials, result.Data.ServerTime, nil
}

// SaveMaterials 保存素材数据
func (d *DataSyncClient) SaveMaterials(materials []MaterialData) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"materials":  materials,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/materials/batch",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// GetPosts 获取帖子数据
func (d *DataSyncClient) GetPosts(since int64, groupID string) ([]PostData, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}
	if groupID != "" {
		params.Set("group_id", groupID)
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/posts?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Posts      []PostData `json:"posts"`
			ServerTime int64      `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Posts, result.Data.ServerTime, nil
}

// SavePosts 批量保存帖子数据
func (d *DataSyncClient) SavePosts(posts []PostData) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"posts":      posts,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/posts/batch",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// UpdatePostStatus 更新帖子状态
func (d *DataSyncClient) UpdatePostStatus(postID, status string) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"status":     status,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT",
		d.client.serverURL+"/api/client/sync/posts/"+postID+"/status",
		bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// PostGroup 帖子分组
type PostGroup struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetPostGroups 获取帖子分组
func (d *DataSyncClient) GetPostGroups() ([]PostGroup, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/posts/groups?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Groups []PostGroup `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Groups, nil
}

// GetCommentScripts 获取评论话术
func (d *DataSyncClient) GetCommentScripts(since int64, category string) ([]CommentScriptData, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}
	if category != "" {
		params.Set("category", category)
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/comment-scripts?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Scripts    []CommentScriptData `json:"scripts"`
			ServerTime int64               `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.Scripts, result.Data.ServerTime, nil
}

// SaveCommentScripts 批量保存评论话术
func (d *DataSyncClient) SaveCommentScripts(scripts []CommentScriptData) error {
	reqBody := map[string]interface{}{
		"app_key":    d.client.appKey,
		"machine_id": d.client.machineID,
		"scripts":    scripts,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/comment-scripts/batch",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// ==================== TTS声音配置同步 ====================

// GetVoiceConfigs 获取TTS声音配置
func (d *DataSyncClient) GetVoiceConfigs(since int64) ([]VoiceConfigData, int64, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	if since > 0 {
		params.Set("since", strconv.FormatInt(since, 10))
	}

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/sync/voice-configs?" + params.Encode())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			VoiceConfigs []VoiceConfigData `json:"voice_configs"`
			ServerTime   int64             `json:"server_time"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, err
	}
	if result.Code != 0 {
		return nil, 0, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data.VoiceConfigs, result.Data.ServerTime, nil
}

// SaveVoiceConfigs 批量保存TTS声音配置
func (d *DataSyncClient) SaveVoiceConfigs(configs []VoiceConfigData) error {
	reqBody := map[string]interface{}{
		"app_key":       d.client.appKey,
		"machine_id":    d.client.machineID,
		"voice_configs": configs,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/voice-configs/batch",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// SaveVoiceConfig 保存单个TTS声音配置
func (d *DataSyncClient) SaveVoiceConfig(config VoiceConfigData) error {
	reqBody := map[string]interface{}{
		"app_key":      d.client.appKey,
		"machine_id":   d.client.machineID,
		"voice_config": config,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/sync/voice-configs",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// DeleteVoiceConfig 删除TTS声音配置
func (d *DataSyncClient) DeleteVoiceConfig(voiceConfigID int64) error {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)

	req, _ := http.NewRequest("DELETE",
		d.client.serverURL+"/api/client/sync/voice-configs/"+strconv.FormatInt(voiceConfigID, 10)+"?"+params.Encode(), nil)

	resp, err := d.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// ==================== 数据备份和同步功能 ====================

// PushBackup 推送备份数据到服务器
// dataType: 数据类型（scripts/danmaku_groups/ai_config/random_word_ai_config）
// dataJSON: JSON格式的数据
// deviceName: 设备名称（可选）
// itemCount: 条目数量（可选）
func (d *DataSyncClient) PushBackup(dataType, dataJSON, deviceName string, itemCount int) error {
	reqBody := map[string]interface{}{
		"app_key":     d.client.appKey,
		"machine_id":  d.client.machineID,
		"data_type":   dataType,
		"data_json":   dataJSON,
		"device_name": deviceName,
		"item_count":  itemCount,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := d.client.httpClient.Post(
		d.client.serverURL+"/api/client/backup/push",
		"application/json",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("API error: %s", result.Message)
	}

	return nil
}

// PullBackup 从服务器拉取指定类型的备份数据
// dataType: 数据类型（scripts/danmaku_groups/ai_config/random_word_ai_config）
// 返回备份数据列表（按版本降序排列，第一个为当前版本）
func (d *DataSyncClient) PullBackup(dataType string) ([]BackupData, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)
	params.Set("data_type", dataType)

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/backup/pull?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    []BackupData `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data, nil
}

// PullAllBackups 从服务器拉取所有类型的备份数据
// 返回按数据类型分组的备份数据映射
func (d *DataSyncClient) PullAllBackups() (map[string][]BackupData, error) {
	params := url.Values{}
	params.Set("app_key", d.client.appKey)
	params.Set("machine_id", d.client.machineID)

	resp, err := d.client.httpClient.Get(d.client.serverURL + "/api/client/backup/pull?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    []BackupData `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	// 按数据类型分组
	backupMap := make(map[string][]BackupData)
	for _, backup := range result.Data {
		backupMap[backup.DataType] = append(backupMap[backup.DataType], backup)
	}

	return backupMap, nil
}
