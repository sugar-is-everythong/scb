<script>
    import CloseIcon from './close-icon.svelte';

    /**
     * 通用关闭按钮：× 图标 + 悬停变红，设置窗标题栏与对话框右上角共用。
     * 颜色继承主题变量 --text-main，尺寸固定 22×22，图标大小由 size 控制。
     * @type {{onclick?: () => void, label?: string, size?: number}}
     */
    let {onclick, label = '关闭', size = 11} = $props();
</script>

<button class="close-btn" type="button" title={label} aria-label={label} {onclick}>
    <CloseIcon width={size} height={size}/>
</button>

<style>
    .close-btn {
        display: grid;
        place-items: center;
        width: 22px;
        height: 22px;
        padding: 0;
        border: none;
        border-radius: 4px;
        background: transparent;
        fill: var(--text-main, #3a4160);
        cursor: pointer;
        transition: background-color 0.15s, fill 0.15s;
        --wails-draggable: no-drag;
    }

    .close-btn:hover {
        background: #f04b4b;
        fill: #fff;
    }
</style>