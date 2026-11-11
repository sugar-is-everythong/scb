package service

import (
	"scb/cb"

	"github.com/gin-gonic/gin"
)

func localStoreGet(c *gin.Context) {
	s := cb.Store.All()
	OkWithData(c, s)
}

func localStoreSet(c *gin.Context) {
	var r cb.RecordJs

	if err := c.ShouldBindJSON(&r); err != nil {
		FailWithMsg(c, "请求参数解析"+err.Error())
		return
	}

	if err := cb.Store.SetStoreRJ(&r); err != nil {
		FailWithMsg(c, "存储错误"+err.Error())
		return
	}
	OK_(c)
}

func localStoreDelete(c *gin.Context) {
	var req struct {
		Index int `json:"index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.Store.Delete(req.Index); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

func localStoreSetAll(c *gin.Context) {
	var records []cb.RecordJs
	if err := c.ShouldBindJSON(&records); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.Store.SetAll(records); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}
