package service

import (
	"net/http"
	"scb/cb"
	"scb/tools"

	"github.com/gin-gonic/gin"
)

// recordInfo 是 cb.Record 的 JSON 视图：只暴露各格式有无的标志。
type recordInfo struct {
	HasText  bool `json:"hasText"`
	HasImage bool `json:"hasImage"`
	HasHTML  bool `json:"hasHTML"`
	HasRTF   bool `json:"hasRTF"`
	HasFiles bool `json:"hasFiles"`
}

// infoOf 将快照转为 JSON 视图
func infoOf(r *cb.Record) recordInfo {
	return recordInfo{
		HasText:  r.HasText,
		HasImage: r.HasImage,
		HasHTML:  r.HasHTML,
		HasRTF:   r.HasRTF,
		HasFiles: r.HasFiles,
	}
}

// ---------- 读取 ----------

// cbRead GET /cb/read：读取当前剪贴板快照，返回各格式有无的标志。
// 内容按类型调 /cb/text、/cb/image 等接口获取（与 cb 包 Read + Get* 用法一致）。
func cbRead(c *gin.Context) {
	rec, err := cb.Read()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, infoOf(rec))
}

// cbGetText GET /cb/text：当前剪贴板中的纯文本；无文本时返回空字符串。
func cbGetText(c *gin.Context) {
	rec, err := cb.Read()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, rec.GetText())
}

// cbGetImage GET /cb/image：当前剪贴板中的图片，以 PNG 二进制返回（浏览器可直接预览）。
func cbGetImage(c *gin.Context) {
	rec, err := cb.Read()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	png, err := rec.GetImage()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

// cbGetHTML GET /cb/html：当前剪贴板中的 HTML 富文本，含 SourceURL 等头部元信息。
func cbGetHTML(c *gin.Context) {
	rec, err := cb.Read()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	h, err := rec.GetHTML()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, gin.H{
		"fragment":  h.Fragment,
		"sourceURL": h.SourceURL,
		"version":   h.Version,
	})
}

// cbGetRTF GET /cb/rtf：当前剪贴板中的 RTF 富文本。
func cbGetRTF(c *gin.Context) {
	rec, err := cb.Read()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	rtf, err := rec.GetRTF()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, string(rtf))
}

// cbGetFiles GET /cb/files：当前剪贴板中的文件路径列表。
func cbGetFiles(c *gin.Context) {
	rec, err := cb.Read()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	files, err := rec.GetFiles()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OkWithData(c, files)
}

// ---------- 写入 ----------

// cbWriteText POST /cb/text：请求体为要写入剪贴板的原始文本。
func cbWriteText(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.WriteText(string(data)); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

// cbWriteImage POST /cb/image：请求体为 PNG 编码的图片字节。
func cbWriteImage(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.WriteImage(data); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

// cbWriteImageCORS POST /cb/imageCORS：通过后端代理拉取外域图片写入剪贴板，解决前端跨域问题。
func cbWriteImageCORS(c *gin.Context) {
	imgURL := c.Query("url")
	if imgURL == "" {
		FailWithMsg(c, "url 不能为空")
		return
	}

	imgData, err := tools.GetWebImage(imgURL)
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}

	if err := cb.WriteImage(imgData); err != nil {
		FailWithMsg(c, err.Error())
		return
	}

	OK_(c)
}

// cbWriteHTML POST /cb/html：请求体为 HTML 片段，按 CF_HTML 标准格式写入。
func cbWriteHTML(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.WriteHTML(string(data)); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

// cbWriteRTF POST /cb/rtf：请求体为 RTF 原始字节。
func cbWriteRTF(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.WriteRTF(data); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

// cbWriteFiles POST /cb/files：请求体为 JSON 字符串数组，如 ["D:/a.txt","D:/b.png"]。
func cbWriteFiles(c *gin.Context) {
	var paths []string
	if err := c.ShouldBindJSON(&paths); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	if err := cb.WriteFiles(paths); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

func inset(c *gin.Context) {
	err := tools.Paste()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

// ---------- 本地存储 ----------

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
