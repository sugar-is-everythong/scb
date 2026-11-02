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
}

func cbGroup(g *gin.RouterGroup) {
	// 读取：快照标志 + 各类型取值
	g.GET("/read", cbRead)
	g.GET("/text", cbGetText)
	g.GET("/image", cbGetImage)
	g.GET("/html", cbGetHTML)
	g.GET("/rtf", cbGetRTF)
	g.GET("/files", cbGetFiles)

	// 写入
	g.POST("/text", cbWriteText)
	g.POST("/image", cbWriteImage)
	g.POST("/imageCORS", cbWriteImageCORS)
	g.POST("/html", cbWriteHTML)
	g.POST("/rtf", cbWriteRTF)
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
