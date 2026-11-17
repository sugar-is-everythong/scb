//go:build windows

// 本文件是 tools 包的全局键鼠钩子实现：用低级键盘钩子（WH_KEYBOARD_LL）
// 接管被系统占用的快捷键（如 Win+V 的系统剪贴板历史）。
//
// 为什么不用 RegisterHotKey：Win+V 这类系统快捷键由 Explorer 先行用
// RegisterHotKey 注册，Windows 不允许第二个进程注册同一组合，
// Wails 的 GlobalShortcut 在这种情况下必然失败。低级键盘钩子在按键投递到
// 消息队列之前拦截，是唯一能抢占系统快捷键的办法。
//
// 代价是要自己维护 Win 键的完整生命周期：Win 键按下先被吞掉，等下一个
// 键判定这是不是目标组合；不是目标组合就把 Win+该键整体重放（SendInput），
// 保证其他 Win 组合照常工作；单独点 Win 键则重放一次完整的点击，开始菜单
// 等行为不变。
package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// 低级键盘钩子相关常量
const (
	whKeyboardLL = 13 // WH_KEYBOARD_LL
	hcAction     = 0  // HC_ACTION

	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
	wmKeyUp      = 0x0101
	wmSysKeyUp   = 0x0105

	vkLWin  = 0x5B
	vkRWin  = 0x5C
	vkShift = 0x10
	vkCtrl  = 0x11

	llKHFInjected = 0x00000010 // 事件由 SendInput 注入
	llKHFAltDown  = 0x00000020 // Alt 处于按下状态
)

// hookReplayTag 是重放事件在 dwExtraInfo 上的标记。
// 钩子据此识别自己注入的按键并原样放行，避免重放事件再次进入判定逻辑形成自循环。
const hookReplayTag = 0x53434231 // "SCB1"

// 钩子状态：记录一次 Win 按下从出现到结束的处置过程
const (
	hookStateNone   = 0 // 没有 Win 键按下
	hookStateWait   = 1 // 已吞掉 Win 键按下，等待下一个按键判定组合
	hookStateHit    = 2 // 已命中目标组合，相关的抬起事件都要吞掉
	hookStateReplay = 3 // 已重放 Win+其他键，Win 键的抬起事件要放行以与注入的按下配对
)

// hookDebugLog 是临时诊断开关：打开后把每次钩子事件的处理结果追加写入
// %TEMP%\scb-hook.log，用于定位"某些窗口里按 Win+V 面板不弹"。
// 定位完成后置为 false 或删除相关代码。
const hookDebugLog = true

// 进程查询相关常量
const procQueryLimitedInformation = 0x1000

// 看门狗参数
const (
	hookWatchInterval = 15 * time.Millisecond  // Win 键物理状态的轮询间隔
	hookWatchGrace    = 200 * time.Millisecond // 按下后给钩子投递事件的宽限期
	hookWatchCooldown = 2 * time.Second        // 两次判定失效的最小间隔
)

// hookLastEvent 是最近一次钩子回调的时间（UnixNano）。
// 钩子回调线程写、看门狗协程读，因此用原子操作。
var hookLastEvent atomic.Int64

var (
	kernel32 = syscall.MustLoadDLL("kernel32.dll")

	pSetWindowsHookEx    = user32.MustFindProc("SetWindowsHookExW")
	pCallNextHookEx      = user32.MustFindProc("CallNextHookEx")
	pUnhookWindowsHookEx = user32.MustFindProc("UnhookWindowsHookEx")
	pGetAsyncKeyState    = user32.MustFindProc("GetAsyncKeyState")
	pGetModuleHandle     = kernel32.MustFindProc("GetModuleHandleW")

	// 诊断日志用于识别按下按键时前台是哪个程序
	pGetWindowThreadProcessId  = user32.MustFindProc("GetWindowThreadProcessId")
	pOpenProcess               = kernel32.MustFindProc("OpenProcess")
	pQueryFullProcessImageName = kernel32.MustFindProc("QueryFullProcessImageNameW")
	pCloseHandle               = kernel32.MustFindProc("CloseHandle")
)

