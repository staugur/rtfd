'use strict';
const rtfd = {
    //Get script self data
    getUrlQuery: function (key, acq) {
        /*
            Get the query parameters after the URL, without the anchor part,
            such as url is http://x.com/user/message/?status=1&Action=getCount.
            If there is no query key, the entire query parameter object is
            returned, which returns {status: "1", Action: "getCount"};
            If there is a query key, the object value is returned.
            The return value can specify the default value acq:
                such as key=status, return 1; key=non_exsits_key returns acq
        */
        var str = document.getElementById('rtfd-script').getAttribute('data');
        var obj = {};
        if (str) {
            //str = str.substring(1, str.length);
            var arr = str.split("&");
            for (var i = 0; i < arr.length; i++) {
                var tmp_arr = arr[i].split("=");
                obj[decodeURIComponent(tmp_arr[0])] = decodeURIComponent(tmp_arr[1]);
            }
        }
        return key ? obj[key] || acq : obj;
    },
    //Determines whether the id exists on the page. This id returns true, otherwise it returns false.
    hasId: function (id) {
        var element = document.getElementById(id);
        if (element) {
            return true
        } else {
            return false
        }
    },
    //api url
    api: function () {
        var api = this.getUrlQuery("rtfd_api");
        var anf = this.getUrlQuery("api_no_fill", "no");
        return anf === "yes" ? api : api + "/rtfd/api";
    },
    //static url
    static: function () {
        return this.getUrlQuery("rtfd_static");
    },
    //load css
    addCSS: function (href) {
        var link = document.createElement('link');
        link.type = 'text/css';
        link.rel = 'stylesheet';
        link.href = this.static() + href;
        document.getElementsByTagName("head")[0].appendChild(link);
    },
    //load js
    addJS: function (src, cb) {
        var script = document.createElement("script");
        script.type = "text/javascript";
        script.src = this.static() + src;
        document.getElementsByTagName('head')[0].appendChild(script);
        script.onload = typeof cb === "function" ? cb : function () {};
    },
    //Add a html code to the body
    addHtml: function (html) {
        document.body.innerHTML = html + document.body.innerHTML;
    },
    //Initiate an ajax request to get the initialization code of the document
    init: function () {
        var that = this;
        var name = location.host.split('.')[0];
        $.ajax({
            url: that.api() + '?Action=describeProject&name=' + name,
            type: "GET",
            dataType: "json",
            success: function (res) {
                //console.log(res);
                if (res.code === 0 && res.data.showNav != false) {
                    if (res.data.single === false) {
                        var lang = location.pathname.split('/')[1];
                        var branch = location.pathname.split('/')[2];
                        var other_path = location.pathname.split('/').slice(3).join('/');
                        var path_rst = other_path ? other_path.replace(".html", ".rst") : 'index.rst';
                        //console.log(name, lang, branch, 'other is:' + other_path, 'rst is:' + path_rst);
                        var langs_str = res.data.languages.map(function (_lang) {
                            var active = _lang === lang ? "active" : "";
                            return `<dd class="${active}"><a href="/${_lang}/latest/${other_path}">${_lang}</a></dd>`;
                        }).join("");
                        var vers_str = res.data.versions[lang].map(function (_ver) {
                            var active = _ver === branch ? "active" : "";
                            return `<dd class="${active}"><a href="/${lang}/${_ver}/${other_path}">${_ver}</a></dd>`;
                        }).join("");
                        var github_str = '';
                        if (branch === "master" || (branch === "latest" && res.data.latest === "master")) {
                            github_str += `<dd><a href=${res.data.url}/blob/master/${res.data.sourcedir}/${path_rst}>View</a></dd>`;
                            github_str += `<dd><a href=${res.data.url}/edit/master/${res.data.sourcedir}/${path_rst}>Edit</a></dd>`;
                        } else {
                            //for tag
                            github_str += `<dd><a href=${res.data.url}/blob/${branch}/${res.data.sourcedir}/${path_rst}>View</a></dd>`;
                        }
                        var base_str = `<div class=rtfd><div id=rtfd-header><img src="${res.data.icon}"><scan>&nbsp;v: ${branch}&nbsp;</scan></div><div id=rtfd-body><dl><dt>Languages</dt>${langs_str}</dl><dl><dt>Versions</dt>${vers_str}</dl><dl><dt>On GitHub</dt>${github_str}</dl><hr><small class=footer><span>Powered by <a href=https://github.com/staugur/rtfd>rtfd</a></span></small></div></div>`;
                    } else {
                        var branch = "latest";
                        var other_path = location.pathname.split('/').slice(1).join('/');
                        var path_rst = other_path ? other_path.replace(".html", ".rst") : 'index.rst';
                        var github_str = '';
                        github_str += `<dd><a href=${res.data.url}/blob/master/${res.data.sourcedir}/${path_rst}>View</a></dd>`;
                        github_str += `<dd><a href=${res.data.url}/edit/master/${res.data.sourcedir}/${path_rst}>Edit</a></dd>`;
                        var base_str = `<div class=rtfd><div id=rtfd-header><img src="${res.data.icon}"><scan>&nbsp;v: ${branch}&nbsp;</scan></div><div id=rtfd-body><dl><dt>On GitHub</dt>${github_str}</dl><hr><small class=footer><span>Powered by <a href=https://github.com/staugur/rtfd>rtfd</a></span></small></div></div>`;
                    }
                    that.addCSS('tipped.css');
                    that.addJS('tipped.js', function () {
                        that.addHtml(base_str);
                        Tipped.create('#rtfd-header', {
                            title: `Version: ${branch}`,
                            inline: 'rtfd-body',
                            showOn: 'click',
                            hideOn: 'click',
                            close: 'overlap',
                            position: 'left',
                            maxWidth: 300
                        });
                    });
                }
            }
        });
    }
}
$(function () {
    rtfd.init();
});