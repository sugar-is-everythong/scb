package cb

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"scb/global"
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

// ==================== 图片持久化 ====================

// localFileURL 把数据目录下的绝对路径转成 cb.URL+"/file/..." 可访问地址：
// 盘符去冒号当第一段、分隔符统一正斜杠，与 fileGroup 的 StaticFS 路由约定一致。
func localFileURL(abs string) string {
	abs = filepath.Clean(abs)
	rest := strings.ReplaceAll(abs[2:], "\\", "/") // 跳过 "D:" 两段
	return URL + "/file/" + abs[:1] + rest
}

// ResolveImageBytes 把 BUrl 解析成图片原始字节：
// cb.URL 开头的内部地址反解成磁盘路径读文件，其余一律按外网链接下载。
func ResolveImageBytes(u string) ([]byte, error) {
	if strings.HasPrefix(u, URL+"/file/") {
		p := strings.TrimPrefix(u, URL+"/file/")
		if len(p) < 2 {
			return nil, fmt.Errorf("cb: 本地图片地址非法: %s", u)
		}
		p = p[:1] + ":" + p[1:] // 首段盘符补回冒号 D/.. -> D:/..
		return os.ReadFile(filepath.FromSlash(p))
	}
	return tools.GetWebImage(u)
}

// localURLDiskPath 把内部 /file/ 地址反解成磁盘绝对路径，非内部地址返回 false
func localURLDiskPath(u string) (string, bool) {
	if !strings.HasPrefix(u, URL+"/file/") {
		return "", false
	}
	p := strings.TrimPrefix(u, URL+"/file/")
	if len(p) < 2 {
		return "", false
	}
	return filepath.Clean(filepath.FromSlash(p[:1] + ":" + p[1:])), true
}

// isUnderLocal 判断路径是否在本程序数据目录下（大小写不敏感，跨盘符返回 false）
func isUnderLocal(p string) bool {
	rel, err := filepath.Rel(global.Local, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// sniffExt 按图片字节魔数给出后缀，认不出来返回空串
func sniffExt(data []byte) string {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], pngHeader):
		return ".png"
	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8:
		return ".jpg"
	case len(data) >= 4 && string(data[:4]) == "GIF8":
		return ".gif"
	case len(data) >= 2 && data[0] == 'B' && data[1] == 'M':
		return ".bmp"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return ".webp"
	}
	return ""
}

// pickExt 优先用魔数认出的后缀，认不出用默认值
func pickExt(fallback string, data []byte) string {
	if e := sniffExt(data); e != "" {
		return e
	}
	return fallback
}

// persistImage 钉住时把记录的内存图片落到数据目录，并把 SUrl/BUrl 改写成磁盘地址。
// "来啥存啥"指字节原样保存不转码不查重，后缀跟随来源真实格式；
// 指向别的软件目录（如 QQ 临时目录）的文件一律拷进来，生命周期必须自己管。
func persistImage(r *RecordJs) error {
	if !r.HasImage || r.Id == "" {
		return nil
	}

	var cache LuaArrImage
	if img, has := ImgArray.Find(func(a any) bool {
		return a.(LuaArrImage).Id == r.Id
	}); has {
		cache = img.(LuaArrImage)
	}

	if err := os.MkdirAll(filepath.Join(global.Local, "imgs"), 0755); err != nil {
		return err
	}

	// 原图：文件名 = 记录 id + 来源真实后缀
	switch {
	case r.Image.BUrl == "": // 纯位图，字节只在内存 LRU 里
		if len(cache.BData) > 0 {
			ext := pickExt(".png", cache.BData)
			if err := saveLocalImage(r, r.Id+ext, cache.BData); err != nil {
				return err
			}
		} else {
			// 缓存已淘汰又无现成地址，图片成员无从持久化，只能丢；记录其余成员照常保存
			fmt.Println("cb: 图片字节不在缓存中，该条记录图片未持久化, id:", r.Id)
		}

	case strings.HasPrefix(r.Image.BUrl, URL+"/file/"):
		src, ok := localURLDiskPath(r.Image.BUrl)
		switch {
		case !ok, isUnderLocal(src): // 已在自家目录或非规范地址，保持原样
		default:
			// 外部软件的文件（如 QQ 临时目录）：拷进 imgs 按 <id>+原后缀命名，BUrl 改指副本
			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("cb: 读取外部图片失败: %w", err)
			}
			ext := strings.ToLower(filepath.Ext(src))
			if ext == "" {
				ext = pickExt(".png", data)
			}
			dst := imgPath(r.Id + ext)
			if err := saveLocalImage(r, r.Id+ext, data); err != nil {
				return err
			}

			// files 列表里指向这个源路径的项一并改成副本，粘贴不再依赖来源文件
			for i, f := range r.Files {
				if strings.EqualFold(filepath.Clean(f), src) {
					r.Files[i] = dst
				}
			}
		}
	}
	// 外网 BUrl 只有链接没有文件，原样存链接

	// 缩略图：实时版 SUrl 指向 /image/getSimg，靠内存缓存重启即失效，一律落盘改写
	if len(cache.SData) > 0 {
		p := imgPath(r.Id + ".thumb" + pickExt(".jpg", cache.SData))
		if err := os.WriteFile(p, cache.SData, 0644); err != nil {
			return fmt.Errorf("cb: 缩略图落盘失败: %w", err)
		}
		r.Image.SUrl = localFileURL(p)
	}
	return nil
}