// 钩子的运行状态。低级钩子回调只在安装线程上执行，无需加锁。
var (
	hookTargetKey uint32 // 目标键的虚拟键码
	hookOnHit     func() // 命中目标组合时的回调
	hookState     int    // 见 hookState* 常量
	hookWinVk     uint32 // 本次 Win 按下的键（VK_LWIN/VK_RWIN）

	// 诊断日志的状态：文件句柄与前台窗口缓存
	hookLogFile *os.File
	hookFgHwnd  uintptr
	hookFgName  string

	// 钩子的安装信息：回调桩、安装线程的模块句柄与当前钩子句柄。
	// 重装钩子时要复用同一个回调桩（syscall.NewCallback 的数量有限）。
	hookProc   uintptr
	hookMod    uintptr
	hookHandle uintptr
)

// kbdLLHookStruct 对应 Win32 的 KBDLLHOOKSTRUCT
type kbdLLHookStruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// HookWinV 拦截 Win+V：命中时调用 onHit，并把该组合从系统输入流里吃掉，
// 使系统自带的剪贴板历史（Win+V）不再弹出。
//
// 钩子回调运行在安装线程的消息循环里，安装线程必须有消息循环
// （Wails 应用的主线程满足这一条件）。回调有超时限制（Win10 2004+ 默认 1 秒），
// 因此 onHit 会被放到独立协程执行，不要在钩子回调里做耗时操作。
//
// 返回的函数用于卸载钩子。
func HookWinV(onHit func()) (func(), error) {
	hookTargetKey = vkV
	hookOnHit = onHit
	hookState = hookStateNone

	// 钩子过程在本进程内，hMod 传自身模块句柄
	mod, _, _ := syscall.SyscallN(pGetModuleHandle.Addr(), 0)
	hookMod = mod
	if hookProc == 0 {
		hookProc = syscall.NewCallback(keyboardHookCallback)
	}
	if err := installHook(); err != nil {
		return nil, err
	}
	return func() {
		uninstallHook()
	}, nil
}

// installHook 安装全局低级键盘钩子
func installHook() error {
	r, _, e := syscall.SyscallN(pSetWindowsHookEx.Addr(),
		uintptr(whKeyboardLL),
		hookProc,
		hookMod,
		0, // 全局钩子：监听所有线程
	)
	if r == 0 {
		return osErr("SetWindowsHookEx", e)
	}
	hookHandle = r
	hookLog("hook installed hhook=%#x\n", hookHandle)
	return nil
}

// uninstallHook 卸载当前钩子
func uninstallHook() {
	if hookHandle == 0 {
		return
	}
	syscall.SyscallN(pUnhookWindowsHookEx.Addr(), hookHandle)
	hookHandle = 0
}

// ReinstallHook 卸载并重新安装钩子。
//
// 用途：低级键盘钩子可能被系统按回调超时静默摘除，也可能被后安装的
// 同类钩子排到链尾而再也收不到事件（表现为某些程序在前台时钩子完全
// 不触发）。重新安装会让本钩子重新排到链头并恢复响应。
//
// 必须在钩子的安装线程（带消息循环的线程）上调用，换线程安装会把
// 钩子绑到一个没有消息循环的线程上，导致回调永远不被调用。
func ReinstallHook() {
	if hookHandle == 0 {
		return
	}
	uninstallHook()
	if err := installHook(); err != nil {
		hookLog("hook reinstall failed: %v\n", err)
	}
}

// StartHookWatchdog 启动钩子看门狗。
//
// 原理：用 GetAsyncKeyState 轮询 Win 键的物理状态，只要物理上按下了 Win 而
// 钩子在宽限期内一次回调都没收到，就说明钩子已被系统摘除或被更前面的钩子
// 抢先吞掉，此时调 onFail 让调用方在安装线程上重装钩子。
//
// 这样做只会在“已经确认失效”的那一刻重装，不会像定时重装那样把正在进行的
// 按键序列打断。onFail 会从独立协程调用。
// 返回的函数用于停止看门狗。
func StartHookWatchdog(onFail func()) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(hookWatchInterval)
		defer ticker.Stop()
		var (
			prevDown bool
			lastFail time.Time
		)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
			down := asyncKeyDown(vkLWin) || asyncKeyDown(vkRWin)
			if down && !prevDown {
				// 抓到 Win 键按下的上升沿，等过宽限期再确认钩子有没有收到事件
				edge := time.Now()
				time.Sleep(hookWatchGrace)
				if hookLastEvent.Load() < edge.UnixNano() && time.Since(lastFail) > hookWatchCooldown {
					lastFail = time.Now()
					hookLog("watchdog: win pressed but hook missed event, reinstall\n")
					if onFail != nil {
						onFail()
					}
				}
			}
			prevDown = down
		}
	}()
	return func() { close(stop) }
}

