package license

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HotUpdateInfo 热更新信息
type HotUpdateInfo struct {
	HasUpdate     bool   `json:"has_update"`
	ID            string `json:"id"`
	FromVersion   string `json:"from_version"`
	ToVersion     string `json:"to_version"`
	PatchType     string `json:"patch_type"`
	UpdateType    string `json:"update_type"` // patch 或 full
	DownloadURL   string `json:"download_url"`
	FileSize      int64  `json:"file_size"`
	FileHash      string `json:"file_hash"`
	Changelog     string `json:"changelog"`
	ForceUpdate   bool   `json:"force_update"`
	MinAppVersion string `json:"min_app_version"`
}

// HotUpdateStatus 更新状态
type HotUpdateStatus string

const (
	HotUpdateStatusPending     HotUpdateStatus = "pending"
	HotUpdateStatusDownloading HotUpdateStatus = "downloading"
	HotUpdateStatusInstalling  HotUpdateStatus = "installing"
	HotUpdateStatusSuccess     HotUpdateStatus = "success"
	HotUpdateStatusFailed      HotUpdateStatus = "failed"
	HotUpdateStatusRollback    HotUpdateStatus = "rollback"
)

// HotUpdateCallback 热更新回调
type HotUpdateCallback func(status HotUpdateStatus, progress float64, err error)

// HotUpdateManager 热更新管理器
type HotUpdateManager struct {
	client         *Client
	currentVersion string
	updateDir      string
	backupDir      string
	callback       HotUpdateCallback
	autoCheck      bool
	checkInterval  time.Duration

	mu            sync.RWMutex
	latestUpdate  *HotUpdateInfo
	isUpdating    bool
	stopAutoCheck chan struct{}
}

// HotUpdateOption 热更新配置选项
type HotUpdateOption func(*HotUpdateManager)

// WithUpdateDir 设置更新目录
func WithUpdateDir(dir string) HotUpdateOption {
	return func(m *HotUpdateManager) {
		m.updateDir = dir
	}
}

// WithBackupDir 设置备份目录
func WithBackupDir(dir string) HotUpdateOption {
	return func(m *HotUpdateManager) {
		m.backupDir = dir
	}
}

// WithAutoCheck 设置自动检查更新
func WithAutoCheck(enabled bool, interval time.Duration) HotUpdateOption {
	return func(m *HotUpdateManager) {
		m.autoCheck = enabled
		m.checkInterval = interval
	}
}

// WithUpdateCallback 设置更新回调
func WithUpdateCallback(callback HotUpdateCallback) HotUpdateOption {
	return func(m *HotUpdateManager) {
		m.callback = callback
	}
}

