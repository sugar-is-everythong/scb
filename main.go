package main

import (
	"embed"
	"log"
	"scb/global"
	"scb/service"
	"scb/set"
	"scb/tools"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 启动前先读设置：窗口尺寸和快捷键都以 config.json 为准，
	// 改完后重启即生效
	cfg := set.Load()

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
	errr := app.Autostart.EnableWithOptions(application.AutostartOptions{
		Identifier: "scb_app",
	})
	if errr != nil {
		log.Println("创建子启东市失败", errr.Error())
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "粘一下",
		Name:             set.MainWindowName,
		Width:            cfg.Width,
		Height:           cfg.Height,
		Frameless:        true,
		DevToolsEnabled:  true,
		BackgroundColour: application.NewRGBA(6, 7, 15, 0),
		URL:              "#/index",
		Hidden:           true,
		HideOnFocusLost:  true,
		AlwaysOnTop:      true,
		BackgroundType:   application.BackgroundTypeTranslucent,
		Windows: application.WindowsWindow{
			ExStyle: w32.WS_EX_NOACTIVATE | w32.WS_EX_LAYERED |
				w32.WS_EX_TOPMOST |
				w32.WS_EX_NOREDIRECTIONBITMAP,
			DisableFramelessWindowDecorations: true,
			HiddenOnTaskbar:                   true,
		},
	})

	set.Window()
	register(app, cfg.Shortcut)

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func register(app *application.App, shortcut string) {
	mainW, has := app.Window.GetByName(set.MainWindowName)
	if !has {
		log.Println("Window not found")
		return
	}
	showAtCursor := func() {
		// 切换：面板已显示则收起。
		// 用 Win32 真实可见性判断，绕开 Wails 的 windowShown 字段（我们不用它的 Show/Hide）。
		hwnd := uintptr(mainW.NativeWindow())
		if tools.IsWindowVisibleReal(hwnd) {
			tools.HideNoActivate(hwnd)
			return
		}

		global.Foreground = tools.ForegroundWindow() // 先获取前台的窗口句柄，用于在粘贴完以后还原

		// 窗口尺寸可能刚在设置页改过，这里重新读取，定位按当前实际大小计算
		cfg := set.Load()

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
			if mx+10+cfg.Width <= wa.X+wa.Width {
				posX = mx + 10
			} else {
				posX = mx - 10 - cfg.Width
			}

			// 垂直：默认以鼠标为中心
			posY := my - cfg.Height/2
			if posY+cfg.Height > wa.Y+wa.Height {
				// 底部放不下：窗口右下角对齐鼠标上方 10px
				posY = my - 10 - cfg.Height
			}
			if posY < wa.Y {
				// 顶部放不下：窗口左上角对齐鼠标下方 10px
				posY = my + 10
			}

			mainW.SetPosition(posX, posY)
		}

		// 反射调 chromium.Show() 让 WebView2 渲染可见，再 SetWindowPos 显示父窗口。
		// 全程不走 ShowWindow(SW_SHOW)，原窗口前台一刻不动。
		if err := tools.ShowNoActivate(mainW, hwnd); err != nil {
			log.Println("面板显示失败", err.Error())
		}
	}
	err := app.GlobalShortcut.Register(shortcut, showAtCursor)

	if err != nil {
		log.Println("全局快捷键注册失败", shortcut, err.Error())
	}

	// 装面板键盘钩子：面板可见时拦 ↑↓EnterEsc 转 emit 给前端。
	// enabled 用 Win32 真实可见性，与 showAtCursor 里的切换判断保持一致。
	if _, err := tools.HookPanelKeys(
		func() bool { return tools.IsWindowVisibleReal(uintptr(mainW.NativeWindow())) },
		func(key string) {
			app.Event.Emit("key-down", map[string]string{"key": key})
		},
	); err != nil {
		log.Println("面板键盘钩子安装失败", err.Error())
	}

	// Win+V 被系统（剪贴板历史）占用，GlobalShortcut 注册不了，
	// 改用低级键盘钩子拦截并接管这个组合。
	// 只有配置里指定的正好是 Win+V 时才装钩子：换成其它组合后如果还留着钩子，
	// Win+V 会照样唤起窗口，等于快捷键改不掉。
	if strings.EqualFold(strings.TrimSpace(shortcut), "super+v") {
		if _, err := tools.HookWinV(func() {
			application.InvokeSync(showAtCursor)
		}); err != nil {
			log.Println("Win+V 钩子安装失败", err.Error())
		}
	}
}
