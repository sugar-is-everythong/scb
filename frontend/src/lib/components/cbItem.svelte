<script>
    import ShadowHtml from "#lib/components/shadow-html.svelte";
    import { setContext } from "svelte";

    let {cb, kind, editing, htmlSource, imgSrc, onHtmlInput, onTextInput, fileClick, inDetail=false, ...others} = $props();
	setContext("inDetail",inDetail)
</script>

{#if cb !== undefined}
    <div class="cb-item-box" style:height={inDetail ? '100%' : undefined}>

        {#if kind === 'html'}
            {#if editing === 'html'}
                <textarea class="editor" value={htmlSource} oninput={onHtmlInput}></textarea>
            {:else}
                <ShadowHtml html={htmlSource}/>
            {/if}
        {:else if kind === 'image'}
            <!-- 在接受数据的时候遇到只包含图片的html或者file会转换成图片显示 -->
            <img class="image" src={imgSrc} alt="">
        {:else if kind === 'files'}
            <ul class="files">
                {#each cb.files as file (file)}
                    <li onclick={(e) => fileClick(file,e)} title='单文件粘贴'>{file}</li>
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
{/if}

<style>

    .cb-item-box {
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .text {
        width: 100%;
        max-height: 120px;
        color: #2b2f3c;
        font: 13px/1.55 system-ui, 'Segoe UI', sans-serif;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
        overflow: auto;
    }

    .muted {
        color: #98a0b4;
    }

    .image {
        display: block;
        max-width: 100%;
        max-height: var(--max-height, 140px);
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
        min-height: var(--height , 80px);
        height: 100%;
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
