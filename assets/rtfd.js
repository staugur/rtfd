'use strict'

var _rtfd_script =
    document.getElementsByTagName('script')[
        document.getElementsByTagName('script').length - 1
    ]

;(function () {
    const _rtfd_style = `
/* start rtfd */
.rtfd {
    position: fixed;
    display: block;
    border: none;
    right: 20px;
    bottom: 50px;
    z-index: 9999999;
    color: #fff;
    max-width: 300px;
    height: auto;
}

.rtfd #rtfd-header {
    cursor: pointer;
    width: 100px;
    color: #fcfcfc;
    background-color: #1f1d1d;
    overflow: hidden;
    text-align: center;
    padding-top: 6px;
    padding-bottom: 6px;
    border-radius: 1px;
}

.rtfd #rtfd-header scan {
    text-align: center;
    color: #27AE60;
    font-size: 90%;
}

.rtfd #rtfd-header img {
    width: 14px;
    height: 14px;
    border: none;
}

#rtfd-body {
    min-width: 200px;
    display: none;
    text-align: left;
    font-size: 90%;
    padding: 5px;
    color: gray;
}

#rtfd-body a {
    text-decoration: none;
}

#rtfd-body dd a {
    display: inline-block;
    padding: 5px;
    color: #fcfcfc;
}

#rtfd-body dl {
    margin: 0;
}

#rtfd-body dl dd {
    display: inline-block;
    margin: 0;
}

#rtfd-body hr {
    border: 1px solid #999;
    display: block;
    height: 1px;
    border: 0;
    margin: 20px 0;
    padding: 0;
    border-top: solid 1px #413d3d;
}

#rtfd-body .footer {
    text-align: center;
}

#rtfd-body .active {
    font-weight: bold;
}

.tpd-tooltip .tpd-title {
    text-transform: none;
}
/* end rtfd*/
`,
        icon_baseuri =
            'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAlUlEQVQ4T92S0Q0CMQxDnydBtwEbABvcRjAKK7DBscGNwCZGRbSKDigB/uhv4lc7svjxqeptj8AeWL9hTpJ2dScCLsAqY0hS00WA7+ITcJA0p2AhQgUMwBHYdAAtxoODYs92hb1k1BhdQMy6hKYAvRukANHB8lYpwB84+DTCVMrzdQ/ib7ZvsI6Ds6RtmbciZXr/bOcKjCNuESAd+XoAAAAASUVORK5CYII='
    //Determines whether the id exists on the page. This id returns true, otherwise it returns false.
    function hasId(id) {
        var element = document.getElementById(id)
        if (element) {
            return true
        } else {
            return false
        }
    }
    //Get script self data
    function getUrlQuery(key, acq) {
        /*
            Get the parameters from the url query or data.
            If there is a query key, the object value is returned.
            The return value can specify the default value acq:
                such as key=status, return 1; key=non_exsits_key returns acq
            */
        if (hasId('rtfd-script') === true) {
            var str = document
                .getElementById('rtfd-script')
                .getAttribute('data')
        } else {
            if (
                _rtfd_script &&
                _rtfd_script.getAttribute('src') &&
                _rtfd_script.getAttribute('src').indexOf('rtfd.js') > -1
            ) {
                var src = _rtfd_script.getAttribute('src')
                var str =
                    src.indexOf('?') > -1
                        ? src.substr(src.indexOf('?') + 1)
                        : ''
            }
        }
        var obj = {}
        if (str) {
            var arr = str.split('&')
            for (var i = 0; i < arr.length; i++) {
                var tmp_arr = arr[i].split('=')
                obj[decodeURIComponent(tmp_arr[0])] = decodeURIComponent(
                    tmp_arr[1]
                )
            }
        }
        return key ? obj[key] || acq : obj
    }
    //load css
    function addCSS(href) {
        var link = document.createElement('link')
        link.type = 'text/css'
        link.rel = 'stylesheet'
        link.href = href
        document.getElementsByTagName('head')[0].appendChild(link)
    }
    //load js
    function addJS(src, cb) {
        var script = document.createElement('script')
        script.type = 'text/javascript'
        script.src = src
        document.getElementsByTagName('head')[0].appendChild(script)
        script.onload = typeof cb === 'function' ? cb : function () {}
    }
    function loadCSSCode(code) {
        var style = document.createElement('style')
        style.rel = 'stylesheet'
        try {
            //for Chrome Firefox Opera Safari
            style.appendChild(document.createTextNode(code))
        } catch (ex) {
            //for IE
            style.styleSheet.cssText = code
        }
        var head = document.getElementsByTagName('head')[0]
        head.appendChild(style)
    }
    //Initiate an ajax request to get the initialization code of the document
    function init() {
        var name = getUrlQuery('name')
        loadCSSCode(_rtfd_style)
        $.ajax({
            url: getUrlQuery('rtfd_api') + '/rtfd/' + name + '/desc',
            type: 'GET',
            dataType: 'json',
            success: function (res) {
                if (res.success === true && res.data.showNav != false) {
                    var dftBranch = res.data.defaultBranch
                    if (res.data.single === false) {
                        var lang = location.pathname.split('/')[1]
                        var branch = location.pathname.split('/')[2]
                        var other_path = location.pathname
                            .split('/')
                            .slice(3)
                            .join('/')
                        var path_rst = other_path
                            ? other_path.replace('.html', '.rst')
                            : 'index.rst'

                        // console.debug(name, lang, branch, 'other is:' + other_path, 'rst is:' + path_rst)
                        var langs_str = res.data.lang
                            .map(function (_lang) {
                                var active = _lang === lang ? 'active' : ''
                                return `<dd class="${active}"><a href="/${_lang}/latest/${other_path}">${_lang}</a></dd>`
                            })
                            .join('')
                        var vers_str = res.data.versions[lang]
                            .map(function (_ver) {
                                var active = _ver === branch ? 'active' : ''
                                return `<dd class="${active}"><a href="/${lang}/${_ver}/${other_path}">${_ver}</a></dd>`
                            })
                            .join('')
                        var github_str = ''
                        if (
                            branch === dftBranch ||
                            (branch === 'latest' &&
                                res.data.latest === dftBranch)
                        ) {
                            github_str += `<dd><a href=${res.data.url}/blob/${dftBranch}/${res.data.sourceDir}/${path_rst}>View</a></dd>`
                            github_str += `<dd><a href=${res.data.url}/edit/${dftBranch}/${res.data.sourceDir}/${path_rst}>Edit</a></dd>`
                        } else {
                            //for tag
                            github_str += `<dd><a href=${res.data.url}/blob/${branch}/${res.data.sourceDir}/${path_rst}>View</a></dd>`
                        }
                        if (res.data.hideGit === true) {
                            github_str = ''
                        }
                        var base_str = `<div id="rtfd" class="rtfd"><div id="rtfd-header"><img src="${icon_baseuri}"><scan>&nbsp;v: ${branch}&nbsp;</scan></div><div id="rtfd-body"><dl><dt>Languages</dt>${langs_str}</dl><dl><dt>Versions</dt>${vers_str}</dl><dl><dt>On ${res.data.gsp}</dt>${github_str}</dl><hr><small class="footer"><span>Powered by <a href="https://github.com/staugur/rtfd">rtfd</a></span></small></div></div>`
                    } else {
                        var branch = 'latest'
                        var other_path = location.pathname
                            .split('/')
                            .slice(1)
                            .join('/')
                        var path_rst = other_path
                            ? other_path.replace('.html', '.rst')
                            : 'index.rst'
                        var github_str = ''
                        github_str += `<dd><a href=${res.data.url}/blob/${dftBranch}/${res.data.sourceDir}/${path_rst}>View</a></dd>`
                        github_str += `<dd><a href=${res.data.url}/edit/${dftBranch}/${res.data.sourceDir}/${path_rst}>Edit</a></dd>`
                        if (res.data.hideGit === true) {
                            github_str = ''
                        }
                        var base_str = `<div id="rtfd" class="rtfd"><div id="rtfd-header"><img src="${icon_baseuri}"><scan>&nbsp;v: ${branch}&nbsp;</scan></div><div id="rtfd-body"><dl><dt>On ${res.data.gsp}</dt>${github_str}</dl><hr><small class="footer"><span>Powered by <a href="https://github.com/staugur/rtfd">rtfd</a></span></small></div></div>`
                    }
                    addCSS(
                        'https://cdn.jsdelivr.net/gh/staaky/tipped/dist/css/tipped.css'
                    )
                    addJS(
                        'https://cdn.jsdelivr.net/gh/staaky/tipped/dist/js/tipped.min.js',
                        function () {
                            $('body').append(base_str)
                            Tipped.create('#rtfd-header', {
                                title:
                                    `Version: ${branch}` +
                                    (branch === 'latest'
                                        ? ' -> ' + res.data.latest
                                        : ''),
                                inline: 'rtfd-body',
                                showOn: 'click',
                                hideOn: 'click',
                                close: 'overlap',
                                position: 'left',
                                maxWidth: 250
                            })
                            $(window).scroll(function () {
                                Tipped.hide('#rtfd-header')
                            })
                        }
                    )
                }
            }
        })
    }

    init()
})()
