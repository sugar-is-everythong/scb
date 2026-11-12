<script>
    import {config} from '#lib/config.svelte.js';

    // 图片背景单独一层：透明度要只作用在图片上，直接给窗口容器设 opacity 会把内容一起变淡
</script>

{#if config.bgMode === 'image' && config.bgImage}
    <div class="bg"
         style="background-image: url('/wg/set/bgimg?v={config.bgImage}'); opacity: {config.opacity}"
         aria-hidden="true"></div>
{/if}

<style>
    .bg {
        position: absolute;
        inset: 0;
        z-index: -1;
        pointer-events: none;
        background-position: center;
        background-size: cover;
    }
</style>
