//go:build !windows

package ginx

// requireWindowsBuild 在非 Windows 平台上为空操作。
// 调用方无需关心平台差异，编译器会在非 Windows 构建中内联为空。
func requireWindowsBuild() {}
