package service

import (
	"fmt"
	"math/rand"
	"scb/global"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	winWidth  = 418
	winHeight = 300
)

func create(c *gin.Context) {
	windowName := fmt.Sprintf("内容详情%d", rand.Int())
	global.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "内容详情",
		Name:             windowName,
		Width:            winWidth,
		Height:           winHeight,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(6, 7, 15, 0),
		URL:              "#/detail",
		//AlwaysOnTop:      true,
		BackgroundType: application.BackgroundTypeTranslucent,
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true,
		},
	})

	OkWithData(c, windowName)
}

// openSet 打开设置窗口，已存在则复用
func openSet(c *gin.Context) {
	sw, _ := global.App.Window.GetByName("设置窗口")
	sw.Center()
	sw.Show()
	OK_(c)
}
