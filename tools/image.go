package tools

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
)

func GetWebImage(url string) ([]byte, error) {

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tools: 图片 %q 响应状态码 %d", url, res.StatusCode)
	}

	imgData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return imgData, nil
}

const defaultJpegQuality = 82

// CompressConfig 图片压缩配置。零值即可用：不缩放、输出 JPEG、质量 82。
type CompressConfig struct {
	MaxWidth int  // 输出最大宽度(px)，高度按比例；<=0 保持原尺寸
	Quality  int  // JPEG 质量 1-100，为 0 或越界按 82
	Png      bool // true 输出 PNG（保留透明，体积比 JPEG 大一个数量级）
	PngFast  bool // PNG 用最快压缩档：速度快一倍，体积略增
}

// CompressImg 图片压缩，第一个参数是原始图片，第二个参数是压缩配置，返回一个图片
func CompressImg(imgData []byte, cfg CompressConfig) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("tools: 图片解码失败: %w", err)
	}

	rgba := toRGBA(src)
	if cfg.MaxWidth > 0 && rgba.Bounds().Dx() > cfg.MaxWidth {
		rgba = downscale(rgba, cfg.MaxWidth)
	}

	var buf bytes.Buffer
	if cfg.Png {
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		if cfg.PngFast {
			enc.CompressionLevel = png.BestSpeed
		}
		if err := enc.Encode(&buf, rgba); err != nil {
			return nil, fmt.Errorf("tools: PNG 编码失败: %w", err)
		}
		return buf.Bytes(), nil
	}

	q := cfg.Quality
	if q < 1 || q > 100 {
		q = defaultJpegQuality
	}
	// JPEG 没有透明通道，先把半透明像素压到白底上，否则透明区会变成黑色
	if err := jpeg.Encode(&buf, flattenAlpha(rgba), &jpeg.Options{Quality: q}); err != nil {
		return nil, fmt.Errorf("tools: JPEG 编码失败: %w", err)
	}
	return buf.Bytes(), nil
}

// toRGBA 统一成 *image.RGBA 以便逐像素缩放；已经是 RGBA 就直接用，省一次全图拷贝
func toRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba
	}
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return dst
}

// flattenAlpha 把带透明的图合成到白底
func flattenAlpha(src *image.RGBA) image.Image {
	if src.Opaque() {
		return src
	}
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Over)
	return dst
}

// downscale 缩到 dstW 宽，高度按比例。先逐级折半（每级 2x2 平均），
// 最后一步再按剩余比例做盒式平均精确对齐。
// 直接对大比例做一次性盒式平均会漏采样高频细节，产生锯齿和摩尔纹。
func downscale(src *image.RGBA, dstW int) *image.RGBA {
	b := src.Bounds()
	dstH := max(1, b.Dy()*dstW/b.Dx())

	cur := src
	for cur.Bounds().Dx()/2 >= dstW {
		cur = halve(cur)
	}
	if cur.Bounds().Dx() != dstW {
		cur = boxFit(cur, dstW, dstH)
	}
	return cur
}

// halve 2 倍折半降采样
func halve(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dw, dh := max(1, b.Dx()/2), max(1, b.Dy()/2)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var sum [4]uint32
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					o := src.PixOffset(b.Min.X+x*2+dx, b.Min.Y+y*2+dy)
					for i := 0; i < 4; i++ {
						sum[i] += uint32(src.Pix[o+i])
					}
				}
			}
			o := dst.PixOffset(x, y)
			for i := 0; i < 4; i++ {
				dst.Pix[o+i] = byte(sum[i] / 4)
			}
		}
	}
	return dst
}

// boxFit 任意比例盒式平均缩放：每个目标像素聚合其对应的源像素块
func boxFit(src *image.RGBA, dstW, dstH int) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		y0 := y * sh / dstH
		y1 := max((y+1)*sh/dstH, y0+1)
		for x := 0; x < dstW; x++ {
			x0 := x * sw / dstW
			x1 := max((x+1)*sw/dstW, x0+1)
			var sum [4]uint32
			var n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					o := src.PixOffset(b.Min.X+sx, b.Min.Y+sy)
					for i := 0; i < 4; i++ {
						sum[i] += uint32(src.Pix[o+i])
					}
					n++
				}
			}
			o := dst.PixOffset(x, y)
			for i := 0; i < 4; i++ {
				dst.Pix[o+i] = byte(sum[i] / n)
			}
		}
	}
	return dst
}
