import {browser} from '$app/environment';
import {loadConfig} from '#lib/config.svelte.js';

// 加载初始化配置
export async function load() {
    if (browser) await loadConfig();
    return {};
}
