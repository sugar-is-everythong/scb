<script>
    import {onMount} from 'svelte';
    import {config, backgroundStyle, syncConfig} from '#lib/config.svelte.js';
    import WindowBg from '#lib/components/window-bg.svelte';

    let {children} = $props();

    // 接收其它窗口保存后的广播；背景、卡片底色都在这一层，三个窗口天然共用
    onMount(() => syncConfig());
</script>

<svelte:head>
</svelte:head>

<div id="win-root" style="--card-alpha: {config.cardOpacity}; {backgroundStyle(config)}">
    <WindowBg/>
    {@render children()}
</div>

<style>
    #win-root {
        position: relative;
        isolation: isolate;
        height: 100vh;
        border-radius: 6px;
        overflow: hidden;
    }
</style>
