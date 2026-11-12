<script>
    import {Events, Window} from '@wailsio/runtime';
    import ToolRail from '#lib/components/tool-rail.svelte';
    import {cbInset, withKind} from "#lib/cb.js";
    import CbItem from "#lib/components/cbItem.svelte";
    import {localPathReplace} from "#lib/file.js";

    /**
     * @type import('#bindings/scb/cb/models.ts').RecordJs
     */
    let cb = $state()
    Events.Emit("detail-window-open")
    Events.Once("detail-window-opened", (res) => {
        console.log(res)
        cb = res.data
    })

    const funcBtnHeight = '20px', funcBtnWidth = '15px'
    let isMax = $state(false)
    const toggleFull = () => {
        isMax = !isMax
        Window.ToggleMaximise()
    }

    /** 置顶 */
    let isTop = $state(false)
    const toggleTop = () => {
        isTop = !isTop
        Window.SetAlwaysOnTop(isTop)
    }

    let {kind, label} = $derived(withKind(cb))

    let pinned = $state('file')

    /** 'none' | 'text' | 'html' */
    let editing = $state('none');

    // 详情页的图片使用的是 Burl或者 Bdata
    const imgSrc = $derived.by(() => {
        if (cb?.image.bUrl) return cb.image.bUrl
        else return '/wg/getBimg/'+cb?.id
    });

    const htmlSource = $derived.by(() => {
        let hf = cb?.HTML?.Fragment ?? ''
        return localPathReplace(hf)
    });

    /** @param {Event & { currentTarget: HTMLTextAreaElement }} e */
    const onTextInput = (e) => (cb.text = e.currentTarget.value);

    /** @param {Event & { currentTarget: HTMLTextAreaElement }} e */
    const onHtmlInput = (e) => {
        if (cb?.HTML) cb.HTML.Fragment = e.currentTarget.value;
    };

    /** 切换编辑态，退出编辑时自动保存回写 */
    /** @param {"image" | "text" | "html" | "files" | "rtf"} kind */
    const toggleEdit = (kind) => {
        const next = kind === 'html' ? 'html' : 'text';
        if (editing === next) {
            // 退出编辑态，触发保存
            doSave();
        }
        editing = editing === next ? 'none' : next;
    };

    /**
     * @param {string} f 文件路径
     * @param {MouseEvent} e 鼠标事件
     */
    const fileClick = (f, e) => {
        e.stopPropagation()
        let body = JSON.stringify([f]);

        cbInset('files', body)
    }

    const onpin = () => {

    }
    const onreveal = () => {

    }

    /** 保存编辑内容，回传给 index 页面 */
    const doSave = () => {
        if (!cb?.id) return;
        Events.Emit("detail-save", {
            id: cb.id,
            text: cb.text,
            HTML: cb.HTML
        });
    };

    const ondelete = () => {
        if (!cb?.id) return;
        Events.Emit("detail-delete", {id: cb.id});
        Window.Close();
    }

    /** @param {MouseEvent} e */
    const tagClick = (e) => {
        e.stopPropagation()
        console.log('tagClick', cb)
        if (kind === 'html') {
            label = '文'
            kind = 'text'
        } else if (kind === 'text' && cb.hasHTML) {
            label = 'H'
            kind = 'html'
        }
    }
</script>