// imgPath 图片文件的统一落点：local/imgs/ 下并排平铺，文件名由调用方给定
func imgPath(name string) string {
	return filepath.Join(global.Local, "imgs", name)
}

// saveLocalImage 把字节写入 imgs 目录并把 BUrl 改写成可访问地址
func saveLocalImage(r *RecordJs, name string, data []byte) error {
	p := imgPath(name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("cb: 原图落盘失败: %w", err)
	}
	r.Image.BUrl = localFileURL(p)
	return nil
}

// persistFiles 把钉住记录 files 里指向外部路径的真实文件拷进数据目录
// files/<id>/ 子目录（保留原文件名，粘出去的文件名不变）并改写路径。
// 已在自家目录的跳过；目录型条目不整体搬运，粘贴语义本就是复制目录，保持原路径。
func persistFiles(r *RecordJs) error {
	if !r.HasFiles || r.Id == "" {
		return nil
	}
	for i, f := range r.Files {
		src := filepath.Clean(f)
		if isUnderLocal(src) {
			continue
		}
		st, err := os.Stat(src)
		if err != nil {
			// 源文件已不存在（被来源软件清理）无法拷，保留原路径、不阻断钉住
			fmt.Println("cb: 文件不可访问，保留原路径:", src, err.Error())
			continue
		}
		if st.IsDir() {
			continue
		}
		dst, err := copyUnderLocal(r.Id, src)
		if err != nil {
			fmt.Println("cb: 文件拷贝失败，保留原路径:", src, err.Error())
			continue
		}
		r.Files[i] = dst
	}
	return nil
}

// copyUnderLocal 把外部文件拷到数据目录 files/<id>/ 下并保留原名，返回新路径。
// 流式拷贝，不让大文件整体进内存。
func copyUnderLocal(id, src string) (string, error) {
	dir := filepath.Join(global.Local, "files", id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, filepath.Base(src))
	sf, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("cb: 文件拷贝失败: %w", err)
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("cb: 文件拷贝失败: %w", err)
	}
	if _, err := io.Copy(df, sf); err != nil {
		_ = df.Close()
		return "", fmt.Errorf("cb: 文件拷贝失败: %w", err)
	}
	return dst, df.Close()
}

// removeLocalImageFiles 删除钉住时写在数据目录里的图片与文件副本。
// 现行图片在 imgs/<id>+后缀；早期直接平在 local 根目录与 files/<id>/ 的旧版布局一并清。
// 只碰本程序自己写的文件，外部真实文件与网络链接不受影响。
func removeLocalImageFiles(id string) {
	if id == "" {
		return
	}
	for _, dir := range []string{filepath.Join(global.Local, "imgs"), global.Local} {
		// id 要求后接点号，不会误中更长 id 的文件
		matches, _ := filepath.Glob(filepath.Join(dir, id+".*"))
		for _, m := range matches {
			_ = os.Remove(m)
		}
		_ = os.Remove(filepath.Join(dir, id)) // 无后缀方案的遗留文件
	}
	_ = os.RemoveAll(filepath.Join(global.Local, "files", id))
}
