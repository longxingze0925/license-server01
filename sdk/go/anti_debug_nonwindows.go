//go:build !windows
// +build !windows

package license

// checkWindowsDebugFlags 非 Windows 平台返回 false
func (ead *EnhancedAntiDebug) checkWindowsDebugFlags() bool {
	return false
}
