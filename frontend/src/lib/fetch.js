/**
 * 一个封装的 fetch 和 window上的原生 fetch 基本一致
 * @param url {string} 请求地址
 * @param opt 请求配置
 * @param opt.methods {string} 请求方法
 * @param opt.body {RequestInit | null | undefined } 请求方法
 */
export const Fetch = (url, opt = {}) => {
    if (!url.startsWith('http')) url = '/wg' + url
    return fetch(url, opt)
        .then((res) => {
            if (!res.ok) {                       // HTTP 状态码不在 200-299
                const err = new Error(`HTTP ${res.status}`)
                err.status = res.status          // 网络层错误：带 status
                throw err
            }
            return res.json()
        })
        .then((res) => {
            if (res.code !== 200) console.error(res.code)
            return res.data
        })
}