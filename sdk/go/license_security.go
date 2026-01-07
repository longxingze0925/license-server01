// Package license 安全模块 - 提供反破解防护
//
// 功能：
// 1. 代码完整性校验
// 2. 反调试检测
// 3. 时间回拨检测
// 4. 多点分散验证
// 5. 环境检测

package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ==================== 反调试检测 ====================

// AntiDebug 反调试检测器
type AntiDebug struct{}

// IsDebuggerPresent 检测是否有调试器
func (a *AntiDebug) IsDebuggerPresent() bool {
	checks := []func() bool{
		a.checkTiming,
		a.checkParentProcess,
		a.checkDebugEnv,
	}

	for _, check := range checks {
		if check() {
			return true
		}
	}
	return false
}

// checkTiming 时间检测
func (a *AntiDebug) checkTiming() bool {
	start := time.Now()
	// 执行一些简单操作
	sum := 0
	for i := 0; i < 100000; i++ {
		sum += i
	}
	_ = sum
	elapsed := time.Since(start)
	// 正常情况下应该很快，调试时会变慢
	return elapsed > 50*time.Millisecond
}

// checkParentProcess 检查父进程
func (a *AntiDebug) checkParentProcess() bool {
	if runtime.GOOS == "windows" {
		return a.checkWindowsDebugger()
	}
	return a.checkUnixDebugger()
}

func (a *AntiDebug) checkWindowsDebugger() bool {
	// 检查常见调试器进程
	debuggers := []string{"ollydbg", "x64dbg", "x32dbg", "ida", "ida64", "windbg", "devenv"}

	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	outputLower := strings.ToLower(string(output))
	for _, debugger := range debuggers {
		if strings.Contains(outputLower, debugger) {
			return true
		}
	}
	return false
}

func (a *AntiDebug) checkUnixDebugger() bool {
	// 检查 /proc/self/status 中的 TracerPid
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/self/status")
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "TracerPid:") {
					pid := strings.TrimSpace(strings.TrimPrefix(line, "TracerPid:"))
					if pid != "0" {
						return true
					}
				}
			}
		}
	}
	return false
}

