//go:build windows

// 本文件是 cb 包的 Win32 底层实现：剪贴板打开/读写、全局堆内存、
// 以及 CF_DIB / CF_HTML / CF_HDROP 的编解码。对外 API 见 cb.go。
package cb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

// 标准剪贴板格式编号（Win32 固定值）
const (
	cfUnicodeText uint32 = 13 // 文本
	cfDIB         uint32 = 8  // 设备无关位图（图片）
	cfHDROP       uint32 = 15 // 文件列表
)

const gmemMoveable uintptr = 0x0002

// 下面用 syscall.DLL + MustFindProc 取得 *Proc，再统一通过 syscall.SyscallN(proc.Addr(), ...) 调用。
var (
	user32   = syscall.MustLoadDLL("user32.dll")
	kernel32 = syscall.MustLoadDLL("kernel32.dll")

	pOpenClipboard    = user32.MustFindProc("OpenClipboard")
	pCloseClipboard   = user32.MustFindProc("CloseClipboard")
	pEmptyClipboard   = user32.MustFindProc("EmptyClipboard")
	pGetClipboardData = user32.MustFindProc("GetClipboardData")
	pSetClipboardData = user32.MustFindProc("SetClipboardData")
	pIsFormatAvail    = user32.MustFindProc("IsClipboardFormatAvailable")
	pRegisterFormat   = user32.MustFindProc("RegisterClipboardFormatW")
	pGetClipboardSeq  = user32.MustFindProc("GetClipboardSequenceNumber")
	pGlobalAlloc      = kernel32.MustFindProc("GlobalAlloc")
	pGlobalFree       = kernel32.MustFindProc("GlobalFree")
	pGlobalLock       = kernel32.MustFindProc("GlobalLock")
	pGlobalUnlock     = kernel32.MustFindProc("GlobalUnlock")
	pGlobalSize       = kernel32.MustFindProc("GlobalSize")
)

// 富文本与图片对应的注册格式，进程启动时注册一次
var (
	formatHTML = registerClipboardFormat("HTML Format")
	formatPNG  = registerClipboardFormat("PNG")
)

var le = binary.LittleEndian

// ---------- Win32 剪贴板基础封装 ----------

func osErr(op string, e error) error {
	if errno, ok := e.(syscall.Errno); ok && errno != 0 {
		return fmt.Errorf("cb: %s: %w", op, errno)
	}
	return fmt.Errorf("cb: %s: 调用失败", op)
}

func registerClipboardFormat(name string) uint32 {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := syscall.SyscallN(pRegisterFormat.Addr(), uintptr(unsafe.Pointer(p)))
	return uint32(r)
}

func formatAvailable(cf uint32) bool {
	r, _, _ := syscall.SyscallN(pIsFormatAvail.Addr(), uintptr(cf))
	return r != 0
}

// currentSequence 返回剪贴板序列号：剪贴板内容每次变化（含被清空）都会使它 +1。
// 仅读取系统计数器，不打开不锁定剪贴板，适合高频轮询做变化检测。
func currentSequence() uint32 {
	r, _, _ := syscall.SyscallN(pGetClipboardSeq.Addr())
	return uint32(r)
}

