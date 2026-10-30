//go:build windows

// Package cb 基于 Win32 剪贴板 API 提供剪贴板读写，零第三方依赖。
//
// 支持的格式：
//   - 文本：CF_UNICODETEXT（UTF-16）
//   - 图片：CF_DIB 与 PNG 字节互转（仅支持 24/32 位未压缩位图）
//   - 富文本：CF_HTML（"HTML Format" 注册格式，含标准头部偏移）与 RTF（"Rich Text Format" 注册格式）
//   - 文件：CF_HDROP（资源管理器可识别的文件列表，粘贴即复制文件）
//
// 用法示例：
//
//	// 写入
//	cb.WriteText("你好")
//	cb.WriteImage(pngBytes)
//	cb.WriteHTML("<b>加粗</b>")
//	cb.WriteRTF(rtfBytes)
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
	"changeme/global"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// selfLock 记录本程序最近一次写入剪贴板完成后的剪贴板序列号。
// 写入成功后由 writeFormats 更新；Watch 检测到变化时若序列号与它相同，
// 说明这次变化是本程序自己造成的（写入目标内容或恢复原内容），直接跳过不通知。
var selfLock atomic.Uint32

func init() {
	type resCb struct {
		HasText  bool     `json:"hasText"` // 含纯文本
		Text     string   `json:"text"`
		HasImage bool     `json:"hasImage"` // 含图片
		Image    []byte   `json:"image"`
		HasHTML  bool     `json:"hasHTML"` // 含 HTML 富文本
		HTML     *HTML    `json:"HTML"`
		HasRTF   bool     `json:"hasRTF"` // 含 RTF 富文本
		RTF      []byte   `json:"RTF"`
		HasFiles bool     `json:"hasFiles"` // 含文件列表
		Files    []string `json:"files"`
	}

	application.RegisterEvent[resCb]("cb-change")

	go func() {
		time.Sleep(1 * time.Second)
		recordChan, _ := Watch(context.Background(), 500*time.Millisecond)
		for rec := range recordChan {
			rc := resCb{
				HasText:  rec.HasText,
				HasImage: rec.HasImage,
				HasHTML:  rec.HasHTML,
				HasRTF:   rec.HasRTF,
				HasFiles: rec.HasFiles,
			}

			// 图片类型可能包含 html 标签和 file 文件，需优先级排序减少图片的内存占用
			// 但是不保留图片又比较难处理，讨厌
			switch {
			case rc.HasHTML:
				rc.HTML, _ = rec.GetHTML()
				rc.Text = rec.GetText()
				//fallthrough
			case rc.HasFiles:
				rc.Files, _ = rec.GetFiles()
				fallthrough
			case rc.HasImage:
				rc.Image, _ = rec.GetImage()
			case rc.HasRTF:
				rc.RTF, _ = rec.GetRTF()
				rc.Text = rec.GetText()
			case rc.HasText:
				rc.Text = rec.GetText()
			}

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
	HasText  bool // 含纯文本
	HasImage bool // 含图片
	HasHTML  bool // 含 HTML 富文本
	HasRTF   bool // 含 RTF 富文本
	HasFiles bool // 含文件列表

	rawText  []byte // CF_UNICODETEXT 原始 UTF-16 字节
	rawImage []byte // CF_DIB 原始字节
	rawPNG   []byte // "PNG" 注册格式的现成 PNG 字节（部分来源会同时提供）
	rawHTML  []byte // "HTML Format" 原始字节
	rawRTF   []byte // "Rich Text Format" 原始字节
	rawFiles []byte // CF_HDROP 原始字节
}

// Read 读取当前剪贴板内容并返回快照。
// 任一格式不存在时其 Has* 字段为 false，对应 Get* 方法返回 ErrNoData。
func Read() (*Record, error) {
	r := &Record{}
	if err := withClipboard(func() error {
		r.rawText = readLocked(cfUnicodeText)
		r.rawImage = readLocked(cfDIB)
		r.rawPNG = readLocked(formatPNG)
		r.rawHTML = readLocked(formatHTML)
		r.rawRTF = readLocked(formatRTF)
		r.rawFiles = readLocked(cfHDROP)
		return nil
	}); err != nil {
		return nil, err
	}
	r.HasText = r.rawText != nil
	r.HasImage = r.rawImage != nil || r.rawPNG != nil
	r.HasHTML = r.rawHTML != nil
	r.HasRTF = r.rawRTF != nil
	r.HasFiles = r.rawFiles != nil
	return r, nil
}

// GetText 返回快照中的纯文本；无文本时返回空字符串。
func (r *Record) GetText() string {
	if !r.HasText {
		return ""
	}
	return utf16FromBytes(r.rawText)
}

// GetImage 返回快照中图片的 PNG 编码字节。
// 优先使用来源直接提供的 "PNG" 格式（零转换），没有再用 CF_DIB 解码转换。
func (r *Record) GetImage() ([]byte, error) {
	if r.rawPNG != nil {
		return r.rawPNG, nil
	}
	if !r.HasImage {
		return nil, ErrNoData
	}
	return decodeDIB(r.rawImage)
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
	return parseCFHTML(r.rawHTML)
}

// GetRTF 返回快照中 RTF 原始字节。
func (r *Record) GetRTF() ([]byte, error) {
	if !r.HasRTF {
		return nil, ErrNoData
	}
	return r.rawRTF, nil
}

// GetFiles 返回快照中的文件路径列表（绝对路径）。
func (r *Record) GetFiles() ([]string, error) {
	if !r.HasFiles {
		return nil, ErrNoData
	}
	return parseHDROP(r.rawFiles)
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

// WriteImage 将 PNG 编码的图片写入剪贴板，同时写入 "PNG" 原始格式与 CF_DIB，
// 支持哪种格式的程序（浏览器、图像软件等）就取哪种，无需接收方自行转换。
func WriteImage(pngData []byte) error {
	dib, err := encodeDIB(pngData)
	if err != nil {
		return err
	}
	return writeFormats(map[uint32][]byte{
		formatPNG: pngData,
		cfDIB:     dib,
	})
}

// WriteHTML 将 HTML 片段按 CF_HTML 标准格式写入剪贴板（头部偏移自动计算）
func WriteHTML(fragment string) error {
	frag := []byte(fragment)
	prefix := []byte("<html>\r\n<body>\r\n<!--StartFragment-->")
	suffix := []byte("<!--EndFragment-->\r\n</body>\r\n</html>")
	tpl := "Version:0.9\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n"
	head := fmt.Sprintf(tpl, 0, 0, 0, 0)
	startHTML := len(head)
	startFragment := startHTML + len(prefix)
	endFragment := startFragment + len(frag)
	endHTML := endFragment + len(suffix)
	head = fmt.Sprintf(tpl, startHTML, endHTML, startFragment, endFragment)

	blob := make([]byte, 0, endHTML)
	blob = append(blob, head...)
	blob = append(blob, prefix...)
	blob = append(blob, frag...)
	blob = append(blob, suffix...)
	return writeFormat(formatHTML, blob)
}

// WriteRTF 将 RTF 原始字节写入剪贴板
func WriteRTF(rtf []byte) error {
	return writeFormat(formatRTF, rtf)
}

// WriteFiles 将文件列表写入剪贴板，相对路径会转为绝对路径。
// 注意：仅写入文件路径列表，粘贴方（如资源管理器）会按复制文件处理；
// 若要支持"剪切"语义还需配合 DragLeave/降效语义消息，此处未实现。
func WriteFiles(paths []string) error {
	if len(paths) == 0 {
		return errors.New("cb: 文件列表为空")
	}
	var buf bytes.Buffer
	hdr := make([]byte, 20) // DROPFILES：pFiles(4) + pt.x(4) + pt.y(4) + fNC(4) + fWide(4)
	le.PutUint32(hdr[0:4], 20)
	le.PutUint32(hdr[16:20], 1) // fWide = 1，文件名为 UTF-16
	buf.Write(hdr)
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return fmt.Errorf("cb: 路径 %q 转换失败: %w", p, err)
		}
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("cb: 文件 %q 不存在: %w", abs, err)
		}
		u, err := syscall.UTF16FromString(abs)
		if err != nil {
			return fmt.Errorf("cb: 路径 %q 包含非法字符: %w", p, err)
		}
		buf.Write(utf16Bytes(u))
	}
	buf.Write([]byte{0, 0}) // 列表结尾的额外空字符
	return writeFormat(cfHDROP, buf.Bytes())
}

// WriteAll 将一条记录的多格式数据一次会话同时写入剪贴板
// （文本 + 图片 + HTML + RTF + 文件可同在场，粘贴方按需取最优格式），
// 典型用法是传入 Read() 得到的快照实现"粘贴回原条目"。全空记录返回错误。
func WriteAll(rec *Record) error {
	if rec == nil {
		return errors.New("cb: 记录为空")
	}
	return writeFormats(map[uint32][]byte{
		cfUnicodeText: rec.rawText,
		cfDIB:         rec.rawImage,
		formatPNG:     rec.rawPNG,
		formatHTML:    rec.rawHTML,
		formatRTF:     rec.rawRTF,
		cfHDROP:       rec.rawFiles,
	})
}
