package cb

import (
	"fmt"
	"regexp"
	"scb/tools"
	"strings"
)

// 仅限于剪贴板的图片使用，包含缩略图和原图和图片url
// 复制的时候一般会触发几种包含图片的情况
// 1.纯图片，不包含任何其他类型
// 2.网络图片+文件+图片 可能是<img src="file:///C:/Users/……60981.png">使用文件协议的图片
// 3.文件+图片 文件协议
//
// BData和Burl 分别表示原图和原图的url【永远只存在一个】，如果是文件或者网络图片，则BData为空，纯图片则Burl为空

const URL = "http://wails.localhost:9245/wg"

var ImgArray = tools.NewLRUArray(100)

type cbImage struct {
	SUrl string `json:"sUrl"` // 缩略图片url，网络url可能是别人的也可能是我自己的
	BUrl string `json:"bUrl"` // 原图片url
}

type LuaArrImage struct {
	Id    string `json:"id"`
	SData []byte `json:"sData"` // 缩略图，任何情况下都存在
	BData []byte `json:"bData"` // 原图，只有在一次记录没有文件/网络图片时才有
}

func (l *LuaArrImage) GetID() string { return l.Id }

// 把一个图片生成缩略图和原图和图片url
func generateImage(r *Record) (cbImage, error) {
	i := cbImage{}
	li := LuaArrImage{}

	// 处理缩略预览图片
	pngimg, err := decodeDIB(r.RawImage)
	if err != nil {
		fmt.Println("无法解码图片:", err.Error())
	}
	sl, err := tools.CompressImg(pngimg, tools.CompressConfig{MaxWidth: 200})
	if err != nil {
		return cbImage{}, err
	}
	li.SData = sl
	i.SUrl = URL + "/image/getSimg/" + r.Id // 缩略图的url，内部请求

	// 有图片又有文件，肯定用的是临时文件，可以丢弃原图的内容
	if r.HasFiles {
		files, _ := r.GetFiles()
		npath := strings.Replace(files[0], ":", "", 1) // 把文件路径中的冒号替换为空字符串 D:\ -> D\
		i.BUrl = URL + "/file/" + npath
	}

	if r.HasHTML && i.BUrl == "" {
		h, _ := r.GetHTML()
		hsrc := imgSrcFromHTML(h.Fragment)
		if strings.HasPrefix(hsrc, "file://") { // 如果是本地协议的
			nsrc := strings.Replace(hsrc, "file://", URL+"/file", 1)
			i.BUrl = URL + "/file" + strings.Replace(nsrc, ":", "", 1) // 把文件路径中的冒号替换为空字符串 D:\ -> D\
		}
	}

	if i.BUrl == "" { // 没有图片url就写入原图，这两永远是互斥的
		li.BData = pngimg
	}

	li.Id = r.Id
	ImgArray.Add(li)

	return i, nil
}

// imgSrcRe 匹配 HTML 片段里的 <img ... src="...">，双引号单引号都收
var imgSrcRe = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)

// imgSrcFromHTML 返回片段中第一个 img 的 src 原始值，没有图片时返回空字符串
func imgSrcFromHTML(fragment string) string {
	m := imgSrcRe.FindStringSubmatch(fragment)
	if m == nil {
		return ""
	}
	return m[1]
}
