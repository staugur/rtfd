'use strict'

/*
 * rtfd.js —— 文档页面右下角浮动挂件（无第三方依赖）
 *
 * 数据来源：GET {apiServer}/rtfd/{name}/desc（原生 fetch）
 * 参数来源：脚本标签的 data-* 属性（data-name/data-branch/data-api）
 * 交互：默认只显示当前语言与版本，点击后在右下角展开语言列表、版本列表与仓库链接，
 *       再次点击（或点击面板外、按 Esc）收起
 *
 * 多版本模式：文档路径形如 /{lang}/{branch}/...
 * 单版本模式：static-web-server 根目录即 {lang}/latest，路径中不含 lang/branch 前缀
 */
;(function () {
    const DEFAULT_ICON =
        'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAlUlEQVQ4T92S0Q0CMQxDnydBtwEbABvcRjAKK7DBscGNwCZGRbSKDigB/uhv4lc7svjxqeptj8AeWL9hTpJ2dScCLsAqY0hS00WA7+ITcJA0p2AhQgUMwBHYdAAtxoODYs92hb1k1BhdQMy6hKYAvRukANHB8lYpwB84+DTCVMrzdQ/ib7ZvsI6Ds6RtmbciZXr/bOcKjCNuESAd+XoAAAAASUVORK5CYII='

    const CSS = `
#rtfd-widget {
    position: fixed;
    right: 20px;
    bottom: 20px;
    z-index: 2147483000;
    color: #3e3e3e;
    font-size: 13px;
    line-height: 1.5;
    text-align: left;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Microsoft YaHei", sans-serif;
}
#rtfd-widget,
#rtfd-widget * {
    box-sizing: border-box;
}
#rtfd-widget .rtfd-trigger {
    display: flex;
    align-items: center;
    max-width: 240px;
    padding: 6px 10px;
    cursor: pointer;
    color: #fcfcfc;
    background-color: #2c3e50;
    border-radius: 3px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, .25);
    user-select: none;
    white-space: nowrap;
}
#rtfd-widget .rtfd-trigger:hover {
    background-color: #34495e;
}
#rtfd-widget .rtfd-icon {
    flex: 0 0 auto;
    width: 14px;
    height: 14px;
    margin-right: 6px;
    border: none;
}
#rtfd-widget .rtfd-current {
    min-width: 0;
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
}
#rtfd-widget .rtfd-caret {
    margin-left: 6px;
    font-size: 10px;
    opacity: .8;
    transition: transform .15s;
}
#rtfd-widget.rtfd-open .rtfd-caret {
    transform: rotate(180deg);
}
#rtfd-widget .rtfd-panel {
    display: none;
    position: absolute;
    right: 0;
    bottom: calc(100% + 8px);
    width: 240px;
    max-width: calc(100vw - 40px);
    max-height: 70vh;
    overflow-y: auto;
    padding: 6px 0;
    background-color: #fff;
    border: 1px solid #d9dde1;
    border-radius: 4px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, .18);
}
#rtfd-widget.rtfd-open .rtfd-panel {
    display: block;
}
#rtfd-widget .rtfd-title {
    padding: 6px 12px 2px;
    color: #8a9299;
    font-size: 11px;
    letter-spacing: .5px;
    text-transform: uppercase;
}
#rtfd-widget .rtfd-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin: 4px 12px 8px;
    padding: 0;
    list-style: none;
}
#rtfd-widget .rtfd-list li {
    display: inline-flex;
}
#rtfd-widget .rtfd-list a {
    display: inline-block;
    padding: 4px 10px;
    overflow: hidden;
    color: #2c3e50;
    text-decoration: none;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    background-color: #f0f4f7;
    border: 1px solid #d9dde1;
    border-radius: 4px;
}
#rtfd-widget .rtfd-list a:hover {
    background-color: #e2e9ef;
    border-color: #b9c2ca;
}
#rtfd-widget .rtfd-active > a {
    color: #fff;
    font-weight: 700;
    background-color: #2980b9;
    border-color: #2980b9;
}
#rtfd-widget .rtfd-footer {
    margin-top: 4px;
    padding: 8px 12px 2px;
    color: #9aa4ad;
    font-size: 11px;
    text-align: center;
    border-top: 1px solid #eceff1;
}
#rtfd-widget .rtfd-footer a {
    color: #2980b9;
    text-decoration: none;
}
@media (max-width: 480px) {
    #rtfd-widget {
        right: 10px;
        bottom: 10px;
    }
}
`

    // 取得本脚本节点：优先 document.currentScript（执行期最准确，IIFE 顶层已捕获），
    // 否则回退到扫描 src 含 rtfd.js 的脚本（兼容异步/延迟加载或旧浏览器）
    const SELF = document.currentScript || null
    function currentScript() {
        if (SELF && SELF.src) {
            return SELF
        }
        const found = document.querySelectorAll('script[src*="rtfd.js"]')
        return found.length ? found[found.length - 1] : null
    }

    // 统一解析配置：name（项目名）、branch（分支/版本）、apiServer（接口基址）
    // 取脚本标签的 data-* 属性（data-name/data-branch/data-api）。
    function getConfig() {
        const s = currentScript()
        const src = s ? (s.getAttribute('src') || '') : ''

        const pick = function (attr) {
            const v = s ? s.getAttribute('data-' + attr) : null
            return (v || '').trim()
        }

        let api = pick('api')
        if (!api && src) {
            // 由本脚本 URL 反推：脚本由 API 的 /rtfd/assets/rtfd.js 提供，
            // 去掉该后缀即得服务根地址，无需依赖 server_url 配置
            try {
                api = new URL(src, location.href).href
                    .replace(/\/rtfd\/assets\/rtfd\.js(\?.*)?$/i, '')
                    .replace(/\/+$/, '')
            } catch (e) {
                api = ''
            }
        }

        return {
            name: pick('name'),
            branch: pick('branch') || 'master',
            apiServer: (api || '').replace(/\/+$/, '')
        }
    }

    function injectCSS(code) {
        const style = document.createElement('style')
        style.appendChild(document.createTextNode(code))
        document.head.appendChild(style)
    }

    // 创建元素，文本一律经 textContent 写入，避免注入
    function el(tag, cls, text) {
        const node = document.createElement(tag)
        if (cls) {
            node.className = cls
        }
        if (text !== undefined && text !== null) {
            node.textContent = text
        }
        return node
    }

    // 解析当前页面所处的语言、版本与文档相对路径
    function currentState(single, langs) {
        const segs = location.pathname.split('/').filter(function (s) { return s !== '' })
        if (single) {
            return { lang: langs[0] || '', branch: 'latest', rest: segs.join('/') }
        }
        return {
            lang: segs[0] || langs[0] || '',
            branch: segs[1] || 'latest',
            rest: segs.slice(2).join('/')
        }
    }

    // 语言/版本切换链接：/{lang}/{ver}/{rest}
    function docURL(lang, ver, rest) {
        return '/' + lang + '/' + ver + '/' + (rest || '')
    }

    // 带标题的链接分组，items: [{text, href, active}]
    function group(title, items) {
        const box = el('div', 'rtfd-group')
        box.appendChild(el('div', 'rtfd-title', title))
        const ul = el('ul', 'rtfd-list')
        items.forEach(function (item) {
            const li = el('li', item.active ? 'rtfd-active' : '')
            const a = el('a', '', item.text)
            a.href = item.href
            if (item.active) {
                a.setAttribute('aria-current', 'true')
            }
            li.appendChild(a)
            ul.appendChild(li)
        })
        box.appendChild(ul)
        return box
    }

    // 仓库（查看/编辑源码）分组
    function repoGroup(data, cur) {
        if (data.hideGit === true || !data.url) {
            return null
        }
        const source = data.sourceDir || 'docs'
        const rst = cur.rest ? cur.rest.replace(/\.html$/, '.rst') : 'index.rst'
        const dftBranch = data.defaultBranch || ''
        // 解析当前页面对应的 git ref：latest 指向项目最新版本
        let ref = cur.branch
        let canEdit = cur.branch === dftBranch
        if (data.single === true) {
            ref = data.latest || dftBranch || 'latest'
            canEdit = true
        } else if (cur.branch === 'latest') {
            ref = data.latest || dftBranch || 'latest'
            canEdit = ref === dftBranch
        }
        const suffix = '/' + source + '/' + rst
        const items = [{ text: 'View', href: data.url + '/blob/' + ref + suffix }]
        if (canEdit) {
            items.push({ text: 'Edit', href: data.url + '/edit/' + ref + suffix })
        }
        return group('On ' + (data.gsp || 'Git'), items)
    }

    // 展开/收起面板：不传 open 时切换，传布尔值则显式设置
    function bind(widget, trigger) {
        function toggle(open) {
            const isOpen = open === undefined ? !widget.classList.contains('rtfd-open') : open
            widget.classList.toggle('rtfd-open', isOpen)
            trigger.setAttribute('aria-expanded', String(isOpen))
        }
        trigger.addEventListener('click', function (event) {
            event.stopPropagation()
            toggle()
        })
        trigger.addEventListener('keydown', function (event) {
            if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                toggle()
            }
        })
        document.addEventListener('click', function (event) {
            if (!widget.contains(event.target)) {
                toggle(false)
            }
        })
        document.addEventListener('keydown', function (event) {
            if (event.key === 'Escape') {
                toggle(false)
            }
        })
    }

    function mount(data) {
        injectCSS(CSS)

        const single = data.single === true
        const versions = data.versions || {}
        let langs = Array.isArray(data.lang) ? data.lang.filter(Boolean) : []
        if (!langs.length) {
            langs = Object.keys(versions)
        }
        const cur = currentState(single, langs)

        // 触发器：默认只显示当前语言与版本
        const trigger = el('div', 'rtfd-trigger')
        trigger.setAttribute('role', 'button')
        trigger.setAttribute('tabindex', '0')
        trigger.setAttribute('aria-expanded', 'false')
        trigger.setAttribute('aria-label', '文档语言与版本')
        const icon = el('img', 'rtfd-icon')
        icon.src = data.icon || DEFAULT_ICON
        icon.alt = ''
        trigger.appendChild(icon)
        trigger.appendChild(
            el('span', 'rtfd-current', [cur.lang, cur.branch].filter(Boolean).join(' · '))
        )
        trigger.appendChild(el('span', 'rtfd-caret', '▾'))

        const panel = el('div', 'rtfd-panel')
        if (!single) {
            // 仅有一种语言时无需展示语言列表
            if (langs.length > 1) {
                panel.appendChild(group('语言 Languages', langs.map(function (lang) {
                    return { text: lang, href: docURL(lang, 'latest', cur.rest), active: lang === cur.lang }
                })))
            }
            // 当前语言的可用版本，缺省补上当前版本
            const vers = Array.isArray(versions[cur.lang]) ? versions[cur.lang].slice(0) : []
            if (cur.branch && vers.indexOf(cur.branch) === -1) {
                vers.unshift(cur.branch)
            }
            if (vers.length > 1) {
                panel.appendChild(group('版本 Versions', vers.map(function (ver) {
                    return { text: ver, href: docURL(cur.lang, ver, cur.rest), active: ver === cur.branch }
                })))
            }
        }
        const repo = repoGroup(data, cur)
        if (repo) {
            panel.appendChild(repo)
        }

        const footer = el('div', 'rtfd-footer')
        footer.appendChild(document.createTextNode('Powered by '))
        const home = el('a', '', 'rtfd')
        home.href = 'https://github.com/staugur/rtfd'
        home.target = '_blank'
        home.rel = 'noopener noreferrer'
        footer.appendChild(home)
        panel.appendChild(footer)

        const widget = el('div', '')
        widget.id = 'rtfd-widget'
        widget.appendChild(trigger)
        widget.appendChild(panel)
        document.body.appendChild(widget)
        bind(widget, trigger)
    }

    function init() {
        const cfg = getConfig()
        // 缺少项目名或接口基址则静默退出，不渲染挂件
        if (!cfg.name || !cfg.apiServer) {
            return
        }
        fetch(cfg.apiServer + '/rtfd/' + encodeURIComponent(cfg.name) + '/desc', { credentials: 'omit' })
            .then(function (resp) { return resp.json() })
            .then(function (res) {
                // showNav 显式为 false 时不展示挂件
                if (!res || res.success !== true || !res.data || res.data.showNav === false) {
                    return
                }
                mount(res.data)
            })
            .catch(function () {
                // 元数据不可用时静默降级，不影响文档阅读
            })
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init)
    } else {
        init()
    }
})()