// keyboardHookCallback 是低级键盘钩子的回调入口。
// 必须是包级函数：Windows 通过函数指针回调，不能依赖闭包环境。
func keyboardHookCallback(nCode int, wParam, lParam uintptr) uintptr {
	if nCode == hcAction {
		hookLastEvent.Store(time.Now().UnixNano())
		ks := (*kbdLLHookStruct)(unsafe.Pointer(lParam))
		switch wParam {
		case wmKeyDown, wmSysKeyDown:
			prev := hookState
			swallowed := hookKeyDown(ks)
			hookTrace("down", wParam, ks, prev, swallowed)
			if swallowed {
				return 1 // 吞掉，不让系统看到这个按键
			}
		case wmKeyUp, wmSysKeyUp:
			prev := hookState
			swallowed := hookKeyUp(ks)
			hookTrace("up", wParam, ks, prev, swallowed)
			if swallowed {
				return 1
			}
		}
	}
	return callNextHookEx(0, uintptr(nCode), wParam, lParam)
}

// hookTrace 记录一次钩子事件的处理过程，仅供临时诊断使用。
func hookTrace(kind string, wParam uintptr, ks *kbdLLHookStruct, prevState int, swallowed bool) {
	if !hookDebugLog {
		return
	}
	flags := "-"
	if ks.Flags&llKHFInjected != 0 {
		flags = "INJECTED"
	}
	if isReplayEvent(ks) {
		flags += "+replayTag"
	}
	action := "pass"
	if swallowed {
		action = "SWALLOW"
	}
	hookLog("%s vk=%#04x sys=%t flags=%s state=%s->%s %s fg=%s\n",
		kind, ks.VkCode, wParam == wmSysKeyDown || wParam == wmSysKeyUp, flags,
		hookStateName(prevState), hookStateName(hookState), action, hookForegroundName())
}

// hookStateName 把钩子状态转成日志里的可读名字
func hookStateName(s int) string {
	switch s {
	case hookStateNone:
		return "None"
	case hookStateWait:
		return "Wait"
	case hookStateHit:
		return "Hit"
	case hookStateReplay:
		return "Replay"
	}
	return "?"
}

// hookForegroundName 返回前台窗口所属进程的文件名，同一窗口结果做缓存避免重复查询
func hookForegroundName() string {
	hwnd, _, _ := syscall.SyscallN(pGetForegroundWindow.Addr())
	if hwnd != hookFgHwnd || hookFgName == "" {
		hookFgHwnd = hwnd
		hookFgName = processNameOfWindow(hwnd)
	}
	return hookFgName
}

// processNameOfWindow 通过窗口句柄取得所属进程的可执行文件名
func processNameOfWindow(hwnd uintptr) string {
	if hwnd == 0 {
		return "-"
	}
	var pid uint32
	syscall.SyscallN(pGetWindowThreadProcessId.Addr(), hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "-"
	}
	h, _, _ := syscall.SyscallN(pOpenProcess.Addr(), procQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return "?"
	}
	defer syscall.SyscallN(pCloseHandle.Addr(), h)
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	r, _, _ := syscall.SyscallN(pQueryFullProcessImageName.Addr(), h, 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		return "?"
	}
	return filepath.Base(syscall.UTF16ToString(buf[:size]))
}