// NewHotUpdateManager 创建热更新管理器
func NewHotUpdateManager(client *Client, currentVersion string, opts ...HotUpdateOption) *HotUpdateManager {
	homeDir, _ := os.UserHomeDir()
	m := &HotUpdateManager{
		client:         client,
		currentVersion: currentVersion,
		updateDir:      filepath.Join(homeDir, ".app_updates"),
		backupDir:      filepath.Join(homeDir, ".app_backups"),
		checkInterval:  time.Hour,
		stopAutoCheck:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	// 确保目录存在
	os.MkdirAll(m.updateDir, 0755)
	os.MkdirAll(m.backupDir, 0755)

	return m
}

// CheckUpdate 检查更新
func (m *HotUpdateManager) CheckUpdate() (*HotUpdateInfo, error) {
	url := fmt.Sprintf("%s/api/client/hotupdate/check?app_key=%s&version=%s&machine_id=%s",
		m.client.GetServerURL(), m.client.GetAppKey(), m.currentVersion, m.client.GetMachineID())

	resp, err := m.client.GetHTTPClient().Get(url)
	if err != nil {
		return nil, fmt.Errorf("检查更新失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Data    HotUpdateInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	m.mu.Lock()
	m.latestUpdate = &result.Data
	m.mu.Unlock()

	return &result.Data, nil
}

// GetLatestUpdate 获取最新的更新信息（从缓存）
func (m *HotUpdateManager) GetLatestUpdate() *HotUpdateInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latestUpdate
}

// DownloadUpdate 下载更新
func (m *HotUpdateManager) DownloadUpdate(info *HotUpdateInfo) (string, error) {
	if info == nil || !info.HasUpdate {
		return "", fmt.Errorf("没有可用的更新")
	}

	m.mu.Lock()
	if m.isUpdating {
		m.mu.Unlock()
		return "", fmt.Errorf("正在更新中")
	}
	m.isUpdating = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.isUpdating = false
		m.mu.Unlock()
	}()

	// 上报下载状态
	m.reportStatus(info.ID, HotUpdateStatusDownloading, "")
	m.notifyCallback(HotUpdateStatusDownloading, 0, nil)

	// 构建下载URL
	downloadURL := info.DownloadURL
	// 如果不是完整URL，则拼接服务器地址
	if !strings.HasPrefix(downloadURL, "http://") && !strings.HasPrefix(downloadURL, "https://") {
		downloadURL = m.client.GetServerURL() + info.DownloadURL
	}

	// 创建HTTP请求
	resp, err := m.client.GetHTTPClient().Get(downloadURL)
	if err != nil {
		m.reportStatus(info.ID, HotUpdateStatusFailed, err.Error())
		m.notifyCallback(HotUpdateStatusFailed, 0, err)
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
		m.reportStatus(info.ID, HotUpdateStatusFailed, err.Error())
		m.notifyCallback(HotUpdateStatusFailed, 0, err)
		return "", err
	}

	// 创建临时文件
	filename := fmt.Sprintf("update_%s_to_%s.zip", info.FromVersion, info.ToVersion)
	filePath := filepath.Join(m.updateDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		m.reportStatus(info.ID, HotUpdateStatusFailed, err.Error())
		m.notifyCallback(HotUpdateStatusFailed, 0, err)
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 下载并计算进度
	var downloaded int64
	buf := make([]byte, 32*1024)
	hash := sha256.New()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			file.Write(buf[:n])
			hash.Write(buf[:n])
			downloaded += int64(n)

			if info.FileSize > 0 {
				progress := float64(downloaded) / float64(info.FileSize)
				m.notifyCallback(HotUpdateStatusDownloading, progress, nil)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			m.reportStatus(info.ID, HotUpdateStatusFailed, err.Error())
			m.notifyCallback(HotUpdateStatusFailed, 0, err)
			return "", fmt.Errorf("下载失败: %w", err)
		}
	}

	// 验证哈希
	fileHash := hex.EncodeToString(hash.Sum(nil))
	if info.FileHash != "" && fileHash != info.FileHash {
		os.Remove(filePath)
		err := fmt.Errorf("文件校验失败")
		m.reportStatus(info.ID, HotUpdateStatusFailed, err.Error())
		m.notifyCallback(HotUpdateStatusFailed, 0, err)
		return "", err
	}

	m.notifyCallback(HotUpdateStatusDownloading, 1, nil)
	return filePath, nil
}

// ApplyUpdate 应用更新
func (m *HotUpdateManager) ApplyUpdate(info *HotUpdateInfo, updateFile string, targetDir string) error {
	if info == nil {
		return fmt.Errorf("更新信息为空")
	}

	// 上报安装状态
	m.reportStatus(info.ID, HotUpdateStatusInstalling, "")
	m.notifyCallback(HotUpdateStatusInstalling, 0, nil)

	// 备份当前版本
	backupPath := filepath.Join(m.backupDir, fmt.Sprintf("backup_%s_%d", m.currentVersion, time.Now().Unix()))
	if err := m.backupCurrentVersion(targetDir, backupPath); err != nil {
		m.reportStatus(info.ID, HotUpdateStatusFailed, "备份失败: "+err.Error())
		m.notifyCallback(HotUpdateStatusFailed, 0, err)
		return fmt.Errorf("备份失败: %w", err)
	}

	// 解压更新包
	if err := m.extractUpdate(updateFile, targetDir); err != nil {
		// 回滚
		m.rollback(backupPath, targetDir)
		m.reportStatus(info.ID, HotUpdateStatusFailed, "解压失败: "+err.Error())
		m.notifyCallback(HotUpdateStatusFailed, 0, err)
		return fmt.Errorf("解压失败: %w", err)
	}

	// 更新成功
	m.currentVersion = info.ToVersion
	m.reportStatus(info.ID, HotUpdateStatusSuccess, "")
	m.notifyCallback(HotUpdateStatusSuccess, 1, nil)

	// 清理下载的更新包
	os.Remove(updateFile)

	// 清理旧备份（保留最近3个）
	m.cleanOldBackups(3)

	return nil
}

// Rollback 回滚到上一个版本
func (m *HotUpdateManager) Rollback(targetDir string) error {
	// 查找最新的备份
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("没有可用的备份")
	}

	// 获取最新的备份
	var latestBackup string
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err == nil && info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestBackup = filepath.Join(m.backupDir, entry.Name())
			}
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("没有可用的备份")
	}

	return m.rollback(latestBackup, targetDir)
}

