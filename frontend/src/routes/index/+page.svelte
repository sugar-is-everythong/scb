<script>
    import {Events} from '@wailsio/runtime';
    import Card from './clip/card.svelte';
    import {onMount} from 'svelte';
    import {localPathReplace} from "#lib/file.js";
    import {cbInset} from "#lib/cb.js";

    /** @type {import('#bindings/changeme/cb/models.ts').resCb[]} */
    const cblist = $state([]);
    Events.On('cb-change', (event) => {
        let cb = event.data;
        console.log(cb)

        // html 里 只有纯图片的时候，把 html 转成图片数组拆成多条复制，因为系统复制的时候就不支持多张图片
        if (cb.hasHTML) {
            const parser = new DOMParser()
            const doc = parser.parseFromString(cb.HTML?.Fragment, 'text/html')
            const children = doc.body.children
            const onlyImg = children.length > 0 && Array.from(children).every(el => el.tagName === 'IMG')
            if (onlyImg) {
                let images = []
                for (let i = 0; i < children.length; i++) {
                    images[i] = {
                        image: localPathReplace(children[i].src),
                        hasImage: true,
                        hasText: false,
                        text: '',
                        hasHTML: false,
                        HTML: null,
                        hasRTF: false,
                        RTF: null,
                        hasFiles: false,
                        files: null,
                    }
                }
                cb = images;
            }
        }

        if (Array.isArray(cb)) cblist.unshift(...cb)
        else cblist.unshift(cb)

        if (cblist.length > 100) cblist.pop();
    });

    /**
     * 固定条目，按固定先后排在列表最顶部
     * @type {import('#bindings/changeme/cb/models.ts').resCb[]}
     */
    const pinned = $state([]);

    /** 类型图标：先用单个字母/字占位，后续可换 SVG */
    const FILTERS = [
        {key: 'text', label: '文', title: '纯文本'},
        {key: 'html', label: 'H', title: 'HTML'},
        {key: 'rtf', label: 'R', title: 'RTF'},
        {key: 'image', label: '图', title: '图片'},
        {key: 'files', label: '件', title: '文件'}
    ];

    /** 当前过滤的类型，空字符串表示全部 */
    let filter = $state('');

    /** @param {import('#bindings/changeme/cb/models.ts').resCb} cb */
    const kindOf = (cb) =>
        cb.hasHTML ? 'html' : cb.hasFiles ? 'files' : cb.hasRTF ? 'rtf' : cb.hasImage ? 'image' : 'text';

    const view = $derived(
        [...pinned, ...cblist.filter((cb) => !pinned.includes(cb))].filter(
            (cb) => !filter || kindOf(cb) === filter
        )
    );

    /** 当前选中项下标，-1 表示未选中 */
    let select = $state(-1);

    /** 正在编辑的条目集合，编辑中不响应 Enter 写回 */
    const editingSet = new Set();

    /** 列表变化后把选中下标夹回有效范围 */
    $effect(() => {
        if (select >= view.length) select = view.length - 1;
    });

    /** 选中项变化时把它滚动到可视区中间（只依赖选中下标，列表增删不触发） */
    $effect(() => {
        if (select < 0) return;
        const el = document.querySelector('.list .card.select');
        el?.scrollIntoView({block: 'center', behavior: 'smooth'});
    });

    /** 焦点在输入框里时，键盘事件交还给输入框自己处理 */
    const inInput = (e) => {
        const t = e.target;
        return t instanceof HTMLElement && !!t.closest('textarea, input, select, [contenteditable="true"]');
    };

    /** 焦点在按钮上时，回车交给按钮自己处理（类型芯片、图标栏） */
    const inButton = (e) => {
        const t = e.target;
        return t instanceof HTMLElement && !!t.closest('button, a');
    };

    const documentOnkeydown = (e) => {
        if (e.isComposing || inInput(e)) return;

        e.preventDefault();
        if (e.key === 'ArrowDown') {
            if (select < view.length - 1) select++;
        } else if (e.key === 'ArrowUp') {
            if (select > 0) select--;
        } else if (e.key === 'Enter') {
            if (inButton(e)) return;
            const cb = view[select];
            if (cb) inset(cb);
        }
    };

    /** base64 字符串转字节数组：Go 的 []byte 过 JSON 是 base64，后端要的是 PNG 原始字节 */
    const base64ToBytes = (b64) => {
        const bin = atob(b64);
        const bytes = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
        return bytes;
    };

    /** 把条目写回剪贴板并触发粘贴，编辑中不写回 */
    const inset = async (cb) => {
        if (editingSet.size > 0) return;
        // 只支持【文本、图片】格式写回：HTML/RTF 卡片降级为纯文本
        // 【文件】 类型的在card组件单独处理，文件类型的逻辑比较多
        let type = 'text';
        let body = cb.text;
        console.log(cb)
        if (kindOf(cb) === 'image' && cb.image) {
            // type = 'image';
            // 本来应该使用写入图片类型，但是现在的设计图片预览使用的是【文件】或者【网络连接】所以直接使用 imageCORS 进行下载了
            type = 'imageCORS?url=' + cb.image;
            body = null
        } else if (!cb.text) {
            return; // 没有可写回的内容
        }

        cbInset(type, body, () => {
            select = -1
        })

    };

    /** 清空所有非固定条目 */
    const clearAll = () => {
        for (let i = cblist.length - 1; i >= 0; i--) {
            if (!pinned.includes(cblist[i])) cblist.splice(i, 1);
        }
    };

    /** @param {import('#bindings/changeme/cb/models.ts').resCb} cb */
    const togglePin = (cb) => {
        const i = pinned.indexOf(cb);
        if (i >= 0) pinned.splice(i, 1);
        else pinned.unshift(cb);
    };

    /** @param {import('#bindings/changeme/cb/models.ts').resCb} cb */
    const remove = (cb) => {
        pinned.splice(pinned.indexOf(cb), 1);
        cblist.splice(cblist.indexOf(cb), 1);
        editingSet.delete(cb);
    };

    let listDom = null;
    let hasScrollbar = $state(false);
    onMount(() => {
        const observer = new ResizeObserver(() => {
            hasScrollbar = listDom.scrollHeight > listDom.clientHeight;
        });
        observer.observe(listDom);
        return () => observer.disconnect();
    });
