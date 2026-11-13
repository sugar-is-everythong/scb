package set

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"scb/global"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FilterBar 顶部过滤条的显示项，对应主窗口顶栏上的四个类型开关与搜索、清空
type FilterBar struct {
	Text   bool `json:"text"`   // 文字过滤开关
	HTML   bool `json:"html"`   // HTML 过滤开关
	Image  bool `json:"image"`  // 图片过滤开关
	Files  bool `json:"files"`  // 文件过滤开关
	Search bool `json:"search"` // 搜索框
	Clear  bool `json:"clear"`  // 清空按钮
}

// ToolRail 卡片右侧工具条的显示项
type ToolRail struct {
	Pin    bool `json:"pin"`    // 钉住
	Open   bool `json:"open"`   // 在小窗打开
	Edit   bool `json:"edit"`   // 编辑
	Delete bool `json:"delete"` // 删除
}

// Config 是落盘到 global.Local/config.json 的用户设置。
// JSON 名与前端设置页的字段同名，Go 与 JS 直接互转，不需要额外映射。
type Config struct {
	// FilterBar 顶部过滤条的显示项
	FilterBar FilterBar `json:"filterBar"`
	// ToolRail 卡片右侧工具条的显示项
	ToolRail ToolRail `json:"toolRail"`
	// Opacity 窗口底色的不透明度，取值 0.2~1，1 为完全不透明。
	// 所有窗口的底色都用它
	Opacity float64 `json:"opacity"`
	// BgMode 背景类型，取值 color 或 image，两者互斥
	BgMode string `json:"bgMode"`
	// BgColor 窗口背景色，#rrggbb，仅 BgMode 为 color 时生效
	BgColor string `json:"bgColor"`
	// BgImage 背景图片文件名，位于 global.Local 目录，仅 BgMode 为 image 时生效
	BgImage string `json:"bgImage"`
	// CardOpacity 卡片底色的不透明度，取值 0.2~1，与 Opacity 相互独立。
	// 所有窗口的卡片、面板底色都用它
	CardOpacity float64 `json:"cardOpacity"`
	// TextColor 文字强调色，#rrggbb。正文色与提示色由它按同色相降饱和生成，逐级变浅
	TextColor string `json:"textColor"`
	// Width Height 主窗口逻辑尺寸，单位与 Window.SetSize(width, height int) 一致
	Width  int `json:"width"`
	Height int `json:"height"`
	// MaxRecords 历史记录条数上限，超出后丢弃最旧的临时记录
	MaxRecords int `json:"maxRecords"`
	// Shortcut 唤起主窗口的全局快捷键，写法见文件末尾注释
	Shortcut string `json:"shortcut"`
	// RichText 是否以富文本渲染 HTML；false 时只按纯文本样式显示，HTML 数据仍然保留
	RichText bool `json:"richText"`

	// FocusMode 焦点模式：true=抢焦点（面板显示时抢占前台，支持全部键盘事件、面板内编辑、点击外部关闭）；
	// false=零抢焦点（原窗口焦点不动，仅 ↑↓EnterEsc 四键通过钩子转发）。
	// 修改后需重启应用生效（ExStyle 和 HideOnFocusLost 是窗口创建时选项）。
	FocusMode bool `json:"focusMode"`

	// WindowMode 窗口的创建模式 w32.WS_EX_NOACTIVATE 或者 | w32.WS_EX_TOOLWINDOW
	WindowMode int `json:"windowMode"`
}

const (
	MainWindowName = "粘一下"

	// BgModeColor / BgModeImage 是窗口背景的两个互斥取值
	BgModeColor = "color"
	BgModeImage = "image"

	// defaultBgColor 是主窗口原本写死的底色 #adc5ff（即 rgb 173 197 255）
	defaultBgColor = "#adc5ff"

	// defaultTextColor 是文字强调色的默认值，正文色与提示色由它按色相生成
	defaultTextColor = "#000000"

	fileName = "config.json"

	// 窗口尺寸限制下限
	minWidth  = 300
	minHeight = 418

	// 记录条数下限，最少你得记录一条吧，要不然你干嘛来了
	minRecords = 1

	opacityMin = 0
)

// mu 保护配置文件读写，避免设置页连点保存时两个请求交叉写同一个临时文件
var mu sync.Mutex

// Path 返回配置文件路径：与历史索引、图片同放在可执行文件旁的 local 目录
func Path() string {
	return filepath.Join(global.Local, fileName)
}

