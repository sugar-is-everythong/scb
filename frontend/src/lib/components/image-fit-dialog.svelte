<script>
    import CloseButton from './close-button.svelte';

    /**
     * 通用图片适配编辑对话框：原生 <dialog> 页面内模态，不创建新窗口。
     * 与业务零耦合——图片用 src（任意 URL）传入，选图流程委托给 onpick，
     * 变换用归一化 scale/x/y 表达（scale≥minScale 保证铺满不变形，x/y∈[0,1] 不露边），
     * 编辑实时双向绑定，点“确定”才触发 onchange 持久化，取消/Esc/点遮罩回滚到打开前的快照。
     * @type {{
     *   open?: boolean,
     *   src?: string,
     *   ratio?: number,
     *   scale?: number,
     *   x?: number,
     *   y?: number,
     *   minScale?: number,
     *   maxScale?: number,
     *   title?: string,
     *   onpick?: () => any,
     *   onchange?: () => void,
     *   errorText?: string
     * }}
     */
    let {
        open = $bindable(false),
        src = '',
        ratio = 1,
        scale = $bindable(1),
        x = $bindable(0.5),
        y = $bindable(0.5),
        minScale = 1,
        maxScale = 4,
        title = '调整图片',
        onpick,
        onchange,
        errorText = '选择图片失败，请重试'
    } = $props();

    /** @param {number} v @param {number} lo @param {number} hi */
    const clamp = (v, lo, hi) => Math.min(hi, Math.max(lo, v));
    /** @param {number} v */
    const clamp01 = (v) => clamp(v, 0, 1);

    /**
     * cover 基准 × scale 得显示尺寸，焦点换算左上角偏移；任一尺寸未就绪返回全 0
     * @param {number} s 放大倍数
     * @param {number} fx 水平焦点 0~1
     * @param {number} fy 垂直焦点 0~1
     * @param {number} nw 图片原始宽
     * @param {number} nh 图片原始高
     * @param {number} w 容器宽
     * @param {number} h 容器高
     * @returns {{dw: number, dh: number, ox: number, oy: number}}
     */
    const layout = (s, fx, fy, nw, nh, w, h) => {
        if (!nw || !nh || !w || !h) return {dw: 0, dh: 0, ox: 0, oy: 0};
        const base = Math.max(w / nw, h / nh);
        const dw = nw * base * s;
        const dh = nh * base * s;
        return {dw, dh, ox: (w - dw) * fx, oy: (h - dh) * fy};
    };

    let err = $state('');
    let picking = $state(false);

    // 原始尺寸：src 变化时重新加载
    let natW = $state(0);
    let natH = $state(0);
    $effect(() => {
        natW = natH = 0;
        if (!src) return;
        let alive = true;
        const img = new Image();
        img.onload = () => {
            if (alive) {
                natW = img.naturalWidth;
                natH = img.naturalHeight;
            }
        };
        img.src = src;
        return () => {
            alive = false;
        };
    });

    let boxW = $state(0);
    let boxH = $state(0);
    const bgStyle = $derived.by(() => {
        const url = `background-image: url('${src}');`;
        const g = layout(scale, x, y, natW, natH, boxW, boxH);
        if (!g.dw) return `${url} background-size: cover; background-position: center;`;
        return `${url} background-size: ${g.dw}px ${g.dh}px; background-position: ${x * 100}% ${y * 100}%;`;
    });

    // 对话框生命周期：打开存快照，外部置 open=false 视为取消并回滚
    /** @type {HTMLDialogElement | undefined} */
    let dialogEl = $state();
    /** @type {{scale: number, x: number, y: number} | null} */
    let snapshot = null;
    const takeSnapshot = () => {
        snapshot = {scale, x, y};
    };
    const restore = () => {
        if (snapshot) {
            scale = snapshot.scale;
            x = snapshot.x;
            y = snapshot.y;
        }
    };
    $effect(() => {
        if (!dialogEl) return;
        if (open && !dialogEl.open) {
            takeSnapshot();
            err = '';
            dialogEl.showModal();
        } else if (!open && dialogEl.open) {
            restore();
            dialogEl.close();
        }
    });
    const confirm = () => {
        dialogEl?.close();
        open = false;
        onchange?.();
    };
    const cancel = () => {
        restore();
        dialogEl?.close();
        open = false;
    };

    /** 选图委托给父级：成功后重存快照（换图后变换通常被重置），失败显示错误文案 */
    const doPick = async () => {
        if (!onpick || picking) return;
        picking = true;
        err = '';
        try {
            await onpick();
            takeSnapshot();
        } catch (e) {
            err = errorText;
        } finally {
            picking = false;
        }
    };

    /**
     * 以光标为锚点缩放，保持光标下的图像点不动，再夹紧焦点防露边
     * @param {number} cx 光标在容器内的 x
     * @param {number} cy 光标在容器内的 y
     * @param {number} next 目标倍数
     */
    const zoomAt = (cx, cy, next) => {
        next = clamp(next, minScale, maxScale);
        if (next === scale || !natW || !boxW) return;
        const g = layout(scale, x, y, natW, natH, boxW, boxH);
        const u = g.dw ? (cx - g.ox) / g.dw : 0.5;
        const v = g.dh ? (cy - g.oy) / g.dh : 0.5;
        const n = layout(next, x, y, natW, natH, boxW, boxH);
        scale = next;
        x = boxW - n.dw ? clamp01((cx - u * n.dw) / (boxW - n.dw)) : 0.5;
        y = boxH - n.dh ? clamp01((cy - v * n.dh) / (boxH - n.dh)) : 0.5;
    };

    /**
     * wheel 需 preventDefault 挡滚动，必须 passive:false，用 action 挂
     * @param {HTMLElement} node
     * @returns {{destroy: () => void}}
     */
    const wheel = (node) => {
        /** @param {WheelEvent} e */
        const handler = (e) => {
            e.preventDefault();
            if (!natW) return;
            const rect = node.getBoundingClientRect();
            zoomAt(e.clientX - rect.left, e.clientY - rect.top, scale * Math.exp(-e.deltaY * 0.0015));
        };
        node.addEventListener('wheel', handler, {passive: false});
        return {destroy: () => node.removeEventListener('wheel', handler)};
    };

    // 拖动平移：指针位移换算焦点位移，恰好铺满（分母 0）的轴不动
    let dragging = $state(false);
    let lastX = 0;
    let lastY = 0;
    /** @param {PointerEvent & {currentTarget: HTMLDivElement}} e */
    const onDown = (e) => {
        if (!natW || e.button !== 0) return;
        dragging = true;
        lastX = e.clientX;
        lastY = e.clientY;
        e.currentTarget.setPointerCapture(e.pointerId);
    };
    /** @param {PointerEvent & {currentTarget: HTMLDivElement}} e */
    const onMove = (e) => {
        if (!dragging) return;
        const dx = e.clientX - lastX;
        const dy = e.clientY - lastY;
        lastX = e.clientX;
        lastY = e.clientY;
        const g = layout(scale, x, y, natW, natH, boxW, boxH);
        const sx = boxW - g.dw;
        const sy = boxH - g.dh;
        if (sx) x = clamp01(x + dx / sx);
        if (sy) y = clamp01(y + dy / sy);
    };
    /** @param {PointerEvent & {currentTarget: HTMLDivElement}} e */
    const onUp = (e) => {
        if (!dragging) return;
        dragging = false;
        if (e.currentTarget.hasPointerCapture(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId);
    };

    const reset = () => {
        scale = minScale;
        x = 0.5;
        y = 0.5;
    };

    /** 键盘等价操作：方向键平移、加减缩放、0 重置 */
    /** @param {KeyboardEvent} e */
    const onKey = (e) => {
        if (!natW) return;
        const g = layout(scale, x, y, natW, natH, boxW, boxH);
        const step = 0.05;
        let handled = true;
        switch (e.key) {
            case 'ArrowLeft':
                if (boxW - g.dw) x = clamp01(x - step);
                break;
            case 'ArrowRight':
                if (boxW - g.dw) x = clamp01(x + step);
                break;
            case 'ArrowUp':
                if (boxH - g.dh) y = clamp01(y - step);
                break;
            case 'ArrowDown':
                if (boxH - g.dh) y = clamp01(y + step);
                break;
            case '+':
            case '=':
                zoomAt(boxW / 2, boxH / 2, scale * 1.1);
                break;
            case '-':
            case '_':
                zoomAt(boxW / 2, boxH / 2, scale / 1.1);
                break;
            case '0':
                reset();
                break;
            default:
                handled = false;
        }
        if (handled) e.preventDefault();
    };
</script>

<dialog bind:this={dialogEl} class="fit-dialog" aria-label={title}
        onclick={(e) => { if (e.target === dialogEl) cancel(); }}
        oncancel={(e) => { e.preventDefault(); cancel(); }}>
    <div class="panel">
        <header class="head">
            <div class="title">{title}</div>
            <CloseButton onclick={cancel}/>
        </header>

        <div class="body">
            <div class={['stage', {dragging}]}
                 style="aspect-ratio: {ratio}; width: min(100%, {Math.round(170 * ratio)}px)"
                 bind:clientWidth={boxW} bind:clientHeight={boxH}
                 role="application" tabindex="0" aria-label="图片预览：拖动或方向键调位置，滚轮或加减键缩放"
                 use:wheel
                 onpointerdown={onDown} onpointermove={onMove} onpointerup={onUp} onpointercancel={onUp}
                 onkeydown={onKey}>
                {#if src && natW}
                    <div class="bg" style={bgStyle}></div>
                {:else if src}
                    <span class="loading">加载中…</span>
                {:else}
                    <button class="empty" onclick={doPick} disabled={!onpick || picking}>
                        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor"
                             stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                            <rect x="3" y="3" width="18" height="18" rx="2"></rect>
                            <circle cx="8.5" cy="8.5" r="1.5"></circle>
                            <path d="M21 15l-5-5L5 21"></path>
                        </svg>
                        <span>{picking ? '选择中…' : '选择图片'}</span>
                    </button>
                {/if}
                {#if src}
                    <span class="badge">{Math.round(scale * 100)}%</span>
                {/if}
            </div>

            <div class="ctrl">
                <input type="range" min={minScale} max={maxScale} step="0.01" bind:value={scale}
                       aria-label="图片缩放" disabled={!src}>
                <button class="mini" onclick={reset} disabled={!src}>重置</button>
            </div>

            {#if onpick}
                <button class="pick" onclick={doPick} disabled={picking}>
                    {picking ? '选择中…' : (src ? '更换图片' : '选择图片')}
                </button>
            {/if}

            {#if err}
                <p class="err">{err}</p>
            {/if}
            <p class="hint">拖动调位置，滚轮缩放</p>
        </div>

        <footer class="foot">
            <button class="btn ghost" onclick={cancel}>取消</button>
            <button class="btn primary" onclick={confirm} disabled={!src}>确定</button>
        </footer>
    </div>
</dialog>

<style>
    .fit-dialog {
        position: fixed;
        inset: 0;
        margin: auto;
        padding: 0;
        border: none;
        border-radius: 10px;
        background: transparent;
        width: min(94vw, 360px);
        height: fit-content;
        max-height: 92vh;
        overflow: visible;
        --wails-draggable: no-drag;

    }

    .fit-dialog::backdrop {
        background: rgb(13 16 28 / 48%);
        backdrop-filter: blur(2px);
    }

    .fit-dialog[open] {
        animation: pop 0.18s cubic-bezier(0.2, 0.9, 0.3, 1.2);
    }

    @keyframes pop {
        from {
            opacity: 0;
            transform: translateY(8px) scale(0.96);
        }
        to {
            opacity: 1;
            transform: none;
        }
    }

    .panel {
        margin: auto;
        display: flex;
        flex-direction: column;
        border-radius: 10px;
        background: #fff;
        box-shadow: 0 12px 40px rgb(13 16 28 / 28%);
        color: var(--text-main, #3a4160);
        font: 12px/1.4 system-ui, sans-serif;
        user-select: none;
        overflow: clip;
    }

    .head {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        padding: 3px 8px 2px 8px;
        border-bottom: 1px solid #eef0f7;
    }

    .title {
        font-size: 12px;
        font-weight: 600;
        color: var(--text-main, #1c2340);
    }

    .body {
        flex: 1 1 auto;
        overflow-y: auto;
        padding: 12px 14px;
        display: flex;
        flex-direction: column;
        gap: 8px;
        scrollbar-width: thin;
        scrollbar-color: #c9d8ff #f1f3fa;
    }
    
    .stage {
        display: grid;
        grid-template: 1fr / 1fr;
        flex: 0 0 auto;
        margin: 0 auto;
        overflow: hidden;
        border: 1px solid #e2e5ef;
        border-radius: 8px;
        background: #232736;
        cursor: grab;
        touch-action: none;
    }

    .stage.dragging {
        cursor: grabbing;
    }

    .bg {
        grid-area: 1 / 1;
        background-repeat: no-repeat;
        pointer-events: none;
    }

    .loading {
        grid-area: 1 / 1;
        display: grid;
        place-items: center;
        color: #9aa3bd;
        font-size: 11px;
    }

    .empty {
        grid-area: 1 / 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 6px;
        border: none;
        background: transparent;
        color: #aab3d0;
        cursor: pointer;
        font: 11px/1.4 system-ui, sans-serif;
        transition: color 0.15s;
    }

    .empty:hover:not(:disabled) {
        color: #fff;
    }

    .empty:disabled {
        cursor: default;
    }

    .badge {
        grid-area: 1 / 1;
        justify-self: end;
        align-self: end;
        margin: 5px;
        padding: 1px 5px;
        border-radius: 3px;
        background: rgb(0 0 0 / 45%);
        color: #fff;
        font: 600 10px/1.5 ui-monospace, Consolas, monospace;
        pointer-events: none;
    }

    .ctrl {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .mini {
        flex: 0 0 auto;
        padding: 3px 8px;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
        background: #fff;
        color: var(--text-main, #3a4160);
        cursor: pointer;
        font: 11px/1.5 system-ui, sans-serif;
        transition: border-color 0.15s, color 0.15s;

    }

    .mini:hover:not(:disabled) {
        border-color: #c9d8ff;
        color: #2f6df6;
    }

    .mini:disabled {
        opacity: 0.45;
        cursor: not-allowed;
    }

    .pick {
        padding: 5px 0;
        border: 1px dashed #c9d8ff;
        border-radius: 5px;
        color: #2f6df6;
        cursor: pointer;
        font: 11px/1.5 system-ui, sans-serif;
        transition: background-color 0.15s, border-color 0.15s;
    }

    .pick:hover:not(:disabled) {
        border-color: #2f6df6;
    }

    .pick:disabled {
        opacity: 0.6;
        cursor: default;
    }

    .err {
        margin: 0;
        color: #e4544b;
        font-size: 10px;
    }

    .hint {
        margin: 0;
        color: var(--text-muted, #8a90a8);
        font-size: 10px;
        line-height: 1.35;
    }

    .foot {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        flex: 0 0 auto;
        padding: 5px 14px;
        border-top: 1px solid #eef0f7;
        background: #fbfcff;
    }

    .btn {
        padding: 3px 13px;
        border-radius: 5px;
        cursor: pointer;
        font: 600 11px/1.5 system-ui, sans-serif;
        transition: background-color 0.15s, border-color 0.15s, opacity 0.15s;
    }

    .btn.ghost {
        border: 1px solid #e2e5ef;
        background: #fff;
        color: var(--text-main, #3a4160);
    }

    .btn.ghost:hover {
        border-color: #c9d8ff;
    }

    .btn.primary {
        border: 1px solid #2f6df6;
        background: #2f6df6;
        color: #fff;
    }

    .btn.primary:hover:not(:disabled) {
        background: #2559d6;
    }

    .btn.primary:disabled {
        opacity: 0.45;
        cursor: not-allowed;
    }

    input[type='range'] {
        flex: 1;
        width: 100%;
        height: 14px;
        margin: 0;
        appearance: none;
        background: transparent;
        cursor: pointer;
    }

    input[type='range']:disabled {
        opacity: 0.45;
        cursor: not-allowed;
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
</style>