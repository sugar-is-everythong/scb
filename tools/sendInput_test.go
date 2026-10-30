//go:build windows

package tools

import (
	"testing"
	"unsafe"
)

// TestInputSize 校验 INPUT 结构体大小与 Win32 的 sizeof(INPUT) 一致，
// 不一致时 SendInput 会直接丢弃所有事件且不报错。
func TestInputSize(t *testing.T) {
	want := uintptr(40) // x64
	if unsafe.Sizeof(uintptr(0)) == 4 {
		want = 28 // x86
	}
	if got := unsafe.Sizeof(input{}); got != want {
		t.Fatalf("sizeof(INPUT) = %d, 期望 %d", got, want)
	}
}

// TestUTF16Units 校验非 BMP 字符被拆成代理对
func TestUTF16Units(t *testing.T) {
	if got := utf16Units('A'); len(got) != 1 || got[0] != 'A' {
		t.Fatalf("utf16Units('A') = %v, 期望 [65]", got)
	}
	got := utf16Units('\U0001F600') // 代理对：D83D DE00
	if len(got) != 2 || got[0] != 0xD83D || got[1] != 0xDE00 {
		t.Fatalf("utf16Units(😀) = %v, 期望 [D83D DE00]", got)
	}
}

func TestSi(t *testing.T) {
	target := ForegroundWindow()
	println("target:", target)
}
