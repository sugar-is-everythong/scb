//go:build windows

// Package cb 基于 Win32 剪贴板 API 提供剪贴板读写，零第三方依赖。
//
// 支持的格式：
//   - 文本：CF_UNICODETEXT（UTF-16）
//   - 图片：CF_DIB 与 PNG 字节互转（仅支持 24/32 位未压缩位图）
//   - 富文本：CF_HTML（"HTML Format" 注册格式，含标准头部偏移）
//   - 文件：CF_HDROP（资源管理器可识别的文件列表，粘贴即复制文件）
//
// 用法示例：
//
//	// 写入
//	cb.WriteText("你好")
//	cb.WriteImage(pngBytes)
//	cb.WriteHTML(&cb.HTML{Fragment: "<b>加粗</b>"})
//	cb.WriteFiles([]string{`D:\a.txt`, `D:\b.png`})
//	cb.WriteAll(rec) // 多格式一次写入，如"粘贴回原条目"
//
//	// 读取：一次快照，再按需取值
//	rec, _ := cb.Read()
//	if rec.HasText { fmt.Println(rec.GetText()) }
//	if rec.HasImage { pngData, _ := rec.GetImage() }
//	if rec.HasHTML { html, _ := rec.GetHTML() }
//	if rec.HasFiles { files, _ := rec.GetFiles() }
package cb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"scb/global"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// selfLock 记录本程序最近一次写入剪贴板完成后的剪贴板序列号。
// 写入成功后由 writeFormats 更新；Watch 检测到变化时若序列号与它相同，
// 说明这次变化是本程序自己造成的（写入目标内容或恢复原内容），直接跳过不通知。
var selfLock atomic.Uint32

type RecordJs struct {
	Id string `json:"id"`

	HasText  bool     `json:"hasText"` // 含纯文本
	Text     string   `json:"text"`
	HasImage bool     `json:"hasImage"` // 含图片
	Image    cbImage  `json:"image"`
	HasHTML  bool     `json:"hasHTML"` // 含 HTML 富文本
	HTML     *HTML    `json:"HTML"`
	HasFiles bool     `json:"hasFiles"` // 含文件列表
	Files    []string `json:"files"`
}

func init() {

	application.RegisterEvent[RecordJs]("cb-change")

	go func() {
		time.Sleep(1 * time.Second)
		recordChan, _ := Watch(context.Background(), 500*time.Millisecond)
		for rec := range recordChan {
			rc := rec.transform() // 转换为 JS 可用的结构体并发送事件

			fmt.Printf("HasFiles:%t,\n", rec.HasFiles)
			fmt.Printf("HasText:%t\n", rec.HasText)
			fmt.Printf("HasHTML:%t\n", rec.HasHTML)
			fmt.Printf("HasImage:%t\n\n", rec.HasImage)
			global.App.Event.Emit("cb-change", rc)
		}
	}()
}

// ErrNoData 表示剪贴板中不存在该格式的数据
var ErrNoData = errors.New("cb: 剪贴板中不存在该格式的数据")

// 读取：快照式

// Record 是某一时刻剪贴板内容的快照，由 Read() 一次性读出。
// 先用 Has* 字段判断快照里有哪些数据，再调用对应的 Get* 方法取出。
type Record struct {
	Id string `json:"id"`

	HasText  bool // 含纯文本
	HasImage bool // 含图片
	HasHTML  bool // 含 HTML 富文本
	HasFiles bool // 含文件列表

	RawText  []byte // CF_UNICODETEXT 原始 UTF-16 字节
	RawImage []byte // CF_DIB 原始字节
	RawPNG   []byte // "PNG" 注册格式的现成 PNG 字节（部分来源会同时提供）
	RawHTML  []byte // "HTML Format" 原始字节
	RawFiles []byte // CF_HDROP 原始字节
}

// Read 读取当前剪贴板内容并返回快照。
// 任一格式不存在时其 Has* 字段为 false，对应 Get* 方法返回 ErrNoData。
func Read() (*Record, error) {
	r := &Record{}
	if err := withClipboard(func() error {
		r.RawText = readLocked(cfUnicodeText)
		r.RawImage = readLocked(cfDIB)
		r.RawPNG = readLocked(formatPNG)
		r.RawHTML = readLocked(formatHTML)
		r.RawFiles = readLocked(cfHDROP)
		return nil
	}); err != nil {
		return nil, err
	}
	r.HasText = r.RawText != nil
	r.HasImage = r.RawImage != nil || r.RawPNG != nil
	r.HasHTML = r.RawHTML != nil
	r.HasFiles = r.RawFiles != nil

	r.Id = newId()
	return r, nil
}

// GetText 返回快照中的纯文本；无文本时返回空字符串。
func (r *Record) GetText() string {
	if !r.HasText {
		return ""
	}
	return utf16FromBytes(r.RawText)
}

