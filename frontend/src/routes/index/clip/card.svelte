<script>
    import ShadowHtml from './shadow-html.svelte';
    import ToolRail from './tool-rail.svelte';
    import {localPathReplace} from "#lib/file.js";
    import {cbInset} from "#lib/cb.js";

    /**
     * @type {{
     *   cb: import('#bindings/changeme/cb/models.ts').resCb,
     *   pinned?: boolean,
     *   select?: boolean, // 是否为当前选中项
     *   onpin: () => void,
     *   ondelete: () => void,
     *   oninsert?: () => void, // 请求写回剪贴板
     *   oneditstate?: (cb: import('#bindings/changeme/cb/models.ts').resCb, editing: boolean) => void
     * }}
     */
    let {cb, pinned = false, select = false, onpin, ondelete, oninsert, oneditstate} = $props();

    /** 'none' | 'text' | 'html' */
    let editing = $state('none');

    const kind = $derived(
        cb.hasHTML ? 'html' : cb.hasFiles ? 'files' : cb.hasRTF ? 'rtf' : cb.hasImage ? 'image' : 'text'
    );

    const label = $derived({html: 'HTML', files: '文件', rtf: 'RTF', image: '图片', text: '文本'}[kind]);

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

    const onreveal = () => {
        console.log(cb)
    }

    /**
     * @param {string} f 文件路径
     * @param {MouseEvent} e 鼠标事件
     */
    const fileClick = (f, e) => {
        e.stopPropagation()
        let body = JSON.stringify([f]);

        cbInset('files',body)
    }
</script>

<article class={['card', { pinned } ,{select}]}>
    <div class="body" onclick={onBodyClick}>
        <span class="tag">{label}</span>

        {#if kind === 'html'}
            {#if editing === 'html'}
                <textarea class="editor" value={htmlSource} oninput={onHtmlInput}></textarea>
            {:else}
                <ShadowHtml html={htmlSource}/>
            {/if}
        {:else if kind === 'image'}
            <img class="image" src={imgSrc} alt="">
        {:else if kind === 'files'}
            <ul class="files">
                {#each cb.files as file (file)}
                    <li onclick={(e) => fileClick(file,e)} title={file}>{file}</li>
                {/each}
            </ul>
        {:else if kind === 'rtf'}
            <p class="text muted">RTF 富文本（暂不渲染）</p>
        {:else if editing === 'text'}
            <textarea class="editor" value={cb.text} oninput={onTextInput}></textarea>
        {:else}
            <p class="text">{cb.text}</p>
        {/if}
    </div>

    <ToolRail
            {kind}
            {pinned}
            onpin={onpin}
            onedit={() => toggleEdit(kind)}
            onsave={() => console.log('save', cb)}
            {onreveal}
            {ondelete}
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
    }

    .card.select {
        border: 1px #4b7eff solid;
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
    }

    .text {
        max-height: 120px;
        overflow: hidden;
        color: #2b2f3c;
        font: 13px/1.55 system-ui, 'Segoe UI', sans-serif;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
    }

    .muted {
        color: #98a0b4;
    }

    .image {
        max-width: 100%;
        max-height: 140px;
        object-fit: cover;
    }

    .files {
        color: #4a5169;
        font: 12px/1.5 ui-monospace, Consolas, monospace;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .files li {
        overflow-wrap: anywhere;
        cursor: default;
        list-style: none;
        position: relative;
        padding: 0 5px;
        padding-left: 25px;
        padding-bottom: 4px;
        margin-bottom: 4px;

        &:last-of-type {
            padding-bottom: 4px;
            margin-bottom: 4px;
        }

        &:hover {
            background: #f1f3f9;
            border-radius: 4px;
        }

        &:before {
            position: absolute;
            left: 2px;
            top: 0;
            content: '';
            background-image: url("/imgs/wfjmjw.png");
            background-size: contain;
            background-repeat: no-repeat;
            width: 20px;
            height: 20px;
        }

        &:after {
            position: absolute;
            bottom: 0;
            left: 25px;
            content: '';
            height: 1px;
            width: calc(100% - 25px);
            border-bottom: 1px dashed #e8eaf2;
        }

        &:last-of-type:after {
            display: none;
        }

    }

    .editor {
        width: 100%;
        min-height: 80px;
        padding: 6px 8px;
        border: 1px solid #d8ddeb;
        border-radius: 6px;
        color: #2b2f3c;
        font: 12px/1.5 ui-monospace, Consolas, monospace;
        resize: vertical;
    }

    .editor:focus {
        border-color: #2f6df6;
        outline: none;
    }
</style>
