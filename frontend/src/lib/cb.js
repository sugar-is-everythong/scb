import {Fetch} from "#lib/fetch.js";
import {Window} from "@wailsio/runtime";

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
