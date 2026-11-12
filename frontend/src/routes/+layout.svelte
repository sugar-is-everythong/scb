<script>
    import {onMount} from 'svelte';
    import {config, backgroundStyle, textStyle, syncConfig} from '#lib/config.svelte.js';
    import WindowBg from '#lib/components/window-bg.svelte';
    import '#lib/assets/style.css';

    let {children} = $props();

    // 接收其它窗口保存后的信息
    onMount(() => syncConfig());
</script>

<svelte:head>
</svelte:head>

<div id="win-root" style="--card-alpha: {config.cardOpacity}; {textStyle(config)} {backgroundStyle(config)}">
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
