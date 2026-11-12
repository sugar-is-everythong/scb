<script>
    import {Events, Window} from '@wailsio/runtime';
    import Card from './clip/card.svelte';
    import {onMount} from 'svelte';
    import {cbInset, deleteLocal, saveLocal, updateStoreAll, withKind} from "#lib/cb.js";
    import {Fetch} from "#lib/fetch.js";
    import {config} from '#lib/config.svelte.js';

    /** @type {import('#bindings/scb/cb/models.ts').RecordJs[]} */
    const cblist = $state([]);
    Events.On('cb-change', (event) => {
        let cb = event.data;
        console.log(cb)
        cblist.unshift(cb)

        if (cblist.length > config.maxRecords) cblist.pop();

        // 当有新内容的时候滚动到顶部
        document.querySelector('.list')?.scrollTo(0, 0);
        select = -1;
    });

    /**
     * 固定条目，按固定先后排在列表最顶部
     * @type {import('#bindings/scb/cb/models.ts').RecordJs[]}
     */
    const pinned = $state([]);

    /** 类型图标：先用单个字母/字占位，后续可换 SVG */
    const FILTERS = [
        {key: 'text', label: '文', title: '纯文本'},
        {key: 'html', label: 'H', icon: '/imgs/html.png', title: 'HTML'},
        // {key: 'rtf', label: 'R', title: 'RTF'}, 暂时没看到有RTF的
        {key: 'image', label: '️️️图片', icon: '/imgs/image.png', title: '图片'},
        {key: 'files', label: '文件', icon: '/imgs/wfjmjw.png', title: '文件'}
    ];

    /** 当前过滤的类型，空字符串表示全部 */
    let filter = $state('');
    /** 搜索关键字，空字符串表示不过滤；只匹配 RecordJs.text */
    let keyword = $state('');

    /** @param {import('#bindings/scb/cb/models.ts').RecordJs} cb */
    const kindOf = (cb) => withKind(cb).kind

    /** 只按 text 字段做不区分大小写的子串匹配 */
    const matchKeyword = (cb) => {
        if (!keyword) return true;
        return (cb.text ?? '').toLowerCase().includes(keyword.toLowerCase());
    };

    const matchType = (cb) => !filter || kindOf(cb) === filter;

    const pinnedView = $derived(pinned.filter((cb) => matchType(cb) && matchKeyword(cb)));
    const listView = $derived(cblist.filter((cb) => matchType(cb) && matchKeyword(cb)));
    const view = $derived([...pinnedView, ...listView]);

    /** 底栏状态：钉住条目 + 临时条目总数 */
    const total = $derived(pinned.length + cblist.length);

    /** 顶部过滤条各部件是否可见，全关时留一个最小高度 */
    const hasFilters = $derived(FILTERS.some((f) => config.filterBar[f.key]));
    const hasBar = $derived(hasFilters || config.filterBar.search || config.filterBar.clear);

    // 过滤开关被关掉后，如果还停在被隐藏的过滤/关键字上，列表会被看不见的条件卡住，这里主动恢复
    $effect(() => {
        if (filter && !config.filterBar[filter]) filter = '';
        if (!config.filterBar.search && keyword) keyword = '';
    });

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

    /**
     * 焦点在输入框里时，键盘事件交还给输入框自己处理
     * @param {KeyboardEvent} e
     */
    const inInput = (e) => {
        const t = e.target;
        return t instanceof HTMLElement && !!t.closest('textarea, input, select, [contenteditable="true"]');
    };

    let hasSelection = false // 这次是不是有选中
    let lastHasSelection = false // 上次是不是有选中，优化手感
    /** @param {Event} e */
    const onselectionchange = (e) => {
        let a = document?.getSelection()?.toString();
        hasSelection = !!a?.length
    }

    /**
     * 焦点在按钮上时，回车交给按钮自己处理（类型过滤按钮，清空等）
     * @param {KeyboardEvent} e
     */
    const inButton = (e) => {
        const t = e.target;
        return t instanceof HTMLElement && !!t.closest('button, a');
    };

    /** @type {HTMLElement} */
    let searchDom

    /** @param {KeyboardEvent} e */
    const documentOnkeydown = (e) => {
        if (e.key === 'Escape' && document.activeElement === searchDom) {
            keyword = '';
            searchDom.blur();
            return;
        }
        if (e.isComposing || inInput(e)) return;

        if (e.key === 'ArrowDown') {
            e.preventDefault();
            if (select < view.length - 1) select++;
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (select > 0) select--;
        } else if (e.key === 'Enter') {
            e.preventDefault();
            if (editingSet.size === 0 && select < 0) {
                // 搜索框可能被设置隐藏，没有输入框时不要把按键卡在这里
                if (searchDom) {
                    searchDom.focus();
                    searchDom.select();
                }
                return
            }

            if (inButton(e)) return;
            const cb = view[select];
            if (cb) inset(cb);
        } else if (e.key === 'Escape') {
            if (editingSet.size > 0) return;
            e.preventDefault();

            select = -1;
            Window.Hide()
        }
    };

    /** base64 字符串转字节数组：Go 的 []byte 过 JSON 是 base64，后端要的是 PNG 原始字节
     * @param {string} b64
     */
    const base64ToBytes = (b64) => {
        const bin = atob(b64);
        const bytes = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
        return bytes;
    };

    /**
     * 把条目写回剪贴板并触发粘贴，编辑中不触发
     * @param {import('#bindings/scb/cb/models.ts').RecordJs} cb
     */
    const inset = async (cb) => {
        if (hasSelection || lastHasSelection) {
            lastHasSelection = hasSelection;
            return
        }
        if (editingSet.size > 0) return
        // 只支持【文本、图片】格式写回：HTML/RTF 卡片降级为纯文本
        // 【文件】 类型的在card组件单独处理，文件类型的逻辑比较多
        let type = 'text';
        let body = cb.text;
        if (cb.hasImage) {
            type = 'image'; // 图片的要在go段进行处理
            body = {
                id: cb.id,
                bUrl: cb.image.bUrl
            }
        } else if (cb.hasHTML) {
            body = cb;
            type = 'html';
        } else if (cb.hasFiles) {
            type = 'files';
            body = cb.files;
        }

        cbInset(type, body, () => {
            select = -1
        })

    };

    /** 清空所有非固定条目 */
    const clearAll = () => {
        cblist.length = 0;
    };

    /**
     * 从 store 重新拉取钉住列表
     */
    const reloadPinned = async () => {
        const res = await Fetch('/cb/getlocal');
        pinned.length = 0;
        pinned.push(...res);
    };

    /**
     *  @param {string} w 区分 pinned 还是 cblist
     *  @param {import('#bindings/scb/cb/models.ts').RecordJs} cb
     */
    const togglePin = async (w, cb) => {
        if (w === 'pinned') {
            // 取消钉住：用 id 定位当前 pinned 下标（即 store 索引）
            const idx = pinned.findIndex(c => c.id === cb.id);
            if (idx < 0) return;
            await deleteLocal(idx);
            await reloadPinned();
        } else {
            // 钉住：用 id 从 cblist 移除，保存到 store
            const idx = cblist.findIndex(c => c.id === cb.id);
            if (idx < 0) return;
            cblist.splice(idx, 1);
            await saveLocal($state.snapshot(cb));
            await reloadPinned();
        }
    };

    /**
     * @param {import('#bindings/scb/cb/models.ts').RecordJs} cb
     */
    const remove = (cb) => {
        const idx = cblist.findIndex(c => c.id === cb.id);
        if (idx < 0) return;
        cblist.splice(idx, 1);
        editingSet.delete(cb);
    };

    /** @type {Element} */
    let listDom;
    let hasScrollbar = $state(false);

    // 详情窗口编辑保存回写
    Events.On('detail-save', (event) => {
        const {id, text, HTML} = event.data;
        // 先查 pinned，再查 cblist
        let item = pinned.find(c => c.id === id);
        const inStore = !!item;
        if (!item) item = cblist.find(c => c.id === id);
        if (!item) return;

        item.text = text;
        if (HTML) item.HTML = HTML;

        // pinned 条目同步 store
        if (inStore) {
            updateStoreAll($state.snapshot(pinned));
        }
    });

    // 详情窗口删除
    Events.On('detail-delete', (event) => {
        const {id} = event.data;
        let idx = pinned.findIndex(c => c.id === id);
        if (idx >= 0) {
            deleteLocal(idx).then(reloadPinned).catch(e => console.error('删除失败', e));
            return;
        }
        idx = cblist.findIndex(c => c.id === id);
        if (idx >= 0) {
            const item = cblist[idx];
            cblist.splice(idx, 1);
            editingSet.delete(item);
        }
    });
    onMount(() => {
        // 加载 store 数据，初始化 pinned（顺序与 store 一致）
        reloadPinned();

        const observer = new ResizeObserver(() => {
            hasScrollbar = listDom.scrollHeight > listDom.clientHeight;
        });
        observer.observe(listDom);
        return () => observer.disconnect();
    });
