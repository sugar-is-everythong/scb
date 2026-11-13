package service

import (
	"net/http"
	"scb/tools"

	"github.com/gin-gonic/gin"
)

// setupRoutes configures the API routes
func (s *GinService) setupRoutes() {
	cbg := s.ginEngine.Group("/cb")
	cbGroup(cbg)

	fg := s.ginEngine.Group("/file")
	fileGroup(fg)

	wg := s.ginEngine.Group("/window")
	windowGroup(wg)

	sg := s.ginEngine.Group("/set")
	setGroup(sg)

	imageGroup(s.ginEngine.Group("/image"))
}

func cbGroup(g *gin.RouterGroup) {
	// 读取：快照标志 + 各类型取值
	g.GET("/read", cbRead)
	g.GET("/text", cbGetText)
	g.GET("/image", cbGetImage)
	g.GET("/html", cbGetHTML)
	g.GET("/files", cbGetFiles)

	// 写入
	g.POST("/text", cbWriteText)
	g.POST("/image", cbWriteImage)
	g.POST("/imageCORS", cbWriteImageCORS)
	g.POST("/html", cbWriteHTML)
	g.POST("/files", cbWriteFiles)

	g.GET("/inset", inset)
	g.POST("/hide", cbHide)

	g.GET("/getlocal", localStoreGet)
	g.POST("/savelocal", localStoreSet)
	g.POST("/deletelocal", localStoreDelete)
	g.POST("/setalllocal", localStoreSetAll)

}

func fileGroup(g *gin.RouterGroup) {
	for _, pan := range tools.GetPanName() {
		g.StaticFS("/"+pan, http.Dir(pan+":\\"))
	}
}

func windowGroup(g *gin.RouterGroup) {
	g.POST("create", create)
	g.POST("set", openSet)
}

func setGroup(g *gin.RouterGroup) {
	g.GET("/get", setGet)
	g.POST("/save", setSave)

	// 窗口背景：POST 转存用户选的图片，GET 输出当前背景图（三个窗口共用）
	g.POST("/bgimg", setBgUpload)
	g.GET("/bgimg", setBgImage)
}

func imageGroup(g *gin.RouterGroup) {
	g.GET("getBimg/:id", getBimg) // 原图
	g.GET("getSimg/:id", getSimg) // 获取缩略图
}
