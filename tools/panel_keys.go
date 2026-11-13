//go:build windows

// 本文件实现"面板可见时拦截 ↑ ↓ Enter Esc 四键，转 emit 给前端"的钩子。
// 与 tools/hook.go 的 Win+V 拦截是两条独立钩子，互不干扰。
//
// 为什么需要它：面板显示时若不想抢前台，WebView2 拿不到键盘焦点；
// 把键通过钩子拦下来转成 Wails 事件推给前端，由 JS 直接 dispatch，可以
// 在不抢前台的前提下实现方向键导航 / Enter 选中 / Esc 收起。
package tools

import (
	"syscall"
	"unsafe"
)

// 我们关心的虚拟键码
const (
	vkUp     = 0x26 // VK_UP
	vkDown   = 0x28 // VK_DOWN
	vkReturn = 0x0D // VK_RETURN
	vkEscape = 0x1B // VK_ESCAPE
)

// vkToKeyName 把虚拟键码映射为 JS KeyboardEvent.key 的命名。
var vkToKeyName = map[uint32]string{
	vkUp:     "ArrowUp",
	vkDown:   "ArrowDown",
	vkReturn: "Enter",
	vkEscape: "Escape",
}

// 钩子状态：仅本文件使用，避免与 hook.go 中的 Win+V 状态互相影响。
var (
	panelKeysEnabled func() bool
	panelKeysOnKey   func(key string)
)

// panelKeysProc 是 WH_KEYBOARD_LL 的回调入口。
// 只在 enabled() 为 true 且按下目标键时拦截；其余情况原样放行。
func panelKeysProc(nCode int, wParam, lParam uintptr) uintptr {
	if nCode == hcAction && panelKeysEnabled != nil && panelKeysEnabled() {
		if wParam == wmKeyDown || wParam == wmSysKeyDown {
			ks := (*kbdLLHookStruct)(unsafe.Pointer(lParam))
			if name, ok := vkToKeyName[ks.VkCode]; ok {
				if panelKeysOnKey != nil {
					// 异步派发，避免在钩子线程做耗时操作（系统对低级钩子有超时限制）
					go panelKeysOnKey(name)
				}
				return 1 // 吞掉，前台应用收不到
			}
		}
	}
	return callNextHookEx(0, uintptr(nCode), wParam, lParam)
}

// HookPanelKeys 装一个 WH_KEYBOARD_LL 钩子：在 enabled() 返回 true 时拦截
// ↑ ↓ Enter Esc 四个键的按下事件，转成字符串通过 onKey 回调（通常再 emit 给前端）。
// 面板不可见（enabled 返回 false）时不拦截任何键，原窗口键盘输入不受影响。
// 返回的函数用于卸载钩子。
func HookPanelKeys(enabled func() bool, onKey func(key string)) (func(), error) {
	panelKeysEnabled = enabled
	panelKeysOnKey = onKey

	mod, _, _ := syscall.SyscallN(pGetModuleHandle.Addr(), 0)
	cb := syscall.NewCallback(panelKeysProc)
	r, _, e := syscall.SyscallN(pSetWindowsHookEx.Addr(),
		uintptr(whKeyboardLL),
		cb,
		mod,
		0, // 全局钩子：监听所有线程
	)
	if r == 0 {
		return nil, osErr("SetWindowsHookEx(panelKeys)", e)
	}
	hhook := r
	return func() {
		syscall.SyscallN(pUnhookWindowsHookEx.Addr(), hhook)
		panelKeysEnabled = nil
		panelKeysOnKey = nil
	}, nil
}
