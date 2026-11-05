/**
 * GO语言中的 filepath
 * @param {string[]} strs 文件路径
 */
export const filepath = (strs = []) => strs.join('/')

export const fileType = [
    "application/vnd.ms-package.3dmanufacturing-3dmodel+xml",
    "application/x-7z-compressed",
    "application/x-archive",
    "application/octet-stream",
    "application/x-msaccess",
    "application/x-amf",
    "application/atom+xml",
    "application/x-bzip2",
    "application/vnd.ms-cab-compressed",
    "application/x-installshield",
    "application/x-java-applet",
    "application/x-chrome-extension",
    "application/x-cpio",
    "model/vnd.collada+xml",
    "application/x-dbf",
    "application/dicom",
    "application/vnd.debian.binary-package",
    "application/msword",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/x-rpm",
    "application/vnd.ms-fontobject",
    "application/epub+zip",
    "application/vnd.fdf",
    "application/fits",
    "application/geo+json",
    "model/gltf-binary",
    "application/gml+xml",
    "application/gpx+xml",
    "application/gzip",
    "application/jar",
    "application/javascript",
    "application/vnd.google-earth.kml+xml",
    "application/x-ms-reader",
    "application/x-executable",
    "application/lzip",
    "application/vnd.apple.mpegurl",
    "application/x-ms-shortcut",
    "application/x-mach-binary",
    "application/x-msaccess",
    "application/x-mobipocket-ebook",
    "application/marc",
    "application/x-ms-installer",
    "application/vnd.ms-outlook",
    "application/x-ndjson",
    "application/x-ndjson",
    "application/vnd.nintendo.snes.rom",
    "application/x-object",
    "application/vnd.oasis.opendocument.formula",
    "application/vnd.sun.xml.calc",
    "application/vnd.oasis.opendocument.graphics",
    "application/vnd.oasis.opendocument.presentation",
    "application/vnd.oasis.opendocument.spreadsheet",
    "application/vnd.oasis.opendocument.text",
    "font/otf",
    "application/vnd.oasis.opendocument.graphics-template",
    "application/vnd.oasis.opendocument.presentation-template",
    "application/vnd.oasis.opendocument.spreadsheet-template",
    "application/vnd.oasis.opendocument.text-template",
    "application/vnd.oasis.opendocument.chart",
    "application/owl+xml",
    "application/pdf",
    "application/vnd.ms-powerpoint",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    "application/postscript",
    "application/pkcs7-signature",
    "application/pkcs7-signature",
    "application/vnd.ms-publisher",
    "application/x-rar-compressed",
    "application/vnd.rn-realmedia-vbr",
    "application/x-rpm",
    "application/rss+xml",
    "application/x-mach-binary",
    "application/vnd.shp",
    "application/vnd.shx",
    "application/x-sharedlib",
    "application/vnd.sqlite3",
    "application/x-subrip",
    "application/x-shockwave-flash",
    "application/x-tar",
    "application/vnd.garmin.tcx+xml",
    "application/x-bittorrent",
    "font/collection",
    "font/ttf",
    "application/tzif",
    "application/octet-stream",
    "application/warc",
    "application/wasm",
    "font/woff",
    "font/woff2",
    "model/x3d+xml",
    "application/x-xar",
    "application/vnd.adobe.xfdf",
    "application/x-xliff+xml",
    "application/vnd.ms-excel",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "application/x-xz",
    "application/zstd",
]

export const jsonType = [
    "application/json",
]

export const imgType = [
    "image/vnd.mozilla.apng",
    "image/avif",
    "image/bmp",
    "image/bpg",
    "image/vnd.djvu",
    "image/vnd.dwg",
    "image/vnd.dwg",
    "image/x-gimp-gbr",
    "image/gif",
    "image/vnd.radiance",
    "image/heic",
    "image/heif",
    "image/x-icon",
    "image/jp2",
    "image/jpx",
    "image/jpeg",
    "image/jpm",
    "image/jxl",
    "image/jxr",
    "image/x-xpixmap",
    "image/svg+xml",
    "image/svg+xml",
    "image/tiff",
    "image/webp",
    "image/x-xcf",
    "image/x-gimp-pat",
    "image/png",
]
export const psdType = [
    "image/vnd.adobe.photoshop",
]

