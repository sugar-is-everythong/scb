package service

import (
	"net/http"
	"scb/set"

	"github.com/gin-gonic/gin"
)

// setGet 读取用户设置，文件缺失或损坏时返回默认值
func setGet(c *gin.Context) {
	OkWithData(c, set.Load())
}

// setSave 保存用户设置并回传落盘后的实际内容。
// 从当前配置起再覆盖：请求里缺字段时保留原值，不会把嵌套的开关清成 false
func setSave(c *gin.Context) {
	cfg := set.Load()
	if err := c.ShouldBindJSON(&cfg); err != nil {
		FailWithMsg(c, "设置格式错误")
		return
	}
	saved, err := set.Save(cfg)
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, saved)
}

// setBgUpload 接收前端文件对话框选中的图片路径，复制进数据目录并设为背景
func setBgUpload(c *gin.Context) {
	var body struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Path == "" {
		FailWithMsg(c, "请选择图片文件")
		return
	}
	cfg, err := set.SetBackground(body.Path)
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, cfg)
}

// setBgImage 输出当前背景图片，供三个窗口的 background-image 使用
func setBgImage(c *gin.Context) {
	p := set.BackgroundFile(set.Load())
	if p == "" {
		c.Status(http.StatusNotFound)
		return
	}
	c.File(p)
}