// hookLog 追加一行诊断日志到 %TEMP%\scb-hook.log
func hookLog(format string, args ...any) {
	if !hookDebugLog {
		return
	}
	if hookLogFile == nil {
		f, err := os.OpenFile(filepath.Join(os.TempDir(), "scb-hook.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		hookLogFile = f
	}
	fmt.Fprintf(hookLogFile, "%s %s", time.Now().Format("2006-01-02 15:04:05.000"),
		fmt.Sprintf(format, args...))
}

// hookKeyDown 处理按键按下，返回 true 表示吞掉该事件
func hookKeyDown(ks *kbdLLHookStruct) bool {
	if isReplayEvent(ks) {
		return false // 自己注入的重放事件，原样放行
	}

	if ks.VkCode == vkLWin || ks.VkCode == vkRWin {
		if hookState == hookStateHit || hookState == hookStateReplay {
			return true // 组合已判定，忽略 Win 键按住时的重复事件
		}
		hookState = hookStateWait
		hookWinVk = ks.VkCode
		return true
	}

	if hookState != hookStateWait {
		return false
	}

	if ks.VkCode == hookTargetKey && !otherModifierDown(ks) {
		hookState = hookStateHit
		if hookOnHit != nil {
			go hookOnHit()
		}
		return true
	}

	// 不是目标组合：把 Win+该键整体重放，其他 Win 组合照常工作
	replayWinCombo(hookWinVk, ks.VkCode)
	hookState = hookStateReplay
	return true
}

// hookKeyUp 处理按键抬起，返回 true 表示吞掉该事件
func hookKeyUp(ks *kbdLLHookStruct) bool {
	if isReplayEvent(ks) {
		return false
	}

	switch hookState {
	case hookStateWait:
		if ks.VkCode == hookWinVk {
			// 只按了 Win 键：重放一次完整的点击，开始菜单等行为保持不变
			hookState = hookStateNone
			replayWinTap(hookWinVk)
			return true
		}
	case hookStateHit:
		switch ks.VkCode {
		case hookTargetKey:
			hookState = hookStateNone
			return true
		case hookWinVk:
			return true // 吞掉，避免系统看到一次没有按下过的 Win 抬起
		}
	case hookStateReplay:
		if ks.VkCode == hookWinVk {
			hookState = hookStateNone
			return false // 放行，与注入的 Win 键按下配对
		}
	}
	return false
}

// isReplayEvent 判断事件是否为钩子自己重放注入的按键
func isReplayEvent(ks *kbdLLHookStruct) bool {
	return ks.Flags&llKHFInjected != 0 && ks.DwExtraInfo == hookReplayTag
}

// otherModifierDown 判断除 Win 外是否还有修饰键按住：
// 带 Ctrl/Shift/Alt 的组合不是目标组合，不应被拦截。
func otherModifierDown(ks *kbdLLHookStruct) bool {
	if ks.Flags&llKHFAltDown != 0 {
		return true
	}
	return asyncKeyDown(vkCtrl) || asyncKeyDown(vkShift)
}

// asyncKeyDown 用 GetAsyncKeyState 判断某个虚拟键当前是否按住
func asyncKeyDown(vk uintptr) bool {
	r, _, _ := syscall.SyscallN(pGetAsyncKeyState.Addr(), vk)
	return r&0x8000 != 0
}

// replayWinCombo 重放 Win+其他键：注入按下事件（原始按下已被吞掉），
// 后续的抬起事件由系统正常收到，与注入的按下配对。
func replayWinCombo(winVk, keyVk uint32) {
	replayKeys(keyboardInput{wVk: uint16(winVk)}, keyboardInput{wVk: uint16(keyVk)})
}

// replayWinTap 重放一次完整的 Win 键点击（按下加抬起）
func replayWinTap(winVk uint32) {
	replayKeys(keyboardInput{wVk: uint16(winVk)}, keyboardInput{wVk: uint16(winVk), dwFlags: keyEventKeyUp})
}

// replayKeys 注入一组按键事件，并打上重放标记
func replayKeys(keys ...keyboardInput) {
	ins := make([]input, 0, len(keys))
	for _, k := range keys {
		k.dwExtraInfo = hookReplayTag
		ins = append(ins, keyboardEvent(k))
	}
	// 重放失败只是这一次组合失效，不影响程序运行
	_, _ = sendInput(ins)
}

// callNextHookEx 把事件交给钩子链的下一个环节
func callNextHookEx(hhk uintptr, nCode uintptr, wParam, lParam uintptr) uintptr {
	r, _, _ := syscall.SyscallN(pCallNextHookEx.Addr(), hhk, nCode, wParam, lParam)
	return r
}
