package global

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	Local = "./local"
)

var (
	App        *application.App
	Foreground uintptr // 工具窗口显示前的句柄
)
