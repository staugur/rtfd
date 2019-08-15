const rtfd = {
    //api url
    api: 'http://127.0.0.1:5000/api/docs/',
    //static ur;
    static: 'https://static.saintic.com/rtfd/',
    //Determines whether the id exists on the page. This id returns true, otherwise it returns false.
    hasId: function (id) {
        var element = document.getElementById(id);
        if (element) {
            return true
        } else {
            return false
        }
    },
    //load css
    addCSS: function (href) {
        var link = document.createElement('link');
        link.type = 'text/css';
        link.rel = 'stylesheet';
        link.href = this.static + href;
        document.getElementsByTagName("head")[0].appendChild(link);
    },
    //load js
    addJS: function (src, cb) {
        var script = document.createElement("script");
        script.type = "text/javascript";
        script.src = this.static + src;
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
        //$.ajax({});
        var html = '<div class=rtfd><div id=rtfd-header><img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAlUlEQVQ4T92S0Q0CMQxDnydBtwEbABvcRjAKK7DBscGNwCZGRbSKDigB/uhv4lc7svjxqeptj8AeWL9hTpJ2dScCLsAqY0hS00WA7+ITcJA0p2AhQgUMwBHYdAAtxoODYs92hb1k1BhdQMy6hKYAvRukANHB8lYpwB84+DTCVMrzdQ/ib7ZvsI6Ds6RtmbciZXr/bOcKjCNuESAd+XoAAAAASUVORK5CYII="><scan>&nbsp;v: latest&nbsp;</scan></div><div id=rtfd-body><dl><dt>Languages<dd class=active><a href="https://flask-pluginkit.readthedocs.io/en/latest/">en</a><dd><a href="https://flask-pluginkit.readthedocs.io/zh_CN/latest/">zh_CN</a></dl><dl><dt>Versions<dd class=active><a href="https://flask-pluginkit.readthedocs.io/en/latest/">latest</a></dl><dl><dt>On GitHub<dd><a href=https://github.com/staugur/Flask-PluginKit/blob/master/docs/index.rst>View</a><dd><a href=https://github.com/staugur/Flask-PluginKit/edit/master/docs/index.rst>Edit</a></dl><hr><small class=footer><span>Powered by <a href=https://github.com/staugur/rtfd>rtfd</a></span></small></div></div>';
        that.addCSS('tipped.css');
        that.addJS('tipped.js', function () {
            that.addHtml(html);
            Tipped.create('#rtfd-header', {
                title: 'Hello, v: latest',
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
$(function () {
    console.log("I will reload rtfd.");
    rtfd.init();
});