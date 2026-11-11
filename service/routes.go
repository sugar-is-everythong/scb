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
}

func imageGroup(g *gin.RouterGroup) {
	g.GET("getBimg/:id", getBimg) // 原图
	g.GET("getSimg/:id", getSimg) // 获取缩略图
}
