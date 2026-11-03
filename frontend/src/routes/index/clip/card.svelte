<script>
    import ToolRail from '#lib/components/tool-rail.svelte';
    import {localPathReplace} from "#lib/file.js";
    import {cbInset, withKind} from "#lib/cb.js";
    import {Fetch} from "#lib/fetch.js";
    import {Events} from "@wailsio/runtime";
    import CbItem from '#lib/components/cbItem.svelte';

    /**
     * @type {{
     *   cb: import('#bindings/scb/cb/models.ts').RecordJs,
     *   pinned?: boolean,
     *   select?: boolean, // 是否为当前选中项
     *   onpin: () => void,
     *   ondelete?: () => void,
     *   oninsert?: () => void, // 请求写回剪贴板
     *   oneditstate?: (cb: import('#bindings/scb/cb/models.ts').RecordJs, editing: boolean) => void
     * }}
     */
    let {cb, pinned = false, select = false, onpin, ondelete, oninsert, oneditstate} = $props();

    /** 'none' | 'text' | 'html' */
    let editing = $state('none');

    let {kind, label} = $state(withKind(cb))

    // Go 的 []byte 过 JSON 是 base64，可直接拼 data URL
    const imgSrc = $derived.by(() => {
        if (cb.image?.startsWith('/') || cb.image?.startsWith('data:') || cb.image?.startsWith('blob:') || cb.image?.startsWith('http')) return cb.image
        return `data:image/png;base64,${cb.image}`
    });

    const htmlSource = $derived.by(() => {
        let hf = cb.HTML?.Fragment ?? ''
        return localPathReplace(hf)
    });

    /** @param {Event & { currentTarget: HTMLTextAreaElement }} e */
    const onTextInput = (e) => (cb.text = e.currentTarget.value);

    /** @param {Event & { currentTarget: HTMLTextAreaElement }} e */
    const onHtmlInput = (e) => {
        if (cb.HTML) cb.HTML.Fragment = e.currentTarget.value;
    };

    /** 切换编辑态，并把编辑状态上报给页面（编辑中禁止回车写回） */
    /** @param {"image" | "text" | "html" | "files" | "rtf"} kind */
    const toggleEdit = (kind) => {
        const next = kind === 'html' ? 'html' : 'text';
        editing = editing === next ? 'none' : next;
        oneditstate?.(cb, editing !== 'none');
    };

    /** 点击卡片正文写回剪贴板，编辑中不写回 */
    const onBodyClick = () => {
        if (editing !== 'none') return;
        oninsert?.();
    };

    const tagClick = (e) => {
        e.stopPropagation()
        console.log('tagClick', cb)
        if (kind === 'html') {
            label = '文本'
            kind = 'text'
        } else if (kind === 'text' && cb.hasHTML) {
            label = 'HTML'
            kind = 'html'
        }
    }

    // 粘贴全部文件
    const onreveal = () => {
        cbInset('files', JSON.stringify(cb.files))
    }

    /**
     * @param {string} f 文件路径
     * @param {MouseEvent} e 鼠标事件
     */
    const fileClick = (f, e) => {
        e.stopPropagation()
        let body = JSON.stringify([f]);

        cbInset('files', body)
    }

    const openWindow = () => {
        Fetch('/window/create', {method: 'POST'}).then((windowName) => {
            Events.Once("detail-open", (res) => {
                if (res.sender === windowName) {
                    Events.Emit("detail-opened", $state.snapshot(cb))
                }
            })
        })
    }
</script>

<article class={['card', { pinned } ,{select}]}>
    <div class="body" onclick={onBodyClick}>
        <span class="tag" onclick={tagClick}>{label}</span>

        <CbItem
                {cb}
                {kind}
                {editing}
                {imgSrc}
                {htmlSource}
                {onHtmlInput}
                {onTextInput}
                {fileClick}
        />
    </div>

    <ToolRail
            {kind}
            {pinned}
            onpin={onpin}
            onedit={() => toggleEdit(kind)}
            onsave={() => console.log('save', cb)}
            {onreveal}
            {ondelete}
            {openWindow}
    />
</article>

<style>
    .card {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        grid-gap: 2px;
        padding: 2px 8px;
        padding-right: 2px;
        background: #fff;
        border: 1px solid #e8eaf2;
        border-radius: 6px;
        transition: border-color 0.15s,
        box-shadow 0.15s;
        --wails-draggable: no-drag;
    }

    .card.select {
        border: 1px #4b7eff solid !important;
    }

    .card:hover {
        border-color: #d8ddeb;
        box-shadow: 0 2px 8px rgb(23 29 51 / 7%);
    }

    .card.pinned {
        border-color: #f5d488;
    }

    /* 图标栏平时淡出，悬停卡片才完全显现（参考 Win11 剪贴板） */
    .card :global(.rail) {
        opacity: 0.45;
        transition: opacity 0.15s;
    }

    .card:hover :global(.rail),
    .card.pinned :global(.rail) {
        opacity: 1;
    }

    .body {
        min-width: 0;
    }

    .tag {
        display: inline-block;
        padding: 1px 5px;
        border-radius: 4px;
        background: #f1f3f9;
        color: #7c839b;
        margin-bottom: 2px;
        font: 600 10px/1.6 ui-monospace,
        SFMono-Regular,
        Consolas,
        monospace;
        letter-spacing: 0.06em;
        text-transform: uppercase;
        cursor: default;
        user-select: none;
    }
</style>