// withClipboard 打开剪贴板并执行 fn，结束后关闭。
//
// 剪贴板是线程亲和资源：OpenClipboard 与 CloseClipboard 必须在同一个 OS 线程上调用，
// 跨线程关闭会失败，剪贴板会一直停留在打开状态并绑定在原线程上，其他程序从此打不开它。
// Go 的 goroutine 在系统调用返回后可能被调度到另一个线程，因此这里锁定 OS 线程，
// 保证打开与关闭落在同一线程上。
//
// 剪贴板同一时刻只能被一个线程打开，其他应用占用时会短暂失败，故做少量重试。
func withClipboard(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var r uintptr
	var e error
	for range 10 {
		r, _, e = syscall.SyscallN(pOpenClipboard.Addr(), 0)
		if r != 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if r == 0 {
		return osErr("OpenClipboard", e)
	}
	// 后注册的 defer 先执行：CloseClipboard 一定在解锁线程之前、同一线程上完成
	defer closeClipboard()
	return fn()
}

// closeClipboard 关闭剪贴板。关闭失败会让剪贴板一直被占用且无法自愈，
// 因此重试几次并记录日志，不再静默丢弃返回值。
func closeClipboard() {
	for range 3 {
		if r, _, _ := syscall.SyscallN(pCloseClipboard.Addr()); r != 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	log.Println("cb: CloseClipboard 失败，剪贴板可能仍被占用")
}

// readLocked 在剪贴板已打开的前提下读取指定格式的原始字节；该格式不存在时返回 nil。
// 必须在 withClipboard 会话内调用，以保证整个 Read() 是对同一次剪贴板状态的快照。
func readLocked(cf uint32) []byte {
	if cf == 0 || !formatAvailable(cf) {
		return nil
	}
	h, _, _ := syscall.SyscallN(pGetClipboardData.Addr(), uintptr(cf))
	if h == 0 {
		return nil
	}
	size, _, _ := syscall.SyscallN(pGlobalSize.Addr(), h)
	if size == 0 {
		return nil
	}
	p, _, _ := syscall.SyscallN(pGlobalLock.Addr(), h)
	if p == 0 {
		return nil
	}
	defer syscall.SyscallN(pGlobalUnlock.Addr(), h)
	out := make([]byte, size)
	copy(out, unsafe.Slice((*byte)(unsafe.Pointer(p)), size))
	return out
}

// newGlobalMem 分配 Win32 全局堆内存并拷入 payload
func newGlobalMem(payload []byte) (uintptr, error) {
	h, _, e := syscall.SyscallN(pGlobalAlloc.Addr(), gmemMoveable, uintptr(len(payload)))
	if h == 0 {
		return 0, osErr("GlobalAlloc", e)
	}
	p, _, e := syscall.SyscallN(pGlobalLock.Addr(), h)
	if p == 0 {
		syscall.SyscallN(pGlobalFree.Addr(), h)
		return 0, osErr("GlobalLock", e)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(p)), len(payload)), payload)
	syscall.SyscallN(pGlobalUnlock.Addr(), h)
	return h, nil
}

// writeFormat 以指定格式写入剪贴板原始字节（单格式便捷入口）
func writeFormat(cf uint32, payload []byte) error {
	return writeFormats(map[uint32][]byte{cf: payload})
}

// writeFormats 在一次剪贴板会话内写入多个格式：Empty 一次、Set 多次，
// 使 text/html/图片 等多格式能同时在场，粘贴方按需取最优格式。
// 注意：写入前先清空剪贴板，会覆盖其他应用已有的剪贴板内容。
func writeFormats(items map[uint32][]byte) error {
	valid := make(map[uint32][]byte, len(items))
	for cf, data := range items {
		if cf != 0 && len(data) > 0 {
			valid[cf] = data
		}
	}
	if len(valid) == 0 {
		return errors.New("cb: 没有可写入的数据")
	}

	err := withClipboard(func() error {
		// 先分配好全部内存再清空剪贴板，中途失败时不破坏剪贴板原状
		handles := make(map[uint32]uintptr, len(valid))
		for cf, data := range valid {
			h, err := newGlobalMem(data)
			if err != nil {
				for _, hh := range handles {
					syscall.SyscallN(pGlobalFree.Addr(), hh)
				}
				return err
			}
			handles[cf] = h
		}
		if r, _, e := syscall.SyscallN(pEmptyClipboard.Addr()); r == 0 {
			for _, hh := range handles {
				syscall.SyscallN(pGlobalFree.Addr(), hh)
			}
			return osErr("EmptyClipboard", e)
		}
		for cf, h := range handles {
			// SetClipboardData 成功后内存归系统所有，不可再 GlobalFree
			if r, _, e := syscall.SyscallN(pSetClipboardData.Addr(), uintptr(cf), h); r == 0 {
				syscall.SyscallN(pGlobalFree.Addr(), h)
				return osErr("SetClipboardData", e)
			}
		}

		return nil
	})
	// 写入完成后再取序列号：EmptyClipboard 与每次 SetClipboardData 都会让它递增，
	// 取到的值才是本次写入完成后剪贴板的真实状态标识。
	selfLock.Store(currentSequence())

	return err
}

func utf16Bytes(u []uint16) []byte {
	if len(u) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&u[0])), len(u)*2)
}

func utf16FromBytes(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	u := unsafe.Slice((*uint16)(unsafe.Pointer(&b[0])), len(b)/2)
	return syscall.UTF16ToString(u)
}