// GetImage 返回快照中图片的 PNG 编码字节。
// 优先使用来源直接提供的 "PNG" 格式（零转换），没有再用 CF_DIB 解码转换。
func (r *Record) GetImage() ([]byte, error) {
	if r.RawPNG != nil {
		return r.RawPNG, nil
	}
	if !r.HasImage {
		return nil, ErrNoData
	}
	return decodeDIB(r.RawImage)
}

// HTML 是从剪贴板 CF_HTML 解析出的结果：既含纯净正文片段，也含头部元信息。
type HTML struct {
	Fragment  string // 纯净的 HTML 正文片段（已剥离头部与 <html>/<body> 外壳）
	SourceURL string // 来源网址（头部 SourceURL 字段，可能为空）
	Version   string // CF_HTML 版本号（如 "0.9"、"1.0"）
}

// GetHTML 解析并返回快照中的 CF_HTML：同时给出纯净正文片段与 SourceURL 等头部元信息。
func (r *Record) GetHTML() (*HTML, error) {
	if !r.HasHTML {
		return nil, ErrNoData
	}
	return parseCFHTML(r.RawHTML)
}

// GetFiles 返回快照中的文件路径列表（绝对路径）。
func (r *Record) GetFiles() ([]string, error) {
	if !r.HasFiles {
		return nil, ErrNoData
	}
	return parseHDROP(r.RawFiles)
}

// ==================== 监控 ====================