</script>

<svelte:document onkeydown={documentOnkeydown}/>

<main class="app">
    <header class="bar">
        <div class="filters">
            {#each FILTERS as f (f.key)}
                <button
                        class={['chip', { on: filter === f.key }]}
                        title={f.title}
                        onclick={() => (filter = filter === f.key ? '' : f.key)}
                >
                    {f.label}
                </button>
            {/each}
        </div>

        <div class="acts">
            <button class="ghost danger" onclick={clearAll}>清空</button>
        </div>
    </header>

    <section class="list" bind:this={listDom} style:margin-right={hasScrollbar?'4px':'10px'}>
        {#each view as cb, cbi (cb)}
            <Card {cb}
                  select={cbi === select}
                  pinned={pinned.includes(cb)}
                  onpin={() => togglePin(cb)}
                  ondelete={() => remove(cb)}
                  oninsert={() => inset(cb)}
                  oneditstate={(c, editing) => {
                      if (editing) editingSet.add(c);
                      else editingSet.delete(c);
                  }}
            />
        {/each}

        {#if view.length === 0}
            <p class="empty">复制点什么试试</p>
        {/if}
    </section>
</main>

<style>
    .app {
        box-sizing: border-box;
        display: flex;
        flex-direction: column;
        height: 100vh;
        overflow: hidden;
        background: rgb(244 245 249 / 0.73);
    }

    .bar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        height: 34px;
        padding: 0 12px;
        --wails-draggable: drag;
        user-select: none;
    }

    .filters {
        display: flex;
        gap: 4px;
        --wails-draggable: no-drag;
    }

    .chip {
        display: grid;
        place-items: center;
        width: 24px;
        height: 22px;
        padding: 0;
        border: 1px solid transparent;
        border-radius: 4px;
        background: rgb(255 255 255 / 55%);
        color: #8d94a8;
        font: 600 11px/1 system-ui,
        sans-serif;
        cursor: pointer;
        transition: background-color 0.15s,
        color 0.15s;
    }

    .chip:hover {
        background: #fff;
        color: #4a5169;
    }

    .chip.on {
        border-color: #c9d8ff;
        background: #e8f0ff;
        color: #2f6df6;
    }

    .acts {
        --wails-draggable: no-drag;

        display: flex;
        gap: 4px;
    }

    .ghost {
        padding: 1px 5px;
        padding-top: 2px;
        display: flex;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
        background: rgb(255 255 255 / 70%);
        color: #6d748c;
        cursor: pointer;
        font: 11px/1.4 system-ui, sans-serif;
        transition: background-color 0.15s,
        color 0.15s,
        border-color 0.15s;
        --wails-draggable: no-drag;
    }

    .ghost:hover {
        background: #fff;
    }

    .list {
        border-radius: 4px;
        display: flex;
        flex-direction: column;
        gap: 8px;
        flex: 1 1 auto;
        overflow-y: auto;
        margin: 10px;
        margin-top: 0;
    }

    .empty {
        margin-top: 40px;
        color: #a8aebe;
        font: 12px/1.5 system-ui, sans-serif;
        text-align: center;
    }
</style>