// ---------- 图片编解码（PNG <-> CF_DIB） ----------

// encodeDIBFromImage 将 image.Image 编码为 32 位自顶向下 CF_DIB
func encodeDIBFromImage(src image.Image) ([]byte, error) {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("cb: 图片尺寸非法")
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, bounds.Min, draw.Src)

	dib := make([]byte, 40+w*h*4)
	le.PutUint32(dib[0:4], 40)                 // biSize
	le.PutUint32(dib[4:8], uint32(w))          // biWidth
	le.PutUint32(dib[8:12], uint32(int32(-h))) // biHeight 负值表示自顶向下
	le.PutUint16(dib[12:14], 1)                // biPlanes
	le.PutUint16(dib[14:16], 32)               // biBitCount
	le.PutUint32(dib[16:20], 0)                // biCompression = BI_RGB
	le.PutUint32(dib[20:24], uint32(w*h*4))    // biSizeImage
	i := 40
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			o := rgba.PixOffset(x, y)
			// DIB 内存序为 BGRA
			dib[i] = rgba.Pix[o+2]
			dib[i+1] = rgba.Pix[o+1]
			dib[i+2] = rgba.Pix[o]
			dib[i+3] = rgba.Pix[o+3]
			i += 4
		}
	}
	return dib, nil
}

// decodeDIB 将 CF_DIB 字节编码为 PNG。仅支持 24/32 位、BI_RGB 或 32 位 BI_BITFIELDS 的位图。
// 32 位的第 4 字节统一按不透明处理：BI_RGB 下它是保留位（很多来源填 0），
// BI_BITFIELDS 标准掩码也无 alpha 语义，若当 alpha 用会导致截图类来源整图透明。
func decodeDIB(dib []byte) ([]byte, error) {
	if len(dib) < 40 {
		return nil, errors.New("cb: 位图数据过短")
	}
	if le.Uint32(dib[0:4]) != 40 {
		return nil, errors.New("cb: 仅支持 40 字节的 BITMAPINFOHEADER")
	}
	w := int(int32(le.Uint32(dib[4:8])))
	hRaw := int32(le.Uint32(dib[8:12]))
	topDown := hRaw < 0
	h := int(hRaw)
	if h < 0 {
		h = -h
	}
	bpp := int(le.Uint16(dib[14:16]))
	comp := le.Uint32(dib[16:20])
	clrUsed := int(le.Uint32(dib[32:36]))

	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("cb: 不支持 %d 位色深（仅支持 24/32 位）", bpp)
	}
	dataOffset := 40
	switch {
	case comp == 0: // BI_RGB
		dataOffset += clrUsed * 4
	case comp == 3 && bpp == 32: // BI_BITFIELDS，后面跟 3 个通道掩码
		if len(dib) < 52 {
			return nil, errors.New("cb: 位图通道掩码不完整")
		}
		// 仅支持标准 BGRA 位序掩码
		if le.Uint32(dib[40:44]) != 0x00FF0000 ||
			le.Uint32(dib[44:48]) != 0x0000FF00 ||
			le.Uint32(dib[48:52]) != 0x000000FF {
			return nil, errors.New("cb: 不支持的通道掩码布局")
		}
		dataOffset += 12
	default:
		return nil, errors.New("cb: 不支持的位图压缩方式")
	}
	if w <= 0 || h <= 0 {
		return nil, errors.New("cb: 位图尺寸非法")
	}
	stride := w * bpp / 8
	if bpp == 24 {
		stride = (stride + 3) & ^3 // 行按 4 字节对齐
	}
	if len(dib) < dataOffset+stride*h {
		return nil, errors.New("cb: 位图像素数据不完整")
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		srcY := y
		if !topDown {
			srcY = h - 1 - y // BI_RGB 位图默认自底向上
		}
		row := dib[dataOffset+srcY*stride:]
		for x := 0; x < w; x++ {
			p := row[x*bpp/8:]
			o := img.PixOffset(x, y)
			img.Pix[o+2], img.Pix[o+1], img.Pix[o] = p[0], p[1], p[2] // BGR -> RGBA
			img.Pix[o+3] = 255                                        // 32 位第 4 字节不可信，统一按不透明处理
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("cb: PNG 编码失败: %w", err)
	}
	return buf.Bytes(), nil
}