<main id="detail-box">
    <div id="detail-content" class="card-cfg">
        <CbItem
                {cb}
                {kind}
                {editing}
                {imgSrc}
                {htmlSource}
                {onHtmlInput}
                {onTextInput}
                {fileClick}
				inDetail={true}
                --max-height="100%"
        />
    </div>

    <div id="toolbar" class="card-cfg" ondblclick={toggleFull}>

        <div class="hr">
            <!-- 窗口置顶 -->
            <button onmouseup={toggleTop} title="窗口置顶"
                    class={["top-bar-func-button",{"top": isTop}]}>
                <svg viewBox="0 0 1024 1024" height={funcBtnHeight} width={funcBtnWidth}>
                    <path d="M695.168 73.92l254.912 254.976a33.792 33.792 0 0 1-28.8 57.344l-88.32-12.8-97.408 175.296-17.28 170.176c-4.48 40.32-7.552 62.4-9.152 66.24a189.44 189.44 0 0 1-45.568 69.12 33.792 33.792 0 0 1-42.624 4.288l-5.12-4.224-154.816-154.752-343.168 253.824a33.792 33.792 0 0 1-47.296-47.168l253.056-344.064L169.664 408.32a33.792 33.792 0 0 1 0-47.744 192 192 0 0 1 67.392-44.992l238.144-26.944 175.424-97.28-12.8-88.512a33.92 33.92 0 0 1 57.344-28.864z m24.128 119.744l1.664 11.008a33.792 33.792 0 0 1-12.736 31.616l-4.352 2.816-164.8 91.392-10.624 9.28A90.88 90.88 0 0 1 512 352l-261.696 31.552-6.08 3.648 147.648 147.712a33.792 33.792 0 0 1 3.328 43.904l-137.856 187.456 187.008-138.24 5.184-3.2a33.792 33.792 0 0 1 38.784 6.528l148.48 148.48c3.712-6.016 6.656-11.776 8.832-17.664l1.28-6.4c3.392-20.736 17.472-218.496 23.552-229.12l114.752-206.336 2.816-4.352a33.792 33.792 0 0 1 31.424-12.672l10.944 1.536-111.104-111.168z"
                    ></path>
                </svg>
            </button>
        </div>

        <div class="hr">
            {#if kind === 'text' || kind === 'html'}
                <button class="tag" onclick={tagClick} title="切换">
                    {#if kind === 'html'}
                        <img src="/imgs/html.png" alt="" draggable="false">
                    {:else}
                        文
                    {/if}
                </button>
            {/if}

            <ToolRail
                    {kind}
                    {ondelete}
                    onedit={() => toggleEdit(kind)}
                    onsave={doSave}
                    {onreveal}
            />
        </div>

        <div class="hr">

            <!-- 最小化按钮 -->
            <button onmouseup={Window.Minimise}
                    class={["top-bar-func-button",]}>
                <svg viewBox="0 0 1024 1024" height={funcBtnHeight} width={funcBtnWidth}>
                    <path d="M929.8 528.1H93.5c-15.5 0-28-12.5-28-28s12.5-28 28-28h836.3c15.5 0 28 12.5 28 28s-12.5 28-28 28z"></path>
                </svg>
            </button>

            <!-- 全屏切换按钮 -->
            <button onmouseup={toggleFull}
                    class={["top-bar-func-button"]}>
                {#if isMax}
                    <svg viewBox="0 0 1024 1024" height={funcBtnHeight} width={funcBtnWidth}>
                        <path d="M812.2 65H351.6c-78.3 0-142.5 61.1-147.7 138.1-77 5.1-138.1 69.4-138.1 147.7v460.6c0 81.6 66.4 148 148 148h460.6c78.3 0 142.5-61.1 147.7-138.1 77-5.1 138.1-69.4 138.1-147.7V213c0-81.6-66.4-148-148-148z m-45.8 746.3c0 50.7-41.3 92-92 92H213.8c-50.7 0-92-41.3-92-92V350.7c0-50.7 41.3-92 92-92h460.6c50.7 0 92 41.3 92 92v460.6z m137.8-137.7c0 47.3-35.8 86.3-81.8 91.4V350.7c0-81.6-66.4-148-148-148H260.2c5.1-45.9 44.2-81.8 91.4-81.8h460.6c50.7 0 92 41.3 92 92v460.7z"
                        ></path>
                    </svg>
                {:else}
                    <svg viewBox="0 0 1024 1024" height={funcBtnHeight} width={funcBtnWidth}>
                        <path d="M812.3 959.4H213.7c-81.6 0-148-66.4-148-148V212.9c0-81.6 66.4-148 148-148h598.5c81.6 0 148 66.4 148 148v598.5C960.3 893 893.9 959.4 812.3 959.4zM213.7 120.9c-50.7 0-92 41.3-92 92v598.5c0 50.7 41.3 92 92 92h598.5c50.7 0 92-41.3 92-92V212.9c0-50.7-41.3-92-92-92H213.7z"
                        ></path>
                    </svg>
                {/if}
            </button>

            <!-- 关闭按钮 -->
            <button onmouseup={Window.Close()}
                    class={["close" ,"top-bar-func-button" ]}>
                <svg viewBox="0 0 1024 1024" height={funcBtnHeight} width={funcBtnWidth}>
                    <path d="M96.6 915c-10.9-10.9-10.9-28.7 0-39.6L887 85c10.9-10.9 28.7-10.9 39.6 0 10.9 10.9 10.9 28.7 0 39.6L136.2 915c-10.9 10.9-28.6 10.9-39.6 0z"
                    ></path>
                    <path d="M887 915L96.6 124.6c-10.9-10.9-10.9-28.7 0-39.6 10.9-10.9 28.7-10.9 39.6 0l790.4 790.4c10.9 10.9 10.9 28.7 0 39.6-10.9 10.9-28.6 10.9-39.6 0z"
                    ></path>
                </svg>
            </button>

        </div>

    </div>
</main>

<style>
    #detail-box {
        max-width: 100%;
        height: 100%;
        display: grid;
        grid-template-columns: 1fr 40px;
        grid-gap: 5px;
        padding: 5px;
    }

    #detail-content {
        height: 100%;
        padding: 5px;
        overflow: auto;
        border-radius: 4px;
    }

    #toolbar {
        --wails-draggable: drag;
        border-radius: 4px;
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 5px 0;
        gap: 5px;

        .top-bar-func-button {
            aspect-ratio: 1;
            display: flex;
            align-items: center;
            justify-content: center;
            height: 23px;
            --wails-draggable: no-drag;
            border-radius: 6px;
            transition: .1s ease;
            user-select: none;
            -webkit-user-drag: none;
            cursor: default;
            fill: #444444;
            background-color: #fff0;

            &:nth-of-type(2) > svg {
                width: 14px;
            }

            &:nth-of-type(3) > svg {
                width: 13px;
            }
        }

        .top-bar-func-button:hover {
            background-color: var(--hover-color, #e3e3e3);
        }

        .close:hover {
            background-color: red;

            svg {
                fill: #fff;
            }
        }

        .top {
            background-color: #ffc013;
            fill: #fff !important;

            &:hover {
                background-color: #ffb700 !important;
                fill: #fff !important;
            }
        }

        .tag {
            font: 600 13px ui-monospace,
            SFMono-Regular,
            Consolas,
            monospace;
            cursor: default;
            user-select: none;
            width: 20px;
            height: 20px;
            color: #ffb700;
        }

        .hr {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            background-color: #fff;
            border-radius: 8px;
            padding: 5px;
            transform: translateX(-1px);
            --wails-draggable: no-drag;
        }

        .hr:last-of-type {
            position: sticky;
            top: 100vh;
        }
    }
</style>