func (a *AntiDebug) checkDebugEnv() bool {
	debugVars := []string{"GODEBUG", "GOTRACEBACK", "DEBUG", "DELVE_DEBUGGER"}
	for _, v := range debugVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// ==================== 时间回拨检测 ====================

// TimeChecker 时间检测器（带加密存储）
type TimeChecker struct {
	lastCheckTime int64
	cacheFile     string
	encryptKey    []byte
	mu            sync.Mutex
}

// NewTimeChecker 创建时间检测器
func NewTimeChecker(cacheDir string) *TimeChecker {
	tc := &TimeChecker{
		lastCheckTime: time.Now().Unix(),
		cacheFile:     filepath.Join(cacheDir, ".time_check"),
	}
	// 使用机器特征生成加密密钥
	tc.encryptKey = tc.deriveTimeKey()
	tc.loadLastTime()
	return tc
}

// deriveTimeKey 派生时间文件加密密钥
func (tc *TimeChecker) deriveTimeKey() []byte {
	hostname, _ := os.Hostname()
	keyMaterial := hostname + "time_check_key_v2"
	hash := sha256.Sum256([]byte(keyMaterial))
	return hash[:]
}

// encryptTime 加密时间戳
func (tc *TimeChecker) encryptTime(timestamp int64) ([]byte, error) {
	data := fmt.Sprintf("%d:%s", timestamp, tc.generateChecksum(timestamp))

	block, err := aes.NewCipher(tc.encryptKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
	return ciphertext, nil
}

// decryptTime 解密时间戳
func (tc *TimeChecker) decryptTime(encrypted []byte) (int64, error) {
	block, err := aes.NewCipher(tc.encryptKey)
	if err != nil {
		return 0, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return 0, fmt.Errorf("invalid data")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, err
	}

	// 解析时间戳和校验和
	parts := strings.SplitN(string(plaintext), ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid format")
	}

	var timestamp int64
	fmt.Sscanf(parts[0], "%d", &timestamp)

	// 验证校验和
	expectedChecksum := tc.generateChecksum(timestamp)
	if parts[1] != expectedChecksum {
		return 0, fmt.Errorf("checksum mismatch")
	}

	return timestamp, nil
}

// generateChecksum 生成时间戳校验和
func (tc *TimeChecker) generateChecksum(timestamp int64) string {
	data := fmt.Sprintf("%d:time_integrity_v2", timestamp)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func (tc *TimeChecker) loadLastTime() {
	data, err := os.ReadFile(tc.cacheFile)
	if err != nil {
		return
	}

	// 尝试解密
	timestamp, err := tc.decryptTime(data)
	if err != nil {
		// 文件损坏或被篡改，删除
		os.Remove(tc.cacheFile)
		return
	}

	if timestamp > tc.lastCheckTime {
		tc.lastCheckTime = timestamp
	}
}

func (tc *TimeChecker) saveCurrentTime() {
	os.MkdirAll(filepath.Dir(tc.cacheFile), 0755)

	encrypted, err := tc.encryptTime(time.Now().Unix())
	if err != nil {
		return
	}

	os.WriteFile(tc.cacheFile, encrypted, 0600)
}

// Check 检查时间是否被回拨
func (tc *TimeChecker) Check() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	currentTime := time.Now().Unix()

	// 允许 5 分钟的误差
	if currentTime < tc.lastCheckTime-300 {
		return false
	}

	tc.lastCheckTime = currentTime
	tc.saveCurrentTime()
	return true
}

// ==================== 完整性校验 ====================

// IntegrityChecker 完整性校验器
type IntegrityChecker struct {
	executableHash string
}

// NewIntegrityChecker 创建完整性校验器
func NewIntegrityChecker() *IntegrityChecker {
	ic := &IntegrityChecker{}
	ic.executableHash = ic.calculateExecutableHash()
	return ic
}

func (ic *IntegrityChecker) calculateExecutableHash() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(execPath)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Verify 验证完整性
func (ic *IntegrityChecker) Verify() bool {
	currentHash := ic.calculateExecutableHash()
	return currentHash == ic.executableHash
}

// GetChecksum 获取校验和
func (ic *IntegrityChecker) GetChecksum() string {
	if ic.executableHash == "" {
		return ""
	}
	hash := md5.Sum([]byte(ic.executableHash))
	return hex.EncodeToString(hash[:])[:16]
}

// ==================== 环境检测 ====================

// EnvironmentChecker 环境检测器
type EnvironmentChecker struct{}

// IsVirtualMachine 检测是否在虚拟机中
func (ec *EnvironmentChecker) IsVirtualMachine() bool {
	indicators := 0

	// 检查 MAC 地址
	if ec.checkVMMac() {
		indicators++
	}

	// 检查系统信息
	if ec.checkVMSystemInfo() {
		indicators++
	}

	// 检查特定文件
	if ec.checkVMFiles() {
		indicators++
	}

	return indicators >= 2
}

func (ec *EnvironmentChecker) checkVMMac() bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	vmMacPrefixes := []string{
		"00:0c:29", // VMware
		"00:50:56", // VMware
		"08:00:27", // VirtualBox
		"00:1c:42", // Parallels
		"00:16:3e", // Xen
	}

	for _, iface := range interfaces {
		if len(iface.HardwareAddr) >= 3 {
			mac := iface.HardwareAddr.String()
			for _, prefix := range vmMacPrefixes {
				if strings.HasPrefix(strings.ToLower(mac), prefix) {
					return true
				}
			}
		}
	}
	return false
}

func (ec *EnvironmentChecker) checkVMSystemInfo() bool {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "computersystem", "get", "model")
		output, err := cmd.Output()
		if err == nil {
			outputLower := strings.ToLower(string(output))
			vmKeywords := []string{"vmware", "virtualbox", "virtual", "qemu", "xen", "hyperv"}
			for _, kw := range vmKeywords {
				if strings.Contains(outputLower, kw) {
					return true
				}
			}
		}
	} else if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/sys/class/dmi/id/product_name")
		if err == nil {
			content := strings.ToLower(string(data))
			vmKeywords := []string{"vmware", "virtualbox", "qemu", "xen", "kvm"}
			for _, kw := range vmKeywords {
				if strings.Contains(content, kw) {
					return true
				}
			}
		}
	}
	return false
}

func (ec *EnvironmentChecker) checkVMFiles() bool {
	vmFiles := []string{}

	if runtime.GOOS == "windows" {
		vmFiles = []string{
			"C:\\Windows\\System32\\drivers\\vmmouse.sys",
			"C:\\Windows\\System32\\drivers\\vmhgfs.sys",
			"C:\\Windows\\System32\\drivers\\VBoxMouse.sys",
			"C:\\Windows\\System32\\drivers\\VBoxGuest.sys",
		}
	}

	for _, f := range vmFiles {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	return false
}

// ==================== 分散验证器 ====================

// Validator 验证函数类型
type Validator func() bool

// DistributedValidator 分散验证器
type DistributedValidator struct {
	validators   map[string]Validator
	checkResults map[string]bool
	mu           sync.RWMutex
}

// NewDistributedValidator 创建分散验证器
func NewDistributedValidator() *DistributedValidator {
	return &DistributedValidator{
		validators:   make(map[string]Validator),
		checkResults: make(map[string]bool),
	}
}

// Register 注册验证器
func (dv *DistributedValidator) Register(name string, validator Validator) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.validators[name] = validator
}

