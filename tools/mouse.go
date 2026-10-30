//go:build windows

package tools

import (
	"syscall"
	"unsafe"
)

var pGetCursorPos = user32.MustFindProc("GetCursorPos")

// point 对应 Win32 的 POINT 结构体
type point struct {
	X int32
	Y int32
}

// CursorPos 返回鼠标当前的屏幕坐标。
//
// 坐标原点在虚拟桌面左上角，多显示器时左侧或上方的屏幕会出现负值；
// 坐标受进程 DPI 感知状态影响，per-monitor aware 的进程拿到的是真实物理像素。
func CursorPos() (x int32, y int32, err error) {
	var pt point
	r, _, e := syscall.SyscallN(pGetCursorPos.Addr(), uintptr(unsafe.Pointer(&pt)))
	if r == 0 {
		return 0, 0, osErr("GetCursorPos", e)
	}
	return pt.X, pt.Y, nil
}
