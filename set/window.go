package set

import (
	"scb/global"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var (
	winWidth  = 300
	winHeight = 418
)

const winName = "设置窗口"

func Window() {
	sw := global.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            winName,
		Name:             winName,
		Width:            winWidth,
		Height:           winHeight,
		Hidden:           true,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(6, 7, 15, 0),
		URL:              "#/set",
		//AlwaysOnTop:      true,
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
		},
	})

	sw.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		sw.Hide()
		event.Cancel()
	})
}
