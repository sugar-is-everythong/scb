<script>
    import {getContext} from "svelte";

    /** @type {{ html?: string }} */
    let {html = ""} = $props();
    let inDetail = getContext("inDetail");

    /**
     * 判断元素是否为空：自身没有文字，且所有子元素也为空
     * @param {Element} el
     * @returns {boolean}
     */
    const isEmpty = (el) =>
        !el.textContent.trim() && Array.from(el.children).every(isEmpty);

    /**
     * 判断 html 是否只包含图片（允许夹杂换行和空节点）
     * @param {string} html
     * @returns {boolean}
     */
    const isOnlyImg = (html) => {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, "text/html");
        const children = Array.from(doc.body.children);
        // && !doc.body.textContent.trim(); // 我觉得body里只包含图片节点、换行和文字内容都算是只包含图片，毕竟对我的显示好像没啥影响
        return (
            children.some((el) => el.tagName === "IMG") &&
            children.every((el) => el.tagName === "IMG" || isEmpty(el))
        );
    };

    /**
     * 按每张图片自身的横竖比例约束尺寸：纵向最大高度 200px，横向最大宽度 100%
     * @param {ShadowRoot} root
     */
    const fitImgByRatio = (root) => {
        /** @param {HTMLImageElement} img */
        const apply = (img) => {
            if (!img.naturalWidth || !img.naturalHeight) return;
            const portrait = img.naturalHeight > img.naturalWidth;
            if (inDetail) {
                img.style.width = "auto";
                img.style.height = "auto";
                img.style.maxHeight = portrait ? "50vh" : "40vh";
            } else {
                img.style.maxWidth = "100%";
                img.style.maxHeight = "200px";
                img.style.width = "auto";
                img.style.height = "auto";
            }
        };
        /** @type {HTMLImageElement[]} */
        const imgs = Array.from(root.querySelectorAll("img"));
        imgs.forEach((img) => {
            if (img.complete) {
                apply(img);
            } else {
                img.addEventListener("load", () => apply(img), {once: true});
            }
        });
    };

    /**
     * 把 HTML 片段渲染进影子根：样式、选择器、脚本都与主界面隔离
     * @type {import('svelte/attachments').Attachment}
     */
    const shadow = (node) => {
        const root = node.shadowRoot ?? node.attachShadow({mode: "open"});

        // 到底要不要限制在 isOnlyImg 范围下呢，不显示太大了，限制了可能又会影响显示
        root.innerHTML = `${html}<style>img { max-width: 100%;${isOnlyImg(html) ? ' position: relative !important;' : ''}} </style>`;
        fitImgByRatio(root);
        /*  if (isOnlyImg(html)) {
              // 图片加载前先按横向约束兜底，加载完成后再按真实比例修正
              root.innerHTML = `${html}<style>img { max-width: 100%;}</style>`;
              fitImgByRatio(root);
          } else {
              root.innerHTML = html;
          }*/
    };
</script>

<div class="html" style:--height={inDetail ? "100%" : "150px"}>
<!--    <div style="width:100%;" {@attach shadow}></div>-->
    <div style="width:fit-content;min-width: 100%" {@attach shadow}></div>
</div>

<style>
    .html {
        contain: layout paint;
        max-height: var(--height);
        width: 100%;
        overflow: auto;
    }
</style>
