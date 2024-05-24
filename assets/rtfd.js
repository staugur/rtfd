'use strict'

var _rtfd_script =
    document.getElementsByTagName('script')[
    document.getElementsByTagName('script').length - 1
    ]

    ; (function () {
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
                'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAlUlEQVQ4T92S0Q0CMQxDnydBtwEbABvcRjAKK7DBscGNwCZGRbSKDigB/uhv4lc7svjxqeptj8AeWL9hTpJ2dScCLsAqY0hS00WA7+ITcJA0p2AhQgUMwBHYdAAtxoODYs92hb1k1BhdQMy6hKYAvRukANHB8lYpwB84+DTCVMrzdQ/ib7ZvsI6Ds6RtmbciZXr/bOcKjCNuESAd+XoAAAAASUVORK5CYII=';

        String.prototype.substr = function (start, length) {
            const S = this.toString(); // step 1 and 2
            const size = S.length; // step 3
            let intStart = Number.isNaN(Number(start)) ? 0 : Number.parseInt(start); // step 4
            if (intStart === -Infinity) intStart = 0; // step 5
            else if (intStart < 0) intStart = Math.max(size + intStart, 0); // step 6
            else intStart = Math.min(intStart, size); // step 7
            let intLength = length === undefined ? size : (Number.isNaN(Number(length)) ? 0 : Number.parseInt(length)); // step 8
            intLength = Math.max(Math.min(intLength, size), 0); // step 9
            let intEnd = Math.min(intStart + intLength, size); // step 10
            return S.substring(intStart, intEnd); // step 11
        };

        //Determines whether the id exists on the page. This id returns true, otherwise it returns false.
        function hasId(id) {
            if (document.getElementById(id)) {
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
            let str = null;
            if (hasId('rtfd-script') === true) {
                str = document
                    .getElementById('rtfd-script')
                    .getAttribute('data')
            } else {
                if (
                    _rtfd_script &&
                    _rtfd_script.getAttribute('src') &&
                    _rtfd_script.getAttribute('src').indexOf('rtfd.js') > -1
                ) {
                    let src = _rtfd_script.getAttribute('src')
                    str = src.indexOf('?') > -1 ? src.substr(src.indexOf('?') + 1) : '';
                }
            }
            let obj = {};
            if (str) {
                let arr = str.split('&')
                for (let i = 0; i < arr.length; i++) {
                    let tmp_arr = arr[i].split('=')
                    obj[decodeURIComponent(tmp_arr[0])] = decodeURIComponent(tmp_arr[1])
                }
            }
            return key ? obj[key] || acq : obj
        }
        //load css
        function addCSS(href) {
            let link = document.createElement('link')
            link.type = 'text/css'
            link.rel = 'stylesheet'
            link.href = href
            document.getElementsByTagName('head')[0].appendChild(link)
        }
        //load js
        function addJS(src, cb) {
            let script = document.createElement('script')
            script.type = 'text/javascript'
            script.src = src
            document.getElementsByTagName('head')[0].appendChild(script)
            script.onload = typeof cb === 'function' ? cb : function () { }
        }
        function loadCSSCode(code) {
            let style = document.createElement('style')
            style.rel = 'stylesheet'
            try {
                //for Chrome Firefox Opera Safari
                style.appendChild(document.createTextNode(code))
            } catch (ex) {
                //for IE
                style.styleSheet.cssText = code
            }
            let head = document.getElementsByTagName('head')[0]
            head.appendChild(style)
        }

        //Initiate an ajax request to get the initialization code of the document
        function init() {
            let name = getUrlQuery('name')
            loadCSSCode(_rtfd_style)
            $.ajax({
                url: getUrlQuery('rtfd_api') + '/rtfd/' + name + '/desc',
                type: 'GET',
                dataType: 'json',
                success: function (res) {
                    if (res.success === true && res.data.showNav != false) {
                        let dftBranch = res.data.defaultBranch,
                            base_str = '',
                            branch = ''
                        if (res.data.single === false) {
                            let lang = location.pathname.split('/')[1]
                            branch = location.pathname.split('/')[2]
                            let other_path = location.pathname
                                .split('/')
                                .slice(3)
                                .join('/')
                            let path_rst = other_path
                                ? other_path.replace('.html', '.rst')
                                : 'index.rst'

                            // console.debug(name, lang, branch, 'other is:' + other_path, 'rst is:' + path_rst)
                            let langs_str = res.data.lang
                                .map(function (_lang) {
                                    let active = _lang === lang ? 'active' : ''
                                    return `<dd class="${active}"><a href="/${_lang}/latest/${other_path}">${_lang}</a></dd>`
                                })
                                .join('')
                            let vers_str = res.data.versions[lang]
                                .map(function (_ver) {
                                    let active = _ver === branch ? 'active' : ''
                                    return `<dd class="${active}"><a href="/${lang}/${_ver}/${other_path}">${_ver}</a></dd>`
                                })
                                .join('')
                            let github_str = ''
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
                            base_str = `<div id="rtfd" class="rtfd"><div id="rtfd-header"><img src="${icon_baseuri}"><scan>&nbsp;v: ${branch}&nbsp;</scan></div><div id="rtfd-body"><dl><dt>Languages</dt>${langs_str}</dl><dl><dt>Versions</dt>${vers_str}</dl><dl><dt>On ${res.data.gsp}</dt>${github_str}</dl><hr><small class="footer"><span>Powered by <a href="https://github.com/staugur/rtfd">rtfd</a></span></small></div></div>`
                        } else {
                            branch = 'latest'
                            let other_path = location.pathname
                                .split('/')
                                .slice(1)
                                .join('/')
                            let path_rst = other_path
                                ? other_path.replace('.html', '.rst')
                                : 'index.rst'
                            let github_str = ''
                            github_str += `<dd><a href=${res.data.url}/blob/${dftBranch}/${res.data.sourceDir}/${path_rst}>View</a></dd>`
                            github_str += `<dd><a href=${res.data.url}/edit/${dftBranch}/${res.data.sourceDir}/${path_rst}>Edit</a></dd>`
                            if (res.data.hideGit === true) {
                                github_str = ''
                            }
                            base_str = `<div id="rtfd" class="rtfd"><div id="rtfd-header"><img src="${icon_baseuri}"><scan>&nbsp;v: ${branch}&nbsp;</scan></div><div id="rtfd-body"><dl><dt>On ${res.data.gsp}</dt>${github_str}</dl><hr><small class="footer"><span>Powered by <a href="https://github.com/staugur/rtfd">rtfd</a></span></small></div></div>`
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

        addJS("https://cdn.jsdelivr.net/npm/jquery@3.7.0/dist/jquery.min.js", init)
    })()
