<script>
    import {Dialogs, Window} from '@wailsio/runtime';
    import {applyConfig, config, saveConfig} from '#lib/config.svelte.js';
    import {Fetch} from '#lib/fetch.js';

    // 配置由最顶级 +layout.js 加载，这里直接读写共享的 config：
    // 窗口底色、卡片透明度对所有窗口生效，其它项只影响“粘一下”窗口。

    /** 接口失败提示 */
    let error = $state('');

    /** 提交当前配置；快捷键写法不合法时不发请求 */
    const save = async () => {
        if (shortcutError) {
            error = shortcutError;
            return;
        }
        error = '';
        try {
            config.shortcut = config.shortcut.trim();
            await saveConfig();
        } catch (e) {
            error = '保存设置失败';
        }
    };

    /** 切换背景类型；选图片但还没上传过时直接弹选择框
     * @param {string} mode
     * */
    const setBgMode = async (mode) => {
        if (mode === 'image' && !config.bgImage) {
            await pickImage();
            return;
        }
        config.bgMode = mode;
        await save();
    };

    /** 选本地图片交给后端转存，成功后广播给三个窗口一起换背景 */
    const pickImage = async () => {
        const picked = await Dialogs.OpenFile({
            Title: '选择背景图片',
            Filters: [{DisplayName: '图片', Pattern: '*.png;*.jpg;*.jpeg;*.webp;*.bmp;*.gif'}]
        });
        const path = Array.isArray(picked) ? picked[0] : picked;
        if (!path) return;
        try {
            const cfg = await Fetch('/set/bgimg', {method: 'POST', body: JSON.stringify({path})});
            applyConfig(cfg);
            error = '';
        } catch (e) {
            error = '设置背景图片失败';
        }
    };

    // 与 Wails pkg/application/keys.go 的解析规则一致，用于输入时即时提示
    const MODIFIERS = ['cmdorctrl', 'cmd', 'command', 'ctrl', 'optionoralt', 'alt', 'option', 'shift', 'super'];
    const NAMED_KEYS = [
        'backspace', 'tab', 'return', 'enter', 'escape', 'space', 'delete',
        'left', 'right', 'up', 'down', 'home', 'end', 'page up', 'page down',
        'forward delete', 'numlock',
        ...Array.from({length: 35}, (_, i) => `f${i + 1}`)
    ];

    /** @param {string} value */
    const checkShortcut = (value) => {
        const s = value.trim();
        if (!s) return '快捷键不能为空';
        const parts = s.split('+');
        for (let i = 0; i < parts.length; i++) {
            const key = parts[i].trim().toLowerCase();
            if (i < parts.length - 1) {
                if (!MODIFIERS.includes(key)) return `“${parts[i].trim()}”不是 Wails 支持的修饰符`;
                continue;
            }
            if (key === 'plus' || NAMED_KEYS.includes(key)) continue;
            if (key.length !== 1 || !/^[\x21-\x7e]$/.test(key)) return `“${parts[i].trim()}”不是 Wails 支持的键名`;
        }
        return '';
    };

    const shortcutError = $derived(checkShortcut(config.shortcut));
</script>

