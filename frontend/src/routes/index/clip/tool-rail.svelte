<script>
    /**
     * @type {{
     *   kind: 'text' | 'html' | 'image' | 'files' | 'rtf',
     *   pinned: boolean,
     *   onpin: () => void,
     *   onedit?: () => void,
     *   onsave?: () => void,
     *   onreveal?: () => void,
     *   ondelete: () => void
     * }}
     */
    let {kind, pinned = false, onpin, onedit, onsave, onreveal, ondelete} = $props();

    const editable = $derived(kind === 'text' || kind === 'html');
</script>

<div class="rail">
    <button class={['ico', { on: pinned }]} title={pinned ? '取消固定' : '固定'} onclick={onpin}>
        <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M9 4h6l-1 6 3 3H7l3-3-1-6Z"/>
            <path d="M12 13v7"/>
        </svg>
    </button>

    {#if editable}
        <button class="ico" title="编辑" onclick={onedit}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 20h4L20 8l-4-4L4 16v4Z"/>
                <path d="M14 6l4 4"/>
            </svg>
        </button>
    {/if}

    {#if kind === 'image'}
        <button class="ico" title="保存到本地" onclick={onsave}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M12 4v10"/>
                <path d="M8 10l4 4 4-4"/>
                <path d="M5 19h14"/>
            </svg>
        </button>
    {/if}

    {#if kind === 'files'}
        <button class="ico" title="粘贴文件" onclick={onreveal}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M3 7h6l2 2h10v10H3V7Z"/>
            </svg>
        </button>
    {/if}

    <button class="ico danger" title="删除" onclick={ondelete}>
        <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M4 7h16"/>
            <path d="M9 7V5h6v2"/>
            <path d="M6 7l1 13h10l1-13"/>
        </svg>
    </button>
</div>

<style>
    .rail {
        display: flex;
        flex-direction: column;
        gap: 2px;
        padding: 2px 0;
    }

    .ico {
        display: grid;
        place-items: center;
        width: 25px;
        height: 25px;
        padding: 0;
        border: 0;
        border-radius: 7px;
        background: transparent;
        color: #a2a8ba;
        cursor: pointer;
        transition: background-color 0.15s,
        color 0.15s,
        transform 0.15s;
    }

    .ico svg {
        width: 17px;
        height: 17px;
        fill: none;
        stroke: currentColor;
        stroke-width: 1.7;
        stroke-linecap: round;
        stroke-linejoin: round;
    }

    .ico:hover {
        background: #eef1f8;
        color: #4a5169;
        transform: translateY(-1px);
    }

    .ico:active {
        transform: scale(0.94);
    }

    .ico.on {
        color: #f0a71b;
        background: #fff6e0;
    }

    .danger:hover {
        background: #feecec;
        color: #d93838;
    }
</style>