export const htmlType = [
    "text/html; charset=iso-8859-1",
    "text/html; charset=us-ascii",
    "text/html; charset=utf-8",
    "text/xml; charset=utf-8",
    "text/xml; charset=utf-8",
]

export const textType = [
    "text/calendar",
    "text/calendar",
    "text/plain; charset=iso-8859-1",
    "text/plain; charset=utf-8",
    "text/plain; charset=utf-8",
    "text/plain; charset=utf-8",
    "text/csv",
    "text/x-lua",
    "text/plain; charset=utf-16be",
    "text/plain; charset=utf-16le",
    "text/plain; charset=utf-32be",
    "text/plain; charset=utf-32le",
    "text/plain; charset=utf-8",
    "text/x-tcl",
    "text/tab-separated-values",
    "text/vcard",
    "text/vcard",
    "text/vtt",
    "text/x-php",
    "text/x-perl",
    "text/x-python",
    "text/rtf",
]

export const audioType = [
    "audio/mpeg",
    "audio/mpeg",
    "audio/mpeg",
    "audio/mpeg",
    "audio/x-m4a",
    "audio/mp4",
    "audio/flac",
    "audio/amr",
    "audio/ape",
    "audio/basic",
    "audio/aiff",
    "audio/aac",
    "audio/midi",
    "audio/musepack",
    "audio/ogg",
    "audio/ogg",
    "audio/x-unknown",
    "audio/qcelp",
    "audio/wav",
]

export const videoType = [
    "video/x-matroska",
    "video/quicktime",
    "video/mp4",
    "video/webm",
    "video/mpeg",
    "video/quicktime",
    "video/ogg",
    "video/3gpp2",
    "video/3gpp",
    "video/x-ms-asf",
    "video/x-msvideo",
    "video/x-flv",
]

export const zipType = [
    "application/zip",
    "application/x-rar-compressed",
]

export const exeType = [
    "application/vnd.microsoft.portable-executable",
]

export const gpType = [
    "application/x-ole-storage",
]

// 返回指定类型的文件图标
export const getFileIcon = (file, pathRouets) => {
    let fileType = file.type
    if (fileType === undefined) return ""
    else if (fileType === "") return "/public/win-icon/wfjmjw.png"
    else if (textType.includes(fileType)) return "/public/win-icon/textIcon.png"
    else if (htmlType.includes(fileType)) return "/public/win-icon/web-icon.png"
    else if (jsonType.includes(fileType)) return "/public/win-icon/json-icon.png"
    else if (audioType.includes(fileType)) return "/public/win-icon/audioIcon.png"
    else if (videoType.includes(fileType)) return "/public/win-icon/videoIcon.png"
    else if (zipType.includes(fileType)) return "/public/win-icon/zipIcon.png"
    else if (gpType.includes(fileType)) return "/public/win-icon/msi.png"
    else if (exeType.includes(fileType)) return "/public/win-icon/exeIcon.png"
    else if (imgType.includes(fileType)) {
        pathRouets = pathRouets.replace(":\\", "/")
        // console.log(URL + '/file/' + pathRouets + '/' + file.name)
        return URL + 'file/' + pathRouets + '/' + file.name // 这里会多出一个 “/” 变成“//” 但是不影响，可以修复
    } else return "/public/win-icon/unkownIcon.png"
}

// 把图片的本地路径替换为服务器路径
export const localPathReplace = (path) => {
    if (path.includes("file:///")) path = path.replaceAll("file:///", '/wg/file/').replaceAll(':\\', '/').replace(':/', '/')
    // 解码 URL 编码的字符(如 %20 -> 空格)
    try {
        path = decodeURIComponent(path)
    } catch (e) {
        // 解码失败保持原样
    }
    return path
}

// 将服务器路径替换为图片的本地路径
export const toRealPath = (path) => {
    path = path.replace('/wg/file/', '')
    return path[0] + ":" + path.slice(1)
}
