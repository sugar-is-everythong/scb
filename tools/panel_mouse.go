//go:build windows

package tools

import (
	"syscall"
	"unsafe"
)

const (
	whMouseLL     = 14
	wmLButtonDown = 0x0201
)

// msLLHookStruct 对应 Win32 的 MSLLHOOKSTRUCT
type msLLHookStruct struct {
	Pt          point
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// rect 对应 Win32 的 RECT
type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

var pGetWindowRect = user32.MustFindProc("GetWindowRect")

var (
	panelMouseGetHwnd         func() uintptr
	panelMouseEnabled         func() bool
	panelMouseOnClickOutside func()
)

func panelMouseProc(nCode int, wParam, lParam uintptr) uintptr {
	if nCode == hcAction && panelMouseEnabled != nil && panelMouseEnabled() {
		if wParam == wmLButtonDown && panelMouseGetHwnd != nil {
			hwnd := panelMouseGetHwnd()
			if hwnd == 0 {
				return callNextHookEx(0, uintptr(nCode), wParam, lParam)
			}
			ms := (*msLLHookStruct)(unsafe.Pointer(lParam))
			var r rect
			ret, _, _ := syscall.SyscallN(pGetWindowRect.Addr(), hwnd, uintptr(unsafe.Pointer(&r)))
			if ret != 0 {
				outside := ms.Pt.X < r.Left || ms.Pt.X > r.Right || ms.Pt.Y < r.Top || ms.Pt.Y > r.Bottom
				if outside && panelMouseOnClickOutside != nil {
					// 异步派发，避免钩子回调里执行窗口操作阻塞系统消息链
					go panelMouseOnClickOutside()
				}
			}
		}
	}
	// 不吞事件：目标应用正常接收点击，与 Win+V 行为一致
	return callNextHookEx(0, uintptr(nCode), wParam, lParam)
}

// HookPanelMouse 在面板可见时拦截全局左键点击，点击坐标在窗口矩形外时触发 onClickOutside。
// getHwnd 在钩子触发时动态取窗口句柄（app.Run 之前 NativeWindow 返回 0，不能安装时传入）。
// 不吞事件，目标应用正常接收点击。
func HookPanelMouse(getHwnd func() uintptr, enabled func() bool, onClickOutside func()) (func(), error) {
	panelMouseGetHwnd = getHwnd
	panelMouseEnabled = enabled
	panelMouseOnClickOutside = onClickOutside
	mod, _, _ := syscall.SyscallN(pGetModuleHandle.Addr(), 0)
	cb := syscall.NewCallback(panelMouseProc)
	r, _, e := syscall.SyscallN(pSetWindowsHookEx.Addr(),
		uintptr(whMouseLL), cb, mod, 0)
	if r == 0 {
		return nil, osErr("SetWindowsHookEx(panelMouse)", e)
	}
	hhook := r
	return func() {
		syscall.SyscallN(pUnhookWindowsHookEx.Addr(), hhook)
		panelMouseGetHwnd = nil
		panelMouseEnabled = nil
		panelMouseOnClickOutside = nil
	}, nil
}
