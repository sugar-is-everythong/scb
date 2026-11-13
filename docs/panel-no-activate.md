# 面板零抢焦点方案说明

## 需求

Alt+V 唤起剪贴板历史面板时：

1. **一刻都不能抢焦点**——原窗口光标继续闪烁，输入不中断
2. **键盘交互可用**——↑↓ 选择、Enter 粘贴、Esc 关闭
3. 复刻 Win+V 的体验

## 核心矛盾

Win32 层面三条要求不能同时成立：

| 要求 | 约束 |
|------|------|
| 面板能键盘输入 | 必须是前台窗口 |
| 完全不抢焦点 | 必须不是前台窗口 |
| WebView2 内容显示 | Wails `Show()` 内部走 `ShowWindow(SW_SHOW)`，自带激活 |

Win+V 本身不依赖 Win32 前台——它是 WinUI 3 Shell 应用，键盘走 `Windows.UI.Input` 路由，UIA focus 和 Win32 foreground 是两回事。Wails/WebView2 做不到这一点，键盘焦点必须前台。

## 方案演进（四次失败）

| 方案 | 结果 | 失败原因 |
|------|------|----------|
| `WS_EX_NOACTIVATE` + Wails `Show()` | 抢焦点 | `WS_EX_NOACTIVATE` 只拦"用户点击激活"，拦不住 `ShowWindow(SW_SHOW)` 的程序化激活 |
| `Show()` 后 `FocusWindow(原窗口)` 归还 | 瞬间抢焦点 | 显示那一刻还是激活了一次，违反"一刻都不抢" |
| `SW_SHOWNOACTIVATE` 绕开 Wails `Show()` | 黑屏 | Wails 的 `chromium.Show()` 没被触发，WebView2 合成树不可见 |
| `LockSetForegroundWindow` 锁定前台 | 时好时坏 | Alt+V 快捷键里的 Alt 键会自动解锁前台锁（Microsoft 文档明确） |

## 最终方案

**unsafe 反射调 `chromium.Show()` + `SetWindowPos(SWP_NOACTIVATE|SWP_SHOWWINDOW)`**，全程不走 `SW_SHOW`：

```
显示：
  1. 反射调 chromium.Show()        → WebView2 合成树可见（COM 调用，不碰父窗口激活）
  2. SetWindowPos(SWP_SHOWWINDOW)  → 父窗口可见（此 API 从不激活）

隐藏：
  ShowWindow(SW_HIDE)              → 只隐藏窗口，不动 chromium、不碰 Wails 内部状态

键盘：
  WH_KEYBOARD_LL 钩子拦 ↑↓EnterEsc → Wails 事件 key-down → 前端代码处理
```

两条可见性路径都不碰前台，真正零激活。

## 关键实现

### 1. `tools/webview.go`（新建）

- `showChromium`：unsafe 反射穿透 `WebviewWindow.impl` → `windowsWebviewWindow.chromium`，调 `chromium.Show()`。只取 Wails `Show()` 里"让 WebView2 渲染可见"的那一半，不要它"激活父窗口"的那一半。
- `ShowNoActivate`：反射调 `chromium.Show()` + `SetWindowPos(SWP_NOACTIVATE|SWP_SHOWWINDOW)`。
- `HideNoActivate`：直接 `ShowWindow(SW_HIDE)`。**不调 Wails 的 `Hide()`**——后者会调 `chromium.Hide()` 并重置 `windowShown`/`showRequested`，导致下次显示状态错乱。
- `IsWindowVisibleReal`：直接读 `WS_VISIBLE`。`mainW.IsVisible()` 读的是 Wails 内部 `windowShown` 字段，绕开 `Show()/Hide()` 后永远 false，不能用于切换判断。

### 2. `tools/panel_keys.go`（新建）

`HookPanelKeys(enabled, onKey)`：独立的 `WH_KEYBOARD_LL` 钩子，面板可见时拦截 ↑↓EnterEsc，`return 1` 吞掉（前台应用收不到），同时异步 `onKey` 派发。通过 Wails `app.Event.Emit("key-down", {key})` 转发给前端。

### 3. `main.go`

- `showAtCursor`：切换判断用 `IsWindowVisibleReal`，显示用 `ShowNoActivate`，收起用 `HideNoActivate`
- 装 `HookPanelKeys`，enabled 回调用 `IsWindowVisibleReal`
- `HideOnFocusLost: false`（面板从不抢前台，失焦隐藏无意义）
- `ExStyle` 保留 `WS_EX_NOACTIVATE`（阻止点击面板时激活）

### 4. `service/handlers.go`

- `inset`：`HideNoActivate` + `FocusWindow(global.Foreground)` + `WaitForegroundWindow` + `Paste`
- 新增 `cbHide`：给前端 Esc 用的隐藏接口

### 5. `frontend/+page.svelte`

- `handlePanelKey(key)`：统一处理四键，原生 `keydown` 和 Go 钩子转发的 `key-down` 事件共用
- Esc 改走 `Fetch('/cb/hide')`，**不调 `Window.Hide()`**

## 关键坑

### 前端 Esc 不能调 `Window.Hide()`

Wails 的 `Hide()` 做三件事：`ShowWindow(SW_HIDE)` + `chromium.Hide()` + 重置 `windowShown`/`showRequested`。

按 Esc 后 `showRequested=false`，下次显示时 Wails 内部逻辑（`onNavigationCompleted` 里的 `if !windowShown && !showRequested { chromium.Hide() }`）会再次把 chromium 藏掉，导致状态错乱。

**所有隐藏路径必须统一走 `HideNoActivate`**，不能混用 Wails 的 `Hide()`。

### 点击面板时 WebView2 会激活父窗口

WebView2 的点击处理会调 `SetFocus`/`SetForegroundWindow` 程序化激活父窗口，`WS_EX_NOACTIVATE` 拦不住。所以点击面板那一瞬间前台会被抢。

`inset` 里已加 `FocusWindow(global.Foreground)` 显式归还。点击面板其他地方（滚动、空白区域）可能仍有瞬间激活，如需彻底消除要在 `WM_ACTIVATE` 里做"被激活就归还"。

### unsafe 反射的版本耦合

`showChromium` 依赖 Wails 内部字段名 `impl`、`chromium`。当前 v3.0.0 固定可用；升级 Wails 后如果字段名变了，`ShowNoActivate` 返回错误并 log "面板显示失败"，页面黑屏。届时需同步调整反射路径。

## 已知副作用

- 面板可见期间，任何前台应用按 ↑↓EnterEsc 都会被劫持到面板（与 Win+V 行为一致）
- 点击面板时 WebView2 会激活父窗口（`inset` 已归还，其他点击场景未处理）

## 文件清单

| 文件 | 改动 |
|------|------|
| `tools/webview.go` | 新建：`ShowNoActivate`/`HideNoActivate`/`IsWindowVisibleReal`/`showChromium` |
| `tools/panel_keys.go` | 新建：`HookPanelKeys` 键盘钩子 |
| `tools/sendInput.go` | 新增 `LockSetForegroundWindow`（最终未使用，可删） |
| `main.go` | `showAtCursor` 改用新 API，装键盘钩子 |
| `service/handlers.go` | `inset` 改用 `HideNoActivate`+`FocusWindow`，新增 `cbHide` |
| `service/routes.go` | 注册 `POST /cb/hide` |
| `frontend/+page.svelte` | `handlePanelKey` 处理四键，Esc 改走 `/cb/hide` |
