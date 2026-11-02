package main

import (
	"embed"
	"log"
	"scb/global"
	"scb/service"
	"scb/tools"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	winWidth  = 300
	winHeight = 418
)

func main() {

	app := application.New(application.Options{
		Name: "粘一下",
		Services: []application.Service{
			application.NewServiceWithOptions(service.NewGinService(), application.ServiceOptions{
				Route: "/wg",
			}),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
	})
	global.App = app

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "粘一下",
		Name:             "粘一下",
		Width:            winWidth,
		Height:           winHeight,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(6, 7, 15, 0),
		URL:              "/index",
		Hidden:           true,
		//HideOnFocusLost:  true,
		AlwaysOnTop:    true,
		BackgroundType: application.BackgroundTypeTransparent,
		Windows: application.WindowsWindow{
			/*	ExStyle: w32.WS_EX_NOACTIVATE | w32.WS_EX_LAYERED |
				w32.WS_EX_TOPMOST |
				w32.WS_EX_NOREDIRECTIONBITMAP,*/
			DisableFramelessWindowDecorations: true,
			HiddenOnTaskbar:                   true,
			Permissions: map[application.CoreWebView2PermissionKind]application.CoreWebView2PermissionState{
				application.CoreWebView2PermissionKindOtherSensors: application.CoreWebView2PermissionStateAllow,
			},
		},
	})

	register(app)

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func register(app *application.App) {
	mainW, has := app.Window.GetByName("粘一下")
	if !has {
		log.Println("Window not found")
	}
	showAtCursor := func() {
		x, y, err := tools.CursorPos()
		mx := int(x)
		my := int(y)
		if err != nil {
			mainW.Center()
		} else {
			// 找到鼠标所在的屏幕（用 WorkArea 避开任务栏）
			screen := app.Screen.ScreenNearestDipPoint(application.Point{X: mx, Y: my})
			wa := screen.WorkArea

			// 水平：优先右侧，放不下则左侧
			var posX int
			if mx+10+winWidth <= wa.X+wa.Width {
				posX = mx + 10
			} else {
				posX = mx - 10 - winWidth
			}

			// 垂直：默认以鼠标为中心
			posY := my - winHeight/2
			if posY+winHeight > wa.Y+wa.Height {
				// 底部放不下：窗口右下角对齐鼠标上方 10px
				posY = my - 10 - winHeight
			}
			if posY < wa.Y {
				// 顶部放不下：窗口左上角对齐鼠标下方 10px
				posY = my + 10
			}

			mainW.SetPosition(posX, posY)
		}

		mainW.Show()
		mainW.Focus()

		// Win11 前台锁定可能导致 Focus() 未真正激活窗口，检查并强制激活
		hwnd := uintptr(mainW.NativeWindow())
		if hwnd != 0 && tools.ForegroundWindow() != hwnd {
			tools.ForceForeground(hwnd)
		}
	}
	err := app.GlobalShortcut.Register("super+v", showAtCursor)

	if err != nil {
		log.Println("全局快捷键注册失败", err.Error())
	}

	// Win+V 被系统（剪贴板历史）占用，GlobalShortcut 注册不了，
	// 改用低级键盘钩子拦截并接管这个组合。
	if _, err := tools.HookWinV(func() {
		application.InvokeSync(showAtCursor)
	}); err != nil {
		log.Println("Win+V 钩子安装失败", err.Error())
	}
}
