//go:build windows

package ginx

import (
	"fmt"
	"syscall"
	"unsafe"
)

// minWindowsBuild 是支持 AF_UNIX 的最低 Windows build 号（1803）。
const minWindowsBuild = 17134

// getWindowsBuild 通过 RtlGetNtVersionNumbers 获取准确的 Windows build 号。
//
// 与 GetVersionEx 不同，RtlGetNtVersionNumbers 不受应用程序兼容性清单（manifest）影响，
// 因此总能返回真实的 OS 版本号。
func getWindowsBuild() int {
	ntdll := syscall.NewLazyDLL("ntdll.dll")
	proc := ntdll.NewProc("RtlGetNtVersionNumbers")

	var major, minor, build uint32
	proc.Call(
		uintptr(unsafe.Pointer(&major)),
		uintptr(unsafe.Pointer(&minor)),
		uintptr(unsafe.Pointer(&build)),
	)

	return int(build & 0xFFFF)
}

// requireWindowsBuild 检查当前 Windows 的 build 号是否满足 AF_UNIX 最低要求。
//
// 若低于 Windows 10 build 1803 (10.0.17134)，则立即 panic，因为该版本之前的
// Windows 内核不提供 AF_UNIX 地址族支持。
func requireWindowsBuild() {
	build := getWindowsBuild()
	if build < minWindowsBuild {
		panic(fmt.Sprintf(
			"ginx：当前 Windows 系统版本过低（build %d），"+
				"不支持 Unix Socket 监听。"+
				"需要 Windows 10 build 1803 (10.0.17134) 或更高版本。"+
				"请升级操作系统后重试。",
			build,
		))
	}
}
