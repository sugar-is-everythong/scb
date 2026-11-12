import {Events} from '@wailsio/runtime';
import {Fetch} from '#lib/fetch.js';

/**
 * 与后端 set.Config 一一对应。
 * 默认值取自 main.go 里“粘一下”窗口的现有参数：300×418、不精简、显示富文本、
 * 100 条、super+v、窗口底色 rgb(173 197 255 / 0.3)、卡片底色 rgb(255 255 255 / 0.6)。
 */
export const config = $state({
    // 顶部过滤条：四个类型开关 + 搜索框 + 清空按钮
    filterBar: {
        text: true,
        html: true,
        image: true,
        files: true,
        search: true,
        clear: true
    },
    // 卡片右侧工具条
    toolRail: {
        pin: true,
        open: true,
        edit: true,
        delete: true
    },
    opacity: 0.3,
    // 窗口背景：color 用纯色，#adc5ff 是主窗口原本的底色；image 用上传的图片，两者互斥
    bgMode: 'color',
    bgColor: '#adc5ff',
    bgImage: '',
    cardOpacity: 0.6,
    // 强调色，用户选什么就是什么；正文色与提示色由它按同色相派生，一级比一级浅
    textColor: '#000000',
    width: 300,
    height: 418,
    maxRecords: 100,
    shortcut: 'super+v',
    richText: true,
    windowMode: 0
});

/** 读取 local/config.json，失败时保持上面的前端默认值 */
export const loadConfig = async () => {
    try {
        const cfg = await Fetch('/set/get');
        if (cfg) Object.assign(config, cfg);
    } catch (e) {
        console.error('读取设置失败', e);
    }
};

/** 把 #rrggbb 与透明度拼成 rgba()；非法颜色回落到主窗口原底色 */
const hexToRgba = (hex, alpha) => {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex ?? '');
    const n = m ? parseInt(m[1], 16) : 0xadc5ff;
    return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
};

/**
 * 三个窗口共用的背景样式。
 * 纯色直接给容器；图片由 window-bg 图层负责自己处理opacity，这里只把容器底色清空。
 */
export const backgroundStyle = (cfg) => {
    if (cfg.bgMode === 'image' && cfg.bgImage) {
        return 'background-color: transparent;';
    }
    return `background-color: ${hexToRgba(cfg.bgColor, cfg.opacity)};`;
};

/** #rrggbb 转 [色相 0-360, 饱和度 0-1, 亮度 0-1]，非法颜色当黑色 */
const hexToHsl = (hex) => {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex ?? '');
    const n = m ? parseInt(m[1], 16) : 0;
    const r = ((n >> 16) & 255) / 255;
    const g = ((n >> 8) & 255) / 255;
    const b = (n & 255) / 255;
    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    const l = (max + min) / 2;
    if (max === min) return [0, 0, l];
    const d = max - min;
    const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    let h;
    if (max === r) h = (g - b) / d + (g < b ? 6 : 0);
    else if (max === g) h = (b - r) / d + 2;
    else h = (r - g) / d + 4;
    return [h * 60, s, l];
};

/** [色相, 饱和度, 亮度] 转 #rrggbb */
const hslToHex = (h, s, l) => {
    const f = (n) => {
        const k = (n + h / 30) % 12;
        const a = s * Math.min(l, 1 - l);
        const c = l - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
        return Math.round(c * 255).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`;
};

/**
 * 强调色原样用用户选的颜色，正文色与提示色在它的色相与亮度上小幅递进，一级比一级浅又不脱色系。
 */
export const textStyle = (cfg) => {
    const [h, s, l] = hexToHsl(cfg.textColor);
    return `--text-strong: ${cfg.textColor}; --text-main: ${hslToHex(h, s * 0.6, Math.min(0.85, l + 0.18))}; --text-muted: ${hslToHex(h, s * 0.4, Math.min(0.9, l + 0.36))};`;
};

/** 后端返回的配置应用到共享状态并广播，所有窗口一起换 */
export const applyConfig = (cfg) => {
    if (cfg) Object.assign(config, cfg);
    Events.Emit('config-change', $state.snapshot(config));
};

/** 保存当前配置，并把结果广播给其它窗口（背景、透明度都是跨窗口生效的） */
export const saveConfig = async () => {
    const cfg = await Fetch('/set/save', {
        method: 'POST',
        body: JSON.stringify($state.snapshot(config))
    });
    applyConfig(cfg);
    return cfg;
};

/** 监听其它窗口的保存结果，返回取消函数供 onMount 清理 */
export const syncConfig = () =>
    Events.On('config-change', (event) => {
        if (event.data) Object.assign(config, event.data);
    });
