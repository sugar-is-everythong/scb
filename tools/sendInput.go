//go:build windows

// 本文件是 tools 包的 Win32 输入注入实现：把合成的键盘事件送入系统输入队列，
// 用于在目标程序获得焦点后模拟按键（如 Ctrl+V 粘贴）。
//
// 原理：Windows 没有"触发粘贴"的系统事件，粘贴是目标控件对快捷键的响应，
// 所以只能制造一次真实的 Ctrl+V 输入，由控件自己从剪贴板取数并插到光标处。
package tools

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// Win32 输入注入相关常量
const (
	inputKeyboard   = 1      // INPUT_KEYBOARD
	keyEventKeyUp   = 0x0002 // KEYEVENTF_KEYUP
	keyEventUnicode = 0x0004 // KEYEVENTF_UNICODE
	keyEventExtKey  = 0x0001 // KEYEVENTF_EXTENDEDKEY

	vkControl = 0x11 // VK_CONTROL
	vkMenu    = 0x12 // VK_MENU (Alt 键)
	vkV       = 0x56 // 'V' 键
)

var (
	user32 = syscall.MustLoadDLL("user32.dll")

	pSendInput           = user32.MustFindProc("SendInput")
	pGetForegroundWindow = user32.MustFindProc("GetForegroundWindow")
	pSetForegroundWindow = user32.MustFindProc("SetForegroundWindow")
	pLockSetForeground   = user32.MustFindProc("LockSetForegroundWindow")
)

// keyboardInput 对应 Win32 的 KEYBDINPUT
type keyboardInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	dwTime      uint32
	dwExtraInfo uintptr
}

// mouseInput 对应 Win32 的 MOUSEINPUT
type mouseInput struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	dwTime      uint32
	dwExtraInfo uintptr
}

// input 对应 Win32 的 INPUT 结构体：type(4) + 联合体。
// 联合体按最大成员 MOUSEINPUT 占位（x64 为 32 字节、x86 为 24 字节），
// KEYBDINPUT 写在联合体起始偏移处（与 C 侧各联合体成员 offset 0 一致）。
// Go 的对齐规则与 C 一致，因此 unsafe.Sizeof(input{}) 恰好等于 C 的 sizeof(INPUT)：
// x64 为 40 字节、x86 为 28 字节，可直接传给 SendInput。
type input struct {
	type_ uint32
	union mouseInput
}

// keyboardEvent 构造一个键盘事件：把 KEYBDINPUT 写入联合体起始处
func keyboardEvent(ki keyboardInput) input {
	in := input{type_: inputKeyboard}
	*(*keyboardInput)(unsafe.Pointer(&in.union)) = ki
	return in
}

// osErr 把系统调用错误包装成带操作名的错误
func osErr(op string, e error) error {
	if errno, ok := e.(syscall.Errno); ok && errno != 0 {
		return fmt.Errorf("tools: %s: %w", op, errno)
	}
	return fmt.Errorf("tools: %s: 调用失败", op)
}

// sendInput 把一批输入事件送入系统输入队列，返回实际注入的事件数。
// 实际注入数少于请求数说明被系统拦截（UIPI：目标程序权限高于本进程）。
func sendInput(inputs []input) (uint32, error) {
	if len(inputs) == 0 {
		return 0, nil
	}
	r, _, e := syscall.SyscallN(pSendInput.Addr(),
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(input{}),
	)
	if r == 0 {
		return 0, osErr("SendInput", e)
	}
	return uint32(r), nil
}

// Paste 模拟按下 Ctrl+V，把当前剪贴板内容粘贴到前台窗口中聚焦的控件。
//
// 注意：
//  1. SendInput 只能投递给当前前台窗口，调用前需确保目标窗口已在前台；
//  2. 目标程序以更高权限运行时，输入会被 UIPI 拦截，此时返回错误；
//  3. 目标控件自己从剪贴板取数并插入，因此粘贴位置由目标控件的光标决定。
func Paste() error {
	ins := []input{
		keyboardEvent(keyboardInput{wVk: vkControl}),
		keyboardEvent(keyboardInput{wVk: vkV}),
		keyboardEvent(keyboardInput{wVk: vkV, dwFlags: keyEventKeyUp}),
		keyboardEvent(keyboardInput{wVk: vkControl, dwFlags: keyEventKeyUp}),
	}
	sent, err := sendInput(ins)
	if err != nil {
		return err
	}
	if sent != uint32(len(ins)) {
		return errors.New("tools: SendInput 只注入了部分事件，目标程序权限更高时会被 UIPI 拦截")
	}
	return nil
}

