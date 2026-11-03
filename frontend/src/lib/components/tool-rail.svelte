<script>
    import {Dialogs} from "@wailsio/runtime";
    import {Fetch} from "#lib/fetch.js";

    /**
     * @type {{
     *   kind: 'text' | 'html' | 'image' | 'files' | 'rtf',
     *   pinned: boolean,
     *   onpin?: () => void,
     *   onedit?: () => void,
     *   onsave?: () => void,
     *   onreveal?: () => void,
     *   ondelete?: () => void
     *   openWindow?: () => void
     * }}
     */
    let {kind, pinned = false, onpin, onedit, onsave, onreveal, ondelete, openWindow} = $props();

    const editable = $derived(kind === 'text' || kind === 'html'|| kind === 'image');

    const onsaveClick = () => {
        Dialogs.SaveFile({
            CanChooseDirectories:true,
        }).then((res)=> {
            Fetch('/cb/')
        })
        onsave?.()
    }
</script>

<div class="rail">
    {#if onpin}
        <button class={['ico', { on: pinned }]} title={pinned ? '取消固定' : '固定'} onclick={onpin}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M9 4h6l-1 6 3 3H7l3-3-1-6Z"/>
                <path d="M12 13v7"/>
            </svg>
        </button>
    {/if}

    {#if editable}
        <button class="ico" title="编辑" onclick={onedit}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 20h4L20 8l-4-4L4 16v4Z"/>
                <path d="M14 6l4 4"/>
            </svg>
        </button>
    {/if}

    {#if openWindow}
        <button class={['ico']} title='在小窗打开' onclick={openWindow}>
            <svg viewBox="-100 -100 1200 1000" width="20" height="20">
                <path d="M107.789474 0h592.842105c59.284211 0 107.789474 48.505263 107.789474 107.789474v592.842105c0 59.284211-48.505263 107.789474-107.789474 107.789474H107.789474c-59.284211 0-107.789474-48.505263-107.789474-107.789474V107.789474c0-59.284211 48.505263-107.789474 107.789474-107.789474z"
                      fill="#6DA1FF" p-id="10098"></path>
                <path d="M592.842105 485.052632h323.368421c59.284211 0 107.789474 48.505263 107.789474 107.789473v323.368421c0 59.284211-48.505263 107.789474-107.789474 107.789474H592.842105c-59.284211 0-107.789474-48.505263-107.789473-107.789474V592.842105c0-59.284211 48.505263-107.789474 107.789473-107.789473z"
                      fill="#D4E3FF" p-id="10099"></path>
                <path d="M420.378947 199.410526v194.021053c0 16.168421-13.473684 26.947368-26.947368 26.947368H199.410526c-16.168421 0-26.947368-13.473684-26.947368-26.947368 0-16.168421 13.473684-26.947368 26.947368-26.947368h126.652632L142.821053 180.547368c-10.778947-10.778947-10.778947-26.947368 0-37.726315 10.778947-10.778947 26.947368-10.778947 37.726315 0l185.936843 185.936842V199.410526c0-8.084211 2.694737-13.473684 8.08421-18.863158 5.389474-5.389474 10.778947-8.084211 18.863158-8.08421 16.168421 0 26.947368 13.473684 26.947368 26.947368z"
                      fill="#FFFFFF" p-id="10100"></path>
            </svg>
        </button>
    {/if}

  <!--
    下载先不做了，没啥必要
    {#if kind === 'image'}
        <button class="ico" title="保存到本地" onclick={onsaveClick}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M12 4v10"/>
                <path d="M8 10l4 4 4-4"/>
                <path d="M5 19h14"/>
            </svg>
        </button>
    {/if}
-->
    {#if kind === 'files'}
        <button class="ico" title="粘贴全部文件" onclick={onreveal}>
            <img src="/imgs/filelink.png" alt="">
        </button>
    {/if}

    {#if ondelete}
        <button class="ico danger" title="删除" onclick={ondelete}>
            <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M4 7h16"/>
                <path d="M9 7V5h6v2"/>
                <path d="M6 7l1 13h10l1-13"/>
            </svg>
        </button>
    {/if}
</div>

<style>
    .rail {
        display: flex;
        flex-direction: column;
        gap: 2px;
        padding: 2px 0;
        --wails-draggable: no-drag;
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

    .ico {
        & svg {
            width: 17px;
            height: 17px;
            fill: none;
            stroke: currentColor;
            stroke-width: 1.7;
            stroke-linecap: round;
            stroke-linejoin: round;
        }

        & img {
            width: 17px;
            height: 17px;
        }
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
