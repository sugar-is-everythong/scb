//go:build windows

package tools

import (
	"testing"
)

// TestCursorPos 验证能取到鼠标的屏幕坐标
func TestCursorPos(t *testing.T) {
	x, y, err := CursorPos()
	if err != nil {
		t.Fatalf("CursorPos 失败: %v", err)
	}
	t.Logf("鼠标位置 x=%d y=%d", x, y)
}