// TypeText 用 Unicode 键盘事件逐字符输入文本，不经过剪贴板。
// 只适用于纯文本，富文本、图片等仍需走 Paste。
func TypeText(text string) error {
	var ins []input
	for _, r := range text {
		for _, unit := range utf16Units(r) {
			ins = append(ins,
				keyboardEvent(keyboardInput{wScan: unit, dwFlags: keyEventUnicode}),
				keyboardEvent(keyboardInput{wScan: unit, dwFlags: keyEventUnicode | keyEventKeyUp}),
			)
		}
	}
	// 分批注入，避免单次事件过多
	const batch = 64
	for start := 0; start < len(ins); start += batch {
		end := min(start+batch, len(ins))
		sent, err := sendInput(ins[start:end])
		if err != nil {
			return err
		}
		if sent != uint32(end-start) {
			return errors.New("tools: SendInput 只注入了部分事件，目标程序权限更高时会被 UIPI 拦截")
		}
	}
	return nil
}

// utf16Units 把一个 rune 拆成 UTF-16 码元，非 BMP 字符拆成代理对
func utf16Units(r rune) []uint16 {
	if r <= 0xFFFF {
		return []uint16{uint16(r)}
	}
	r -= 0x10000
	return []uint16{uint16(0xD800 + (r >> 10)), uint16(0xDC00 + (r & 0x3FF))}
}

// ForegroundWindow 返回当前前台窗口句柄，没有前台窗口时返回 0
func ForegroundWindow() uintptr {
	r, _, _ := syscall.SyscallN(pGetForegroundWindow.Addr())
	return r
}

// FocusWindow 尝试把指定窗口设为前台窗口。
// Windows 限制非前台进程抢占前台，成功率不高，只作兜底手段。
func FocusWindow(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := syscall.SyscallN(pSetForegroundWindow.Addr(), hwnd)
	return r != 0
}

// ForceForeground 强制把指定窗口激活到前台。
// Windows 11 的前台窗口锁定机制可能阻止 SetForegroundWindow 直接生效，
// 模拟一次 Alt 键按下+释放可以解锁前台锁定，让后续的 SetForegroundWindow 成功。
func ForceForeground(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	// 模拟 Alt 键按下+释放，解锁前台窗口锁定
	ins := []input{
		keyboardEvent(keyboardInput{wVk: vkMenu, dwFlags: keyEventExtKey}),
		keyboardEvent(keyboardInput{wVk: vkMenu, dwFlags: keyEventExtKey | keyEventKeyUp}),
	}
	_, _ = sendInput(ins)
	// 现在尝试激活窗口
	return FocusWindow(hwnd)
}

// WaitForegroundWindow 等待前台窗口切换到 target 窗口。
//
// 本程序窗口隐藏后，系统会把前台还给之前的窗口，用它判断切换是否完成，
// 比固定等待固定时长更准：切换通常几十毫秒内完成，粘错窗口的风险也更低。
// 返回 false 表示超时仍未切换。
func WaitForegroundWindow(target uintptr, timeout time.Duration) bool {
	if target == 0 {
		return false
	}
	deadline := time.Now().Add(timeout)
	for {
		if ForegroundWindow() == target {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// LockSetForegroundWindow 锁定/解锁前台切换。
// 锁定后 ShowWindow(SW_SHOW) 等内部走 SetForegroundWindow 的激活路径会被系统拦截，
// 窗口仍然可见但不拿前台。用于在调用 Wails Show() 时做到“可见但零激活”。
//
// 注意：必须成对调用，且只锁极短区间（包住一次 Show 调用即可），长时间锁定
// 会阻止用户手动切换窗口。
func LockSetForegroundWindow(lock bool) {
	code := uintptr(2) // LSFW_UNLOCK
	if lock {
		code = 1 // LSFW_LOCK
	}
	syscall.SyscallN(pLockSetForeground.Addr(), code)
}
