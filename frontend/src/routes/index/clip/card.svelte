<script>
    import ToolRail from '#lib/components/tool-rail.svelte';
    import {localPathReplace} from "#lib/file.js";
    import {cbInset, withKind} from "#lib/cb.js";
    import {Fetch} from "#lib/fetch.js";
    import {Events} from "@wailsio/runtime";
    import CbItem from '#lib/components/cbItem.svelte';
    import {config} from '#lib/config.svelte.js';

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

    // 关闭“显示富文本”后，HTML 卡片降级成纯文本样式展现，标签仍标 HTML
    const viewKind = $derived(kind === 'html' && !config.richText ? 'text' : kind);

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

    /** @param {MouseEvent} e */
    const tagClick = (e) => {
        e.stopPropagation()
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
            Events.Once("detail-window-open", (res) => {
                if (res.sender === windowName) {
                    Events.Emit("detail-window-opened", $state.snapshot(cb))
                }
            })
        })
    }
</script>

<article class={['card-cfg', { pinned }, {select}]}>
    <div class="body" onclick={onBodyClick}>
        <span class="tag" onclick={tagClick} style:cursor={label==='HTML'?'pointer':''}>{label}</span>

        <CbItem
                {cb}
                kind={viewKind}
                {editing}
                imgSrc={cb.image.sUrl}
                {htmlSource}
                {onHtmlInput}
                {onTextInput}
                {fileClick}
        />
    </div>

    <ToolRail
            {kind}
            {pinned}
            useSettings
            onpin={onpin}
            onedit={() => toggleEdit(kind)}
            onsave={() => console.log('save', cb)}
            {onreveal}
            {ondelete}
            {openWindow}
    />
</article>

<style>
    .card-cfg {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        grid-gap: 2px;
        padding: 2px 8px;
        padding-right: 2px;
        transition: border-color 0.15s,
        box-shadow 0.15s;
        --wails-draggable: no-drag;
    }

    .card-cfg.select {
        border: 1px #4b7eff solid !important;
    }

    .card-cfg:hover {
        box-shadow: 0 2px 8px rgb(23 29 51 / 7%);
    }

    .card-cfg.pinned {
        border-color: #f5d488;
    }

    .body {
        min-width: 0;
    }

    .tag {
        display: inline-block;
        border-radius: 4px;
        color: var(--text-main);
        margin-bottom: 2px;
        font: 600 10px/1.6 ui-monospace,
        SFMono-Regular,
        Consolas,
        monospace;
        letter-spacing: 0.06em;
        text-transform: uppercase;
        user-select: none;
    }
</style>