// encodeHDROP 把文件路径列表编码成 CF_HDROP 格式的字节，可直接赋给 Record.RawFiles。
// 相对路径会转为绝对路径并规整，同时校验文件是否存在。
func encodeHDROP(paths []string) ([]byte, error) {
	if len(paths) == 0 {
		return nil, errors.New("cb: 文件列表为空")
	}
	var buf bytes.Buffer
	hdr := make([]byte, 20) // DROPFILES：pFiles(4) + pt.x(4) + pt.y(4) + fNC(4) + fWide(4)
	le.PutUint32(hdr[0:4], 20)
	le.PutUint32(hdr[16:20], 1) // fWide = 1，文件名为 UTF-16
	buf.Write(hdr)
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("cb: 路径 %q 转换失败: %w", p, err)
		}
		abs = filepath.Clean(abs)
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("cb: 文件 %q 不存在: %w", abs, err)
		}
		u, err := syscall.UTF16FromString(abs)
		if err != nil {
			return nil, fmt.Errorf("cb: 路径 %q 包含非法字符: %w", p, err)
		}
		buf.Write(utf16Bytes(u))
	}
	buf.Write([]byte{0, 0}) // 列表结尾的额外空字符
	return buf.Bytes(), nil
}

// ---------- 富文本 / 文件解析 ----------

// parseCFHTML 解析 CF_HTML 字节流：提取头部元信息（Version、SourceURL）与纯净 HTML 正文片段。
func parseCFHTML(b []byte) (*HTML, error) {
	// 头部由若干 "Key:Value\r\n" 行组成，字段名均为纯字母；遇到 HTML 正文即停止。
	header := map[string]string{}
	cur := 0
	for cur < len(b) {
		nl := bytes.Index(b[cur:], []byte("\r\n"))
		if nl < 0 {
			break
		}
		line := b[cur : cur+nl]
		colon := bytes.IndexByte(line, ':')
		if colon <= 0 {
			break
		}
		key := string(line[:colon])
		if !isHeaderKey(key) {
			break
		}
		header[key] = string(line[colon+1:])
		cur += nl + 2
	}
	start, err1 := strconv.Atoi(header["StartFragment"])
	end, err2 := strconv.Atoi(header["EndFragment"])
	if err1 != nil || err2 != nil || start < 0 || end <= start || end > len(b) {
		return nil, errors.New("cb: CF_HTML 头部不完整")
	}
	return &HTML{
		Fragment:  string(b[start:end]),
		SourceURL: header["SourceURL"],
		Version:   header["Version"],
	}, nil
}

// buildCFHTML 把解析后的 HTML 重建为 CF_HTML 原始字节（parseCFHTML 的逆操作）。
func buildCFHTML(h *HTML) []byte {
	frag := []byte(h.Fragment)
	prefix := []byte("<html>\r\n<body>\r\n<!--StartFragment-->")
	suffix := []byte("<!--EndFragment-->\r\n</body>\r\n</html>")

	version := h.Version
	if version == "" {
		version = "0.9"
	}

	var srcLine string
	if h.SourceURL != "" {
		srcLine = "SourceURL:" + h.SourceURL + "\r\n"
	}

	tpl := "Version:" + version + "\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n" + srcLine
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
	return blob
}

// isHeaderKey 判断是否为 CF_HTML 头部字段名（纯 ASCII 字母）。
func isHeaderKey(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

// parseHDROP 解析 CF_HDROP 字节流，返回文件绝对路径列表。
func parseHDROP(b []byte) ([]string, error) {
	if len(b) < 4 {
		return nil, errors.New("cb: DROPFILES 数据过短")
	}
	offset := int(le.Uint32(b[0:4]))
	if offset < 4 || offset >= len(b) {
		return nil, errors.New("cb: DROPFILES 偏移非法")
	}
	var names []string
	var cur []uint16
	for i := offset; i+1 < len(b); i += 2 {
		c := le.Uint16(b[i:])
		if c == 0 {
			if len(cur) == 0 {
				break // 连续两个空字符表示列表结束
			}
			names = append(names, syscall.UTF16ToString(cur))
			cur = cur[:0]
		} else {
			cur = append(cur, c)
		}
	}
	if len(names) == 0 {
		return nil, ErrNoData
	}
	return names, nil
}