// Default 返回默认设置，与前端设置页首次渲染的初值一致
func Default() Config {
	return Config{
		FilterBar:   FilterBar{Text: true, HTML: true, Image: true, Files: true, Search: true, Clear: true},
		ToolRail:    ToolRail{Pin: true, Open: true, Edit: true, Delete: true},
		Opacity:     0.3,
		BgMode:      BgModeColor,
		BgColor:     defaultBgColor,
		CardOpacity: 0.6,
		TextColor:   defaultTextColor,
		Width:       300,
		Height:      418,
		MaxRecords:  100,
		Shortcut:    "super+v",
		RichText:    true,
	}
}

// Load 读取 config.json。
// 文件不存在、内容损坏都返回默认设置，不能因为一个坏配置让程序起不来。
func Load() Config {
	mu.Lock()
	defer mu.Unlock()
	return load()
}

func load() Config {
	data, err := os.ReadFile(Path())
	if err != nil {
		return Default()
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Println("set: config.json 解析失败，使用默认设置:", err)
		return Default()
	}
	cfg.normalize()
	return cfg
}

// Save 校验并写入 config.json，返回规范化后真正落盘的内容，前端据此回显
func Save(cfg Config) (Config, error) {
	if err := ValidateShortcut(cfg.Shortcut); err != nil {
		return Config{}, err
	}
	mu.Lock()
	cfg.normalize()
	err := save(cfg)
	mu.Unlock()
	if err != nil {
		return Config{}, err
	}
	applySize(cfg)
	return cfg, nil
}

// applySize 只对主窗口有效
func applySize(cfg Config) {
	mw, _ := global.App.Window.GetByName(MainWindowName) // 其实吧应该看看窗口在不在，有可能不在，但是不在应该不可能
	mw.SetSize(cfg.Width, cfg.Height)
}

// BackgroundFile 返回背景图的绝对路径；未设置或文件不存在时返回空字符串。
// 只认 global.Local 下的文件名，避免配置里写路径能读到其它目录的文件。
func BackgroundFile(cfg Config) string {
	if cfg.BgImage == "" {
		return ""
	}
	p := filepath.Join(global.Local, filepath.Base(cfg.BgImage))
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// imageExts 是允许用作窗口背景的图片扩展名
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".bmp": true, ".gif": true,
}

// SetBackground 把用户选的图片复制进数据目录并设为窗口背景，返回更新后的设置。
// 每次都用新文件名：同名文件会被浏览器缓存，换图不生效；旧图在写完后删掉。
func SetBackground(src string) (Config, error) {
	ext := strings.ToLower(filepath.Ext(src))
	if !imageExts[ext] {
		return Config{}, fmt.Errorf("只支持 png / jpg / jpeg / webp / bmp / gif 图片")
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return Config{}, fmt.Errorf("读取图片失败: %w", err)
	}

	mu.Lock()
	cfg := load()
	// 文件名带纳秒时间戳，保证每次换图 URL 都变
	name := fmt.Sprintf("bg-%d%s", time.Now().UnixNano(), ext)
	if err := os.MkdirAll(global.Local, 0755); err != nil {
		mu.Unlock()
		return Config{}, err
	}
	if err := os.WriteFile(filepath.Join(global.Local, name), data, 0644); err != nil {
		mu.Unlock()
		return Config{}, err
	}
	old := cfg.BgImage
	cfg.BgImage = name
	cfg.BgMode = BgModeImage
	if err := save(cfg); err != nil {
		mu.Unlock()
		return Config{}, err
	}
	mu.Unlock()

	// 旧图清理放在锁外：删不掉只是多占一点磁盘，不影响本次设置
	removeBackground(old)
	return cfg, nil
}

// removeBackground 只删数据目录里由本程序生成的 bg- 文件，避免误删用户文件
func removeBackground(name string) {
	base := filepath.Base(name)
	if name == "" || !strings.HasPrefix(base, "bg-") {
		return
	}
	_ = os.Remove(filepath.Join(global.Local, base))
}

// save 先写临时文件再改名，避免写到一半被读到半截 JSON
func save(cfg Config) error {
	if err := os.MkdirAll(global.Local, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	p := Path()
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p)
}

