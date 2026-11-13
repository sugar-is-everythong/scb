//go:build windows

package tools

import (
	"fmt"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	pSetWindowPos    = user32.MustFindProc("SetWindowPos")
	pShowWindow      = user32.MustFindProc("ShowWindow")
	pIsWindowVisible = user32.MustFindProc("IsWindowVisible")
)

const (
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
)

// showChromium 通过 unsafe 反射调用 Wails 内部 chromium.Show()。
// Wails 的 Show() 内部走 ShowWindow(SW_SHOW) 自带激活；
// 这里只取它"让 WebView2 渲染可见"的那一半，父窗口激活完全由调用方控制。
func showChromium(w *application.WebviewWindow) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tools: 反射调用 chromium.Show 失败: %v", r)
		}
	}()

	wv := reflect.ValueOf(w).Elem()
	implField := wv.FieldByName("impl")
	if !implField.IsValid() {
		return fmt.Errorf("tools: WebviewWindow.impl 字段不存在")
	}
	impl := reflect.NewAt(implField.Type(), unsafe.Pointer(implField.UnsafeAddr())).Elem().Interface()

	implVal := reflect.ValueOf(impl).Elem()
	chromiumField := implVal.FieldByName("chromium")
	if !chromiumField.IsValid() {
		return fmt.Errorf("tools: windowsWebviewWindow.chromium 字段不存在")
	}
	chromium := reflect.NewAt(chromiumField.Type(), unsafe.Pointer(chromiumField.UnsafeAddr())).Elem().Interface()

	showMethod := reflect.ValueOf(chromium).MethodByName("Show")
	if !showMethod.IsValid() {
		return fmt.Errorf("tools: chromium.Show 方法不存在")
	}
	showMethod.Call(nil)
	return nil
}

// ShowNoActivate 显示窗口但不激活，全程零抢焦点。
// 1. 反射调 chromium.Show() 确保 WebView2 渲染可见（绕开 Wails Show 里的 SW_SHOW）
// 2. SetWindowPos(SWP_NOACTIVATE|SWP_SHOWWINDOW) 把父窗口摆到可见状态，不碰前台
func ShowNoActivate(w application.Window, hwnd uintptr) error {
	wv, ok := w.(*application.WebviewWindow)
	if !ok {
		return fmt.Errorf("tools: 窗口不是 *WebviewWindow，无法反射调 chromium.Show")
	}
	if err := showChromium(wv); err != nil {
		return err
	}
	syscall.SyscallN(pSetWindowPos.Addr(),
		hwnd, 0, 0, 0, 0, 0,
		uintptr(swpNoSize|swpNoMove|swpNoZOrder|swpNoActivate|swpShowWindow),
	)
	return nil
}

// HideNoActivate 隐藏窗口。
// 不调 Wails 的 Hide()，因此 chromium 保持 Show 状态，下次显示无需重新初始化 WebView2。
func HideNoActivate(hwnd uintptr) {
	syscall.SyscallN(pShowWindow.Addr(), hwnd, 0) // SW_HIDE = 0
}

// IsWindowVisibleReal 直接读 WS_VISIBLE。
// mainW.IsVisible() 读的是 Wails 内部 windowShown 字段，绕开 Show()/Hide() 后不准，用这个代替。
func IsWindowVisibleReal(hwnd uintptr) bool {
	r, _, _ := syscall.SyscallN(pIsWindowVisible.Addr(), hwnd)
	return r != 0
}
