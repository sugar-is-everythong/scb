package service

import (
	"net/http"
	"scb/cb"

	"github.com/gin-gonic/gin"
)

func getSimg(c *gin.Context) {
	id := c.Param("id")
	im, has := cb.ImgArray.Find(func(a any) bool {
		return a.(cb.LuaArrImage).Id == id
	})

	if has {
		c.Data(http.StatusOK, "image/jpeg", im.(cb.LuaArrImage).SData)
		return
	}
	FailWithMsg(c, "图片不存在")
}

func getBimg(c *gin.Context) {
	id := c.Param("id")
	im, has := cb.ImgArray.Find(func(a any) bool {
		return a.(cb.LuaArrImage).Id == id
	})

	if has {
		c.Data(http.StatusOK, "image/png", im.(cb.LuaArrImage).BData)
		return
	}
	FailWithMsg(c, "图片不存在")
}