</script>

<svelte:document onkeydown={documentOnkeydown} {onselectionchange}/>

<main class="app">
    <header class="bar" style:height={hasBar?'':'10px'}>
        {#if hasFilters}
            <div class="filters">
                {#each FILTERS as f (f.key)}
                    {#if config.filterBar[f.key]}
                        <button
                                class={['chip', { on: filter === f.key, file: f.key === 'files' }]}
                                title={f.title}
                                onclick={() => (filter = filter === f.key ? '' : f.key)}
                        >
                            {#if f.icon}
                                <img src={f.icon} alt="">
                            {:else}
                                {f.label}
                            {/if}
                        </button>
                    {/if}
                {/each}
            </div>
        {/if}

        {#if config.filterBar.search}
            <div class="search">
                <input type="text" bind:this={searchDom} bind:value={keyword} placeholder="搜索文本">
            </div>
        {/if}

        {#if config.filterBar.clear}
            <div class="acts">
                <button class="ghost danger" onclick={clearAll}>清空</button>
            </div>
        {/if}
    </header>

    <section class="list" bind:this={listDom} style:margin-right={hasScrollbar?'4px':'10px'}>
        {#each pinnedView as cb, cbi (cb)}
            <Card {cb}
                  select={cbi === select}
                  pinned
                  onpin={() => togglePin('pinned',cb)}
                  oninsert={() => inset(cb)}
                  oneditstate={(c, editing) => {
                      if (editing) editingSet.add(c);
                      else editingSet.delete(c);
                  }}
            />
        {/each}

        {#each listView as cb, cbi (cb)}
            <Card {cb}
                  select={cbi + pinnedView.length === select}
                  onpin={() => togglePin('cblist',cb)}
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

    <footer class="foot">
        <span class="state">共 {total} 条</span>
        <button class="gear" title="设置" onclick={() => Fetch('/window/set', {method: 'POST'})}>
            <svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor"
                 stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="3"></circle>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
            </svg>
        </button>
    </footer>
</main>

<style>
    .app {
        box-sizing: border-box;
        display: flex;
        flex-direction: column;
        height: 100vh;
        overflow: hidden;
        --wails-draggable: drag;
    }

    .bar {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 7px;
        flex: 0 0 auto;
        height: 34px;
        padding: 0 12px;
        user-select: none;
    }

    .filters {
        display: flex;
        align-items: center;
        gap: 4px;
        --wails-draggable: no-drag;

        img {
            width: 16px;
            height: 16px;
        }

        & .file > img {
            height: 14px;
        }
    }

    .search {
        flex: 1;
        max-width: 300px;
        display: flex;
        align-items: center;
        --wails-draggable: no-drag;

        input {
            border-radius: 4px;
            border: 1px solid #e2e5ef;
            padding: 3px 5px;
            width: 100%;
        }
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
        flex-shrink: 0;
        padding: 1px 5px;
        padding-top: 2px;
        border: 1px solid #e2e5ef;
        border-radius: 4px;
        background: rgb(255 255 255 / 70%);
        color: #6d748c;
        cursor: pointer;
        font: 11px/1.6 system-ui, sans-serif;
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
        margin-bottom: 4px;
    }

    .foot {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex: 0 0 auto;
        height: 18px;
        padding: 0 10px 4px 12px;
        user-select: none;
        --wails-draggable: drag;
        mix-blend-mode: difference;
    }

    .state {
        color: #8d94a8;
        font: 11px/1 system-ui, sans-serif;
    }

    .gear {
        display: grid;
        place-items: center;
        width: 20px;
        height: 20px;
        padding: 0;
        border: 1px solid transparent;
        border-radius: 4px;
        background: transparent;
        color: #8d94a8;
        cursor: pointer;
        transition: background-color 0.15s,
        color 0.15s;
        --wails-draggable: no-drag;
    }

    .gear:hover {
        background: #fff;
        color: #4a5169;
    }

    .empty {
        margin-top: 40px;
        color: #a8aebe;
        font: 12px/1.5 system-ui, sans-serif;
        text-align: center;
    }
</style>
