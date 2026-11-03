<script>
    /** @type {{ html?: string }} */
    let {html = ''} = $props();

    /**
     * 把 HTML 片段渲染进影子根：样式、选择器、脚本都与主界面隔离
     * @type {import('svelte/attachments').Attachment}
     */
    const shadow = (node) => {
        const root = node.shadowRoot ?? node.attachShadow({mode: 'open'});
        root.innerHTML = `${html}`;
    };
</script>

<div class="html">
    <div style="width: fit-content;" {@attach shadow}></div>
</div>

<style>
    .html {
        max-height: 250px;
        overflow: auto;
    }
</style>
