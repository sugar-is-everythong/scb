import {Fetch} from "#lib/fetch.js";
import {Window} from "@wailsio/runtime";
import {toRealPath} from "#lib/file.js";

// 写入剪贴板
export const cbInset = (url, body, onsuccess, onerror) => {
    Fetch(`/cb/${url}`, {method: 'POST', body})
        .then(() => {
            Window.Hide();
            setTimeout(() => {
                Fetch('/cb/inset');
                onsuccess?.()
            }, 1);
        })
        .catch((e) => {
            console.error('写入剪贴板失败', e)
            onerror()
        });
}

/**
 * 保存到 store
 * @param {import('#bindings/scb/cb/models.ts').RecordJs} record
 * */
export const saveLocal = (record) => {
    // 为啥要这样做呢？因为go侧需要的图片类型是二进制，而js为了方便渲染使用的是 <img src={record.image}> ，那类型肯定是不对的啊，所以直接传空，go侧使用hasFile进行处理
    if (typeof record.image === 'string') {
        record.image = null
        record.files[0] = toRealPath(record.files[0])
    }
    return Fetch('/cb/savelocal', {method: 'POST', body: JSON.stringify(record)})
}

// 从 store 删除指定 index
export const deleteLocal = (index) => {
    return Fetch('/cb/deletelocal', {method: 'POST', body: JSON.stringify({index})})
}

/**
 * 全量覆盖 store。传入的 records 顺序与 All() 一致（最新在前），内部反转后发给后端
 * @param {import('#bindings/scb/cb/models.ts').RecordJs[]} records
 */
export const updateStoreAll = (records) => {
    return Fetch('/cb/setalllocal', {method: 'POST', body: JSON.stringify([...records].reverse())})
}

export const withKind = (cb) => {
    if (cb === undefined) return {}
    if (cb.hasHTML) return {kind: 'html', label: 'HTML'};
    if (cb.hasImage) return {kind: 'image', label: '图片'};
    if (cb.hasFiles) return {kind: 'files', label: '文件'};
    if (cb.hasRTF) return {kind: 'rtf', label: 'RTF'};
    return {kind: 'text', label: '文本'};
}