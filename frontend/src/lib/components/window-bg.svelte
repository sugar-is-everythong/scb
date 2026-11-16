<script>
    import {config, bgLayout} from '#lib/config.svelte.js';

    // 图片背景单独一层：透明度要只作用在图片上，直接给窗口容器设 opacity 会把内容一起变淡

    // 原始尺寸与本窗口实际尺寸：各窗口大小不同、还会被用户改，几何按当前容器实时算
    let natW = $state(0);
    let natH = $state(0);
    let w = $state(0);
    let h = $state(0);

    $effect(() => {
        const url = config.bgMode === 'image' && config.bgImage ? `/wg/set/bgimg?v=${config.bgImage}` : '';
        natW = natH = 0;
        if (!url) return;
        let alive = true;
        const img = new Image();
        img.onload = () => {
            if (alive) {
                natW = img.naturalWidth;
                natH = img.naturalHeight;
            }
        };
        img.src = url;
        return () => {
            alive = false;
        };
    });

    // 尺寸未就绪时沿用 CSS 的 cover/center，就绪后按 bgScale/bgX/bgY 精确铺满
    const style = $derived.by(() => {
        const base = `background-image: url('/wg/set/bgimg?v=${config.bgImage}'); opacity: ${config.opacity};`;
        const g = bgLayout(config.bgScale, config.bgX, config.bgY, natW, natH, w, h);
        if (!g.dw) return base;
        return `${base} background-size: ${g.dw}px ${g.dh}px; background-position: ${config.bgX * 100}% ${config.bgY * 100}%;`;
    });
</script>

{#if config.bgMode === 'image' && config.bgImage}
    <div class="bg" style={style} bind:clientWidth={w} bind:clientHeight={h} aria-hidden="true"></div>
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