// normalize 把越界值修正回合法范围，防止手改配置后落盘出 0 宽的窗口
func (c *Config) normalize() {
	d := Default()
	if c.Width < minWidth {
		c.Width = d.Width
	}
	if c.Height < minHeight {
		c.Height = d.Height
	}
	if c.MaxRecords < minRecords {
		c.MaxRecords = d.MaxRecords
	}
	if c.Opacity < opacityMin || c.Opacity > 1 {
		c.Opacity = d.Opacity
	}
	if c.CardOpacity < opacityMin || c.CardOpacity > 1 {
		c.CardOpacity = d.CardOpacity
	}
	c.Shortcut = strings.TrimSpace(c.Shortcut)
	if c.Shortcut == "" {
		c.Shortcut = d.Shortcut
	}
	if c.BgMode != BgModeImage {
		c.BgMode = BgModeColor
	}
	c.TextColor = normalizeHexColor(c.TextColor, d.TextColor)
	c.BgColor = normalizeHexColor(c.BgColor, d.BgColor)
	// 零抢焦点模式下面板不是前台，搜索框无法输入、编辑无法打字，强制关闭
	if !c.FocusMode {
		c.FilterBar.Search = false
		c.ToolRail.Edit = false
	}
	// 选的图片丢了就退回纯色，否则窗口背景会变成完全透明
	if c.BgMode == BgModeImage && BackgroundFile(*c) == "" {
		c.BgMode = BgModeColor
	}
}

// normalizeHexColor 把 #rrggbb 规整成小写；不合法时用 fallback
func normalizeHexColor(v, fallback string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if !strings.HasPrefix(v, "#") {
		v = "#" + v
	}
	if len(v) == 7 {
		if _, err := strconv.ParseUint(v[1:], 16, 32); err == nil {
			return v
		}
	}
	return fallback
}

// acceleratorModifiers 是 Wails 解析器接受的修饰符名，来源 pkg/application/keys.go 的 modifierMap。
// 注意 Windows 键必须写 super，SuperKey 的 String() 输出是 "Win" 但解析时不认 "win"。
var acceleratorModifiers = map[string]bool{
	"cmdorctrl":   true,
	"cmd":         true,
	"command":     true,
	"ctrl":        true,
	"optionoralt": true,
	"alt":         true,
	"option":      true,
	"shift":       true,
	"super":       true,
}

// acceleratorNamedKeys 是 Wails 解析器接受的命名键，与 keys.go 的 namedKeys 保持一致
var acceleratorNamedKeys = func() map[string]bool {
	m := map[string]bool{
		"backspace": true, "tab": true, "return": true, "enter": true, "escape": true,
		"left": true, "right": true, "up": true, "down": true, "space": true, "delete": true,
		"forward delete": true, "home": true, "end": true, "page up": true, "page down": true,
		"numlock": true,
	}
	for i := 1; i <= 35; i++ {
		m["f"+strconv.Itoa(i)] = true
	}
	return m
}()

// ValidateShortcut 校验快捷键是否符合 Wails accelerator 写法，规则与 keys.go 的 parseAccelerator 一致：
// 除最后一段外必须都是修饰符，最后一段是主键（命名键、plus 或单个可打印字符）
func ValidateShortcut(shortcut string) error {
	s := strings.TrimSpace(shortcut)
	if s == "" {
		return fmt.Errorf("快捷键不能为空")
	}
	parts := strings.Split(s, "+")
	for i, part := range parts {
		key := strings.ToLower(strings.TrimSpace(part))
		if i < len(parts)-1 {
			if !acceleratorModifiers[key] {
				return fmt.Errorf("“%s”不是 Wails 支持的修饰符", part)
			}
			continue
		}
		if !validAcceleratorKey(key) {
			return fmt.Errorf("“%s”不是 Wails 支持的键名", part)
		}
	}
	return nil
}

func validAcceleratorKey(key string) bool {
	if key == "plus" || acceleratorNamedKeys[key] {
		return true
	}
	// 与 Wails 一致：只接受长度为 1 字节的可打印字符
	return len(key) == 1 && strconv.IsPrint(rune(key[0]))
}

/*
Wails v3 accelerator 写法，可直接原样填进设置页的“启动快捷键”输入框：

	修饰符（不区分大小写，用 + 连接，写在最前面）：
	    cmdorctrl | cmd | command   Control（macOS 上为 Command）
	    ctrl                        仅 Control
	    optionoralt | alt | option  Alt
	    shift                       Shift
	    super                       Windows 键（macOS 上为 Command）——不能写 Win，解析器不认

	主键（必须放最后一段）：
	    a~z、0~9、标点等单个可打印字符，如 v、`、1
	    加号本身必须写 plus，因为 + 是分隔符
	    命名键：backspace tab return enter escape space delete
	           left right up down home end page up page down
	           forward delete numlock f1~f35

	示例：super+v、ctrl+shift+a、alt+space、ctrl+`、super+plus

	系统已占用的组合（如 super+v，被资源管理器剪贴板历史注册）用 RegisterHotKey 必然注册失败，
	这类组合只能靠键盘钩子接管。
*/