<main class="set">
    <header class="titlebar">
        <span class="title">设置</span>
        <button class="close" title="关闭" onclick={() => Window.Hide()}>
            <svg viewBox="0 0 1024 1024" width="11" height="11">
                <path d="M96.6 915c-10.9-10.9-10.9-28.7 0-39.6L887 85c10.9-10.9 28.7-10.9 39.6 0 10.9 10.9 10.9 28.7 0 39.6L136.2 915c-10.9 10.9-28.6 10.9-39.6 0z"></path>
                <path d="M887 915L96.6 124.6c-10.9-10.9-10.9-28.7 0-39.6 10.9-10.9 28.7-10.9 39.6 0l790.4 790.4c10.9 10.9 10.9 28.7 0 39.6-10.9 10.9-28.6 10.9-39.6 0z"></path>
            </svg>
        </button>
    </header>

    <section class="content">
        {#snippet switchRow(
            /** @type {string} */ name,
            /** @type {boolean} */ value,
            /** @type {{ (): void; }} */ toggle
        )}
            <div class="row">
                <span class="name">{name}</span>
                <button class="switch" class:on={value} role="switch" aria-checked={value} aria-label={name}
                        onclick={toggle}>
                    <i></i>
                </button>
            </div>
        {/snippet}

        <p class="group">界面</p>
        <div class="card-cfg">
            <div class="row col">
                <span class="name">窗口背景</span>
                <div class="seg">
                    <button class="seg-btn" class:on={config.bgMode === 'color'} onclick={() => setBgMode('color')}>背景色</button>
                    <button class="seg-btn" class:on={config.bgMode === 'image'} onclick={() => setBgMode('image')}>背景图片</button>
                </div>
            </div>

            {#if config.bgMode === 'color'}
                <div class="row">
                    <span class="name">背景颜色</span>
                    <label class="color">
                        <input type="color" bind:value={config.bgColor} onchange={save}>
                        <span class="val">{config.bgColor}</span>
                    </label>
                </div>

            {:else}
                <div class="row col">
                    <div class="text spread">
                        <span class="name">背景图片</span>
                        <span class="desc">png / jpg / jpeg / webp / bmp / gif</span>
                    </div>
                    <div class="pick">
                        {#if config.bgImage}
                            <img src="/wg/set/bgimg?v={config.bgImage}" alt="">
                        {/if}
                        <button class="pick-btn" onclick={pickImage}>{config.bgImage ? '更换图片' : '选择图片'}</button>
                    </div>
                </div>
            {/if}

            <div class="row">
                <span class="name">文字颜色</span>
                <label class="color">
                    <input type="color" bind:value={config.textColor} onchange={save}>
                    <span class="val">{config.textColor}</span>
                </label>
            </div>

            <div class="row col">
                <div class="text spread">
                    <span class="name">窗口透明度</span>
                    <span class="val">{Math.round(config.opacity * 100)}%</span>
                </div>
                <input type="range" min="0.2" max="1" step="0.01" bind:value={config.opacity}
                       aria-label="窗口透明度" onchange={save}>
            </div>

            <div class="row col">
                <div class="text spread">
                    <span class="name">卡片透明度</span>
                    <span class="val">{Math.round(config.cardOpacity * 100)}%</span>
                </div>
                <input type="range" min="0.2" max="1" step="0.01" bind:value={config.cardOpacity}
                       aria-label="卡片透明度" onchange={save}>
            </div>

            <div class="row col">
                <span class="name">窗口大小</span>
                <div class="sizes">
                    <label class="num"><span>宽</span>
                        <input type="number" min="300" bind:value={config.width} onchange={save}>
                    </label>
                    <label class="num"><span>高</span>
                        <input type="number" min="418" bind:value={config.height} onchange={save}>
                    </label>
                    <span class="unit">px</span>
                </div>
            </div>

            <div class="row">
                <div class="text">
                    <span class="name">焦点模式</span>
                    <span class="desc">开启：面板抢占前台，支持全部键盘事件、面板内编辑、点击外部关闭</span>
                    <span class="desc">关闭：原窗口焦点不动，仅支持 ↑↓ Enter Esc 四键，无法面板内编辑</span>
                    <span class="desc">修改后需<span class="restart">重启应用</span>生效</span>
                </div>
                <button class="switch" class:on={config.focusMode} role="switch" aria-checked={config.focusMode}
                        aria-label="焦点模式" onclick={() => {config.focusMode = !config.focusMode; save();}}>
                    <i></i>
                </button>
            </div>

        </div>

        <p class="group">顶部过滤条</p>
        <div class="card-cfg">
            {@render switchRow('文字', config.filterBar.text, () => {
                config.filterBar.text = !config.filterBar.text;
                save();
            })}
            {@render switchRow('HTML', config.filterBar.html, () => {
                config.filterBar.html = !config.filterBar.html;
                save();
            })}
            {@render switchRow('图片', config.filterBar.image, () => {
                config.filterBar.image = !config.filterBar.image;
                save();
            })}
            {@render switchRow('文件', config.filterBar.files, () => {
                config.filterBar.files = !config.filterBar.files;
                save();
            })}
            {@render switchRow('搜索框', config.filterBar.search, () => {
                config.filterBar.search = !config.filterBar.search;
                save();
            })}
            {@render switchRow('清空按钮', config.filterBar.clear, () => {
                config.filterBar.clear = !config.filterBar.clear;
                save();
            })}
        </div>

        <p class="group">右侧工具条</p>
        <div class="card-cfg">
            {@render switchRow('钉住', config.toolRail.pin, () => {
                config.toolRail.pin = !config.toolRail.pin;
                save();
            })}
            {@render switchRow('打开小窗', config.toolRail.open, () => {
                config.toolRail.open = !config.toolRail.open;
                save();
            })}
            {@render switchRow('编辑', config.toolRail.edit, () => {
                config.toolRail.edit = !config.toolRail.edit;
                save();
            })}
            {@render switchRow('删除', config.toolRail.delete, () => {
                config.toolRail.delete = !config.toolRail.delete;
                save();
            })}
        </div>

        <p class="group">记录与启动</p>
        <div class="card-cfg">

            <div class="row">
                <div class="text">
                    <span class="name">显示富文本内容</span>
                    <span class="desc">关闭后 HTML 仍会保留，只按纯文本样式显示</span>
                </div>
                <button class="switch" class:on={config.richText} role="switch" aria-checked={config.richText}
                        aria-label="显示富文本内容" onclick={() => {config.richText = !config.richText; save();}}>
                    <i></i>
                </button>
            </div>

            <div class="row">
                <div class="text">
                    <span class="name">记录条数上限</span>
                    <span class="desc">超出后丢弃最旧的临时记录</span>
                </div>
                <label class="num">
                    <input type="number" min="1" bind:value={config.maxRecords} onchange={save}>
                    <span>条</span>
                </label>
            </div>

            <div class="row col">
                <span class="name">启动快捷键</span>
                <input class="key" type="text" spellcheck="false" placeholder="如 super+v"
                       bind:value={config.shortcut} onchange={save}>
                {#if shortcutError}
                    <span class="hint err">{shortcutError}</span>
                {:else}
                    <span class="hint">修饰符：ctrl / alt / shift / super（Win 键写 super，写 Win 不生效）</span>
                    <span class="hint">特殊键：space enter escape tab delete left right up down home end page up page down F1–F35</span>
                    <span class="hint">修改后需<span class="restart">重启应用</span>才会重新注册快捷键</span>
                {/if}
            </div>
        </div>

        {#if error}
            <p class="hint err">{error}</p>
        {/if}
    </section>
</main>

<style>
    .restart{
      color: red;  
    }
    
    .set {
        display: flex;
        flex-direction: column;
        height: 100vh;
        overflow: hidden;
        color: var(--text-main);
        font: 12px/1.4 system-ui, sans-serif;
        padding-bottom: 5px;
        user-select: none;
    }

    .titlebar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        height: 34px;
        padding: 0 8px 0 14px;
        user-select: none;
        --wails-draggable: drag;
    }

    .title {
        font-size: 12px;
        font-weight: 600;
        color: var(--text-strong);
    }

    .close {
        display: grid;
        place-items: center;
        width: 22px;
        height: 22px;
        padding: 0;
        border: none;
        border-radius: 4px;
        background: transparent;
        fill: var(--text-main);
        cursor: pointer;
        transition: background-color 0.15s,
        fill 0.15s;
        --wails-draggable: no-drag;
    }

    .close:hover {
        background: #f04b4b;
        fill: #fff;
    }

    .content {
        flex: 1 1 auto;
        overflow-y: auto;
        padding: 0 10px 10px;
    }

    .group {
        margin: 6px 0 5px 4px;
        color: var(--text-main);
        font: 600 12px/1.4 system-ui, sans-serif;
        letter-spacing: 0.04em;
        user-select: none;
    }

    .card-cfg {
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 10px;
        padding: 8px 10px;
    }

    .row + .row {
        border-top: 1px solid #eef0f7;
    }

    .row.col {
        flex-direction: column;
        align-items: stretch;
        gap: 6px;
    }

    .text {
        display: flex;
        flex-direction: column;
        gap: 2px;
        min-width: 0;
    }

    .text.spread {
        flex-direction: row;
        align-items: baseline;
        justify-content: space-between;
    }

    .name {
        font-size: 12px;
        color: var(--text-main);
    }

    .desc {
        color: var(--text-muted);
        font-size: 10px;
        line-height: 1.3;
    }

    .val {
        color: var(--text-strong);
        font: 600 11px/1 ui-monospace,
        Consolas,
        monospace;
    }

    .switch {
        position: relative;
        flex: 0 0 auto;
        width: 34px;
        height: 19px;
        padding: 0;
        border: 1px solid #d7dbe7;
        border-radius: 10px;
        background: #edeff5;
        cursor: pointer;
        transition: background-color 0.15s,
        border-color 0.15s;
    }

    .switch i {
        position: absolute;
        top: 1px;
        left: 1px;
        width: 15px;
        height: 15px;
        border-radius: 50%;
        background: #fff;
        box-shadow: 0 1px 2px rgb(23 29 51 / 22%);
        transition: transform 0.15s;
    }

    .switch.on {
        border-color: #2f6df6;
        background: #2f6df6;
    }

    .switch.on i {
        transform: translateX(15px);
    }

    input[type='range'] {
        width: 100%;
        height: 14px;
        margin: 0;
        appearance: none;
        background: transparent;
        cursor: pointer;
    }

    input[type='range']::-webkit-slider-runnable-track {
        height: 4px;
        border-radius: 2px;
        background: #e3e7f2;
    }

    input[type='range']::-webkit-slider-thumb {
        width: 12px;
        height: 12px;
        margin-top: -4px;
        border-radius: 50%;
        background: #2f6df6;
        box-shadow: 0 1px 3px rgb(47 109 246 / 35%);
        appearance: none;
    }

    .num {
        display: flex;
        align-items: center;
        gap: 3px;
        flex: 0 0 auto;
        color: var(--text-muted);
        font-size: 10px;
    }

    .num input {
        width: 46px;
        padding: 2px 4px;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
        background: #fff;
        color: var(--text-main);
        font: 11px/1.5 system-ui, sans-serif;
        text-align: right;
        outline: none;
    }

    .num input:focus {
        border-color: #c9d8ff;
    }

    .sizes {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .sizes .num input {
        width: 52px;
    }

    .unit {
        color: var(--text-muted);
        font-size: 10px;
    }

    .key {
        box-sizing: border-box;
        width: 100%;
        padding: 3px 6px;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
        background: #fff;
        color: var(--text-main);
        font: 600 10px/1.4 ui-monospace,
        Consolas,
        monospace;
        outline: none;
        transition: border-color 0.15s;
    }

    .key:focus {
        border-color: #c9d8ff;
    }

    .hint {
        color: var(--text-muted);
        font-size: 10px;
        line-height: 1.35;
        user-select: text;
    }

    .hint.err {
        color: #e4544b;
    }

    .seg {
        display: flex;
        gap: 4px;
        padding: 2px;
        border-radius: 5px;
        background: #eef0f7;
    }

    .seg-btn {
        flex: 1;
        padding: 4px 0;
        border: none;
        border-radius: 4px;
        background: transparent;
        color: var(--text-main);
        cursor: pointer;
        font: 11px/1.4 system-ui, sans-serif;
    }

    .seg-btn.on {
        background: #fff;
        color: var(--text-strong);
        box-shadow: 0 1px 2px rgb(23 29 51 / 10%);
    }

    .color {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .color input {
        width: 26px;
        height: 18px;
        padding: 0;
        border: 1px solid #e2e5ef;
        border-radius: 3px;
        background: none;
        cursor: pointer;
    }

    .pick {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .pick img {
        width: 64px;
        height: 36px;
        object-fit: cover;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
    }

    .pick-btn {
        padding: 3px 8px;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
        background: #fff;
        color: var(--text-main);
        cursor: pointer;
        font: 11px/1.5 system-ui, sans-serif;
    }
</style>