// StartAutoCheck 启动自动检查更新
func (m *HotUpdateManager) StartAutoCheck() {
	if !m.autoCheck {
		return
	}

	go func() {
		ticker := time.NewTicker(m.checkInterval)
		defer ticker.Stop()

		// 立即检查一次
		m.CheckUpdate()

		for {
			select {
			case <-ticker.C:
				m.CheckUpdate()
			case <-m.stopAutoCheck:
				return
			}
		}
	}()
}

// StopAutoCheck 停止自动检查更新
func (m *HotUpdateManager) StopAutoCheck() {
	select {
	case m.stopAutoCheck <- struct{}{}:
	default:
	}
}

// GetUpdateHistory 获取更新历史
func (m *HotUpdateManager) GetUpdateHistory() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/client/hotupdate/history?app_key=%s&machine_id=%s",
		m.client.GetServerURL(), m.client.GetAppKey(), m.client.GetMachineID())

	resp, err := m.client.GetHTTPClient().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int                      `json:"code"`
		Message string                   `json:"message"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Message)
	}

	return result.Data, nil
}

// IsUpdating 是否正在更新
func (m *HotUpdateManager) IsUpdating() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isUpdating
}

// GetCurrentVersion 获取当前版本
func (m *HotUpdateManager) GetCurrentVersion() string {
	return m.currentVersion
}

// SetCurrentVersion 设置当前版本
func (m *HotUpdateManager) SetCurrentVersion(version string) {
	m.currentVersion = version
}

// 内部方法

func (m *HotUpdateManager) reportStatus(hotUpdateID string, status HotUpdateStatus, errorMsg string) {
	data := map[string]interface{}{
		"app_key":       m.client.GetAppKey(),
		"hot_update_id": hotUpdateID,
		"machine_id":    m.client.GetMachineID(),
		"from_version":  m.currentVersion,
		"status":        string(status),
	}
	if errorMsg != "" {
		data["error_message"] = errorMsg
	}

	go m.client.request("POST", "/hotupdate/report", data)
}

func (m *HotUpdateManager) notifyCallback(status HotUpdateStatus, progress float64, err error) {
	if m.callback != nil {
		m.callback(status, progress, err)
	}
}

func (m *HotUpdateManager) backupCurrentVersion(sourceDir, backupPath string) error {
	return copyDir(sourceDir, backupPath)
}

func (m *HotUpdateManager) extractUpdate(zipFile, targetDir string) error {
	// 简单实现：直接复制文件
	// 实际应用中应该使用 archive/zip 解压
	return unzip(zipFile, targetDir)
}

func (m *HotUpdateManager) rollback(backupPath, targetDir string) error {
	// 删除当前目录内容
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		os.RemoveAll(filepath.Join(targetDir, entry.Name()))
	}

	// 从备份恢复
	return copyDir(backupPath, targetDir)
}

func (m *HotUpdateManager) cleanOldBackups(keep int) {
	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return
	}

	if len(entries) <= keep {
		return
	}

	// 按时间排序，删除旧的
	type backupInfo struct {
		path    string
		modTime time.Time
	}
	var backups []backupInfo

	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				backups = append(backups, backupInfo{
					path:    filepath.Join(m.backupDir, entry.Name()),
					modTime: info.ModTime(),
				})
			}
		}
	}

	// 简单排序
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].modTime.After(backups[j].modTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// 删除旧的备份
	for i := 0; i < len(backups)-keep; i++ {
		os.RemoveAll(backups[i].path)
	}
}

// 辅助函数

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func unzip(src, dst string) error {
	// 使用 archive/zip 解压
	// 这里提供一个简化实现，实际使用时需要完整实现

	// 如果是单个文件，直接复制
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		// 检查是否是 zip 文件
		if filepath.Ext(src) == ".zip" {
			return unzipFile(src, dst)
		}
		// 否则直接复制
		return copyFile(src, dst)
	}

	return copyDir(src, dst)
}

func unzipFile(src, dst string) error {
	// 打开 zip 文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开 zip 文件失败: %w", err)
	}
	defer r.Close()

	// 创建目标目录
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 遍历 zip 文件中的所有文件
	for _, f := range r.File {
		// 构建目标路径
		fpath := filepath.Join(dst, f.Name)

		// 安全检查：防止 zip slip 攻击（路径遍历漏洞）
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}

		// 解压文件
		if err := extractZipFile(f, fpath); err != nil {
			return err
		}
	}

	return nil
}

func extractZipFile(f *zip.File, destPath string) error {
	// 打开 zip 中的文件
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("打开 zip 内文件失败: %w", err)
	}
	defer rc.Close()

	// 创建目标文件
	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer outFile.Close()

	// 复制内容
	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}
