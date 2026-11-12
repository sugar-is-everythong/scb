<script>
    import {Window} from '@wailsio/runtime';
    import {config} from '#lib/config.svelte.js';

    let {children} = $props();

    // 窗口大小只对“粘一下”生效；放到根 layout 会把设置、详情窗口也改成这个尺寸
    $effect(() => {
        Window.SetSize(config.width, config.height);
    });
</script>

{@render children()}