// Watch 监控剪贴板内容变化，每当剪贴板更新就把新快照发送到返回的 channel。
//
// 监控从启动后开始，只报告之后发生的变化；需要当前内容请直接调用 Read()。
// 重新复制相同内容也会触发（序列号变化即通知），未做内容去重。
// interval 为轮询间隔，如 500*time.Millisecond（每秒两次）；实现上每轮仅对比
// 剪贴板序列号（GetClipboardSequenceNumber），不打开不锁定剪贴板，开销极小。
// 消费方处理较慢时发送会阻塞，监控随之暂停，恢复后直接读取最新快照，中间态不堆积。
// ctx 取消后监控停止并关闭 channel。
func Watch(ctx context.Context, interval time.Duration) (<-chan *Record, error) {
	if interval <= 0 {
		return nil, errors.New("cb: 轮询间隔必须大于 0")
	}
	ch := make(chan *Record)
	go func() {
		defer close(ch)
		last := currentSequence()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			if cur := currentSequence(); cur != last {
				last = cur
				if cur == selfLock.Load() {
					continue // 本程序自己写入造成的变化，不通知消费方
				}
				rec, err := Read()
				if err != nil {
					continue // 打开失败等临时错误，下一轮重试
				}
				select {
				case ch <- rec:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

// ==================== 写入 ====================

// WriteText 将文本写入剪贴板
func WriteText(s string) error {
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return fmt.Errorf("cb: 文本包含非法字符: %w", err)
	}
	return writeFormat(cfUnicodeText, utf16Bytes(u))
}

// WriteImage 将任意格式图片（PNG/JPEG/GIF 等）写入剪贴板，同时写入 "PNG" 格式与 CF_DIB，
// 支持哪种格式的程序（浏览器、图像软件等）就取哪种，无需接收方自行转换。
// 如果原图是 PNG，直接写入原始字节（零转换）；否则统一转码成 PNG。
func WriteImage(imgData []byte) error {
	// 先解码一次
	src, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return fmt.Errorf("cb: 图片解码失败: %w", err)
	}

	// 生成 DIB
	dib, err := encodeDIBFromImage(src)
	if err != nil {
		return err
	}

	// 检测是否是 PNG：是的话直接用原始字节，否则转码
	var pngBytes []byte
	if isPNG(imgData) {
		pngBytes = imgData
	} else {
		var pngBuf bytes.Buffer
		if err := png.Encode(&pngBuf, src); err != nil {
			return fmt.Errorf("cb: PNG 编码失败: %w", err)
		}
		pngBytes = pngBuf.Bytes()
	}

	return writeFormats(map[uint32][]byte{
		formatPNG: pngBytes,
		cfDIB:     dib,
	})
}

// isPNG 检测图片数据是否是 PNG 格式（通过文件头魔数判断）
func isPNG(data []byte) bool {
	// PNG 文件头：\x89PNG\r\n\x1a\n（8 字节）
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	return len(data) >= 8 && bytes.Equal(data[:8], pngHeader)
}

// WriteHTML 将完整的 HTML 结构按 CF_HTML 标准格式写入剪贴板，头部偏移由 buildCFHTML 计算。
// Version 为空时补 "0.9"，SourceURL 为空则不写该行，因此允许调用方只给 Fragment。
func WriteHTML(h *HTML) error {
	if h == nil || h.Fragment == "" {
		return errors.New("cb: HTML 片段为空")
	}
	return writeFormat(formatHTML, buildCFHTML(h))
}

// WriteFiles 将文件列表写入剪贴板，相对路径会转为绝对路径。
// 注意：仅写入文件路径列表，粘贴方（如资源管理器）会按复制文件处理；
// 若要支持"剪切"语义还需配合 DragLeave/降效语义消息，此处未实现。
func WriteFiles(paths []string) error {
	data, err := encodeHDROP(paths)
	if err != nil {
		return err
	}
	return writeFormat(cfHDROP, data)
}

// WriteAll 将一条记录的多格式数据一次会话同时写入剪贴板
// （文本 + 图片 + HTML + 文件可同在场，粘贴方按需取最优格式），
// 典型用法是传入 Read() 得到的快照实现"粘贴回原条目"。全空记录返回错误。
func WriteAll(rec *Record) error {
	if rec == nil {
		return errors.New("cb: 记录为空")
	}
	return writeFormats(map[uint32][]byte{
		cfUnicodeText: rec.RawText,
		cfDIB:         rec.RawImage,
		formatPNG:     rec.RawPNG,
		formatHTML:    rec.RawHTML,
		cfHDROP:       rec.RawFiles,
	})
}

// ==================== JS ====================

func (r *Record) transform() RecordJs {
	rc := RecordJs{
		Id:       r.Id,
		HasText:  r.HasText,
		HasImage: r.HasImage,
		HasHTML:  r.HasHTML,
		HasFiles: r.HasFiles,
	}

	if rc.HasHTML {
		rc.HTML, _ = r.GetHTML()
	}
	if rc.HasText {
		rc.Text = r.GetText()
	}
	if rc.HasFiles {
		rc.Files, _ = r.GetFiles()
	}
	if rc.HasImage {
		img, err := generateImage(r)
		if err != nil {
			fmt.Println("生成图片失败:", err.Error())
		}
		rc.Image = img
	}

	return rc
}

func (r *RecordJs) Transform() (*Record, error) {
	rec := Record{
		Id:       r.Id,
		HasText:  r.HasText,
		HasImage: r.HasImage,
		HasHTML:  r.HasHTML,
		HasFiles: r.HasFiles,
		//RawHTML:  r.HTML, 额外转换，放在这里显示提醒我别忘了
		//RawImage: r.Image, // 这里不一定有内容
		//RawFiles: r.Files,
	}

	// RawText 的约定是 CF_UNICODETEXT 的 UTF-16 字节（与 Read 读到的、GetText 解的、
	// WriteAll 写回剪贴板的都一致），直接拿 Go string 的 UTF-8 字节会让取出时按
	// UTF-16 两字节一组误解成乱码，转化方式与 WriteText 保持一致。
	if r.HasText {
		u, err := syscall.UTF16FromString(r.Text)
		if err != nil {
			return nil, fmt.Errorf("cb: 文本包含非法字符: %w", err)
		}
		rec.RawText = utf16Bytes(u)
	}

	if r.HasImage {
		// 先判 BUrl：非空时内部 /file/ 地址读盘、外网链接下载（ResolveImageBytes 分流）；
		// BUrl 为空才是纯位图，退回内存 LRU 缓存。
		var imgBytes []byte
		if r.Image.BUrl != "" {
			data, err := ResolveImageBytes(r.Image.BUrl)
			if err != nil {
				return nil, fmt.Errorf("cb: 图片加载失败: %w", err)
			}
			imgBytes = data
		} else if img, has := ImgArray.Find(func(a any) bool {
			return r.Id == a.(LuaArrImage).Id
		}); has {
			imgBytes = img.(LuaArrImage).BData
		}

		if len(imgBytes) > 0 {
			// 原图是 PNG 直接用作 "PNG" 格式字节，其余转码（与 WriteImage 的约定一致）
			if isPNG(imgBytes) {
				rec.RawPNG = imgBytes
			} else {
				src, _, err := image.Decode(bytes.NewReader(imgBytes))
				if err != nil {
					return nil, fmt.Errorf("解析图片失败: %w", err)
				}
				var pngBuf bytes.Buffer
				if err := png.Encode(&pngBuf, src); err != nil {
					return nil, fmt.Errorf("cb: PNG 编码失败: %w", err)
				}
				rec.RawPNG = pngBuf.Bytes()
			}

			// 把png转成dib
			DIBimg, _, err := image.Decode(bytes.NewReader(rec.RawPNG))
			if err != nil {
				return nil, fmt.Errorf("解析图片失败: %w", err)
			}
			dib, err := encodeDIBFromImage(DIBimg)
			if err != nil {
				return nil, fmt.Errorf("DIB 编码失败: %w", err)
			}
			rec.RawImage = dib
		}
	}

	if r.HasHTML && r.HTML != nil {
		rec.RawHTML = buildCFHTML(r.HTML)
	}

	if r.HasFiles {
		hdrop, err := encodeHDROP(r.Files)
		if err != nil {
			return nil, err
		}
		rec.RawFiles = hdrop
	}

	return &rec, nil
}

func newId() string {
	return fmt.Sprintf("%d", time.Now().UnixMilli())
}
