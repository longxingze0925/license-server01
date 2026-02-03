//go:build windows
// +build windows

package license

import (
	"syscall"
	"unsafe"
)

// checkWindowsDebugFlags Windows 调试标志检测
func (ead *EnhancedAntiDebug) checkWindowsDebugFlags() bool {
	// 检查 IsDebuggerPresent API
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	isDebuggerPresent := kernel32.NewProc("IsDebuggerPresent")

	ret, _, _ := isDebuggerPresent.Call()
	if ret != 0 {
		return true
	}

	// 检查 CheckRemoteDebuggerPresent
	checkRemoteDebuggerPresent := kernel32.NewProc("CheckRemoteDebuggerPresent")
	var isRemoteDebugger int32
	checkRemoteDebuggerPresent.Call(
		uintptr(syscall.Handle(^uintptr(0))), // GetCurrentProcess()
		uintptr(unsafe.Pointer(&isRemoteDebugger)),
	)
	if isRemoteDebugger != 0 {
		return true
	}

	return false
}
