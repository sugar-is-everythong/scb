package service

import (
	"net/http"
	"scb/cb"
	"scb/global"
	"scb/tools"

	"github.com/gin-gonic/gin"
)

// recordInfo 是 cb.Record 的 JSON 视图：只暴露各格式有无的标志。
type recordInfo struct {
	HasText  bool `json:"hasText"`
	HasImage bool `json:"hasImage"`
	HasHTML  bool `json:"hasHTML"`
	HasFiles bool `json:"hasFiles"`
}

// infoOf 将快照转为 JSON 视图
func infoOf(r *cb.Record) recordInfo {
	return recordInfo{
		HasText:  r.HasText,
		HasImage: r.HasImage,
		HasHTML:  r.HasHTML,
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

// cbWriteText POST /cb/text：请求体为 JSON 字符串（前端 cbInset 统一对 body 做了 JSON.stringify）。
// 必须用 ShouldBindJSON 反序列化取出真实文本，直接拿原始请求体会把引号与 \r\n\t 转义符原样写进剪贴板。
func cbWriteText(c *gin.Context) {
	var s string
	if err := c.ShouldBindJSON(&s); err != nil {
		FailWithMsg(c, "请求参数解析"+err.Error())
		return
	}
	if err := cb.WriteText(s); err != nil {
		FailWithMsg(c, err.Error())
		return
	}
	OK_(c)
}

// cbWriteImage POST 通过id去图片的缓存里拿去，因为前端也没有二进制
func cbWriteImage(c *gin.Context) {
	var q struct {
		Id   string `json:"id"`
		BUrl string `json:"bUrl"`
	}

	if err := c.ShouldBindJSON(&q); err != nil {
		FailWithMsg(c, "参数解析错误"+err.Error())
		return
	}

	// 有 BUrl 就先解析成字节：内部 cb.URL/file/ 地址读盘，其余按外网链接下载，
	// 分流逻辑统一收进 cb.ResolveImageBytes，与 Transform 走同一条回源路径
	if q.BUrl != "" {
		img, err := cb.ResolveImageBytes(q.BUrl)
		if err != nil {
			FailWithMsg(c, err.Error())
			return
		}

		if err := cb.WriteImage(img); err != nil {
			FailWithMsg(c, err.Error())
			return
		}

		OK_(c)
		return
	}

	if img, has := cb.ImgArray.Find(func(a any) bool { return a.(cb.LuaArrImage).Id == q.Id }); q.Id != "" && has {
		if err := cb.WriteImage(img.(cb.LuaArrImage).BData); err != nil {
			FailWithMsg(c, err.Error())
			return
		}
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

// cbWriteHTML POST /cb/html：虽然叫写入 HTML，但是是写整个支持的结构，因为只有html片段有的软件写不进去
// 虽说是整个结构，但是目前也就包含 html和text 两个字段，别的暂时没发现有回写的必要
func cbWriteHTML(c *gin.Context) {
	var rj = cb.RecordJs{}

	err := c.ShouldBindJSON(&rj)
	if err != nil {
		FailWithMsg(c, "参数处理错误"+err.Error())
		return
	}

	r, err := rj.Transform()
	if err != nil {
		FailWithMsg(c, "结构转换错误"+err.Error())
		return
	}

	if err := cb.WriteAll(r); err != nil {
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
	mw, _ := global.App.Window.GetByName("粘一下")
	hwnd := uintptr(mw.NativeWindow())
	// 用 HideNoActivate 与 ShowNoActivate 配对
	tools.HideNoActivate(hwnd)
	// 点击面板时 WebView2 会程序化激活父窗口（WS_EX_NOACTIVATE 拦不住），
	// 这里显式把前台还给原窗口，再等待切换完成后粘贴。
	tools.FocusWindow(global.Foreground)
	tools.WaitForegroundWindow(global.Foreground, 1000)

	err := tools.Paste()
	if err != nil {
		FailWithMsg(c, err.Error())
		return
	}

	OK_(c)
}

// cbHide POST /cb/hide：隐藏面板。
// 前端不能调 Wails 的 Window.Hide()（它会调 chromium.Hide() 并重置 windowShown/showRequested，
// 导致下次 ShowNoActivate 显示时状态错乱），统一走这里用 HideNoActivate。
func cbHide(c *gin.Context) {
	mw, _ := global.App.Window.GetByName("粘一下")
	hwnd := uintptr(mw.NativeWindow())
	tools.HideNoActivate(hwnd)
	OK_(c)
}