// ValidateAll 执行所有验证
func (dv *DistributedValidator) ValidateAll() bool {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	dv.checkResults = make(map[string]bool)

	for name, validator := range dv.validators {
		result := validator()
		dv.checkResults[name] = result
		if !result {
			return false
		}
	}
	return true
}

// GetValidationToken 生成验证令牌
func (dv *DistributedValidator) GetValidationToken() string {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	results := ""
	for _, v := range dv.checkResults {
		if v {
			results += "1"
		} else {
			results += "0"
		}
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	combined := results + ":" + timestamp
	hash := md5.Sum([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// ==================== 安全客户端包装器 ====================

// SecureClient 安全客户端包装器
type SecureClient struct {
	client           *Client
	antiDebug        *AntiDebug
	timeChecker      *TimeChecker
	integrityChecker *IntegrityChecker
	envChecker       *EnvironmentChecker
	validator        *DistributedValidator

	checkCount    int
	lastFullCheck int64
	mu            sync.Mutex
}

// NewSecureClient 创建安全客户端
func NewSecureClient(client *Client) *SecureClient {
	sc := &SecureClient{
		client:           client,
		antiDebug:        &AntiDebug{},
		timeChecker:      NewTimeChecker(client.cacheDir),
		integrityChecker: NewIntegrityChecker(),
		envChecker:       &EnvironmentChecker{},
		validator:        NewDistributedValidator(),
	}

	sc.setupValidators()
	return sc
}

func (sc *SecureClient) setupValidators() {
	sc.validator.Register("time", sc.timeChecker.Check)
	sc.validator.Register("license", func() bool {
		return sc.client.licenseInfo != nil
	})
	sc.validator.Register("valid_flag", func() bool {
		if sc.client.licenseInfo == nil {
			return false
		}
		return sc.client.licenseInfo.Valid
	})
}

// IsValid 安全的授权验证
func (sc *SecureClient) IsValid() bool {
	sc.mu.Lock()
	sc.checkCount++
	checkCount := sc.checkCount
	lastFullCheck := sc.lastFullCheck
	sc.mu.Unlock()

	// 每 10 次检查执行一次完整验证
	if checkCount%10 == 0 || time.Now().Unix()-lastFullCheck > 300 {
		if !sc.fullSecurityCheck() {
			return false
		}
		sc.mu.Lock()
		sc.lastFullCheck = time.Now().Unix()
		sc.mu.Unlock()
	}

	return sc.client.IsValid()
}

func (sc *SecureClient) fullSecurityCheck() bool {
	// 1. 反调试检测
	if sc.antiDebug.IsDebuggerPresent() {
		sc.onSecurityViolation("debugger_detected")
		return false
	}

	// 2. 时间检查
	if !sc.timeChecker.Check() {
		sc.onSecurityViolation("time_rollback")
		return false
	}

	// 3. 分散验证
	if !sc.validator.ValidateAll() {
		sc.onSecurityViolation("validation_failed")
		return false
	}

	return true
}

func (sc *SecureClient) onSecurityViolation(reason string) {
	// 静默清除缓存
	sc.client.clearCache()
}

// HasFeature 安全的功能检查
func (sc *SecureClient) HasFeature(feature string) bool {
	if !sc.IsValid() {
		return false
	}
	return sc.client.HasFeature(feature)
}

// GetRemainingDays 获取剩余天数
func (sc *SecureClient) GetRemainingDays() int {
	if !sc.IsValid() {
		return 0
	}
	return sc.client.GetRemainingDays()
}

// Activate 激活
func (sc *SecureClient) Activate(licenseKey string) (*LicenseInfo, error) {
	return sc.client.Activate(licenseKey)
}

// Login 登录
func (sc *SecureClient) Login(email, password string) (*LicenseInfo, error) {
	return sc.client.Login(email, password)
}

// Deactivate 解绑
func (sc *SecureClient) Deactivate() bool {
	return sc.client.Deactivate()
}

// GetSecurityToken 获取安全令牌
func (sc *SecureClient) GetSecurityToken() string {
	parts := []string{
		sc.integrityChecker.GetChecksum(),
		sc.validator.GetValidationToken(),
		fmt.Sprintf("%d", time.Now().Unix()),
	}
	combined := strings.Join(parts, ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])[:32]
}

// Close 关闭
func (sc *SecureClient) Close() {
	sc.client.Close()
}

// ==================== 便捷函数 ====================

// WrapClient 包装客户端，添加安全防护
func WrapClient(client *Client) *SecureClient {
	return NewSecureClient(client)
}

// CheckEnvironment 检查运行环境
func CheckEnvironment() map[string]bool {
	ec := &EnvironmentChecker{}
	ad := &AntiDebug{}

	return map[string]bool{
		"debugger":        ad.IsDebuggerPresent(),
		"virtual_machine": ec.IsVirtualMachine(),
	}
}
