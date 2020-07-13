#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
    cli
    ~~~

    命令行入口

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

import click
import json
from configparser import ConfigParser
from os import mkdir
from os.path import expanduser, isfile, isdir, isabs, abspath
from flask_pluginkit._compat import iteritems


DEFAULT_CFG = expanduser("~/.rtfd.cfg")


def echo(msg, fg=None):
    return click.echo(
        click.style(msg, fg=fg)
    )


def print_version(ctx, param, value):
    if not value or ctx.resilient_parsing:
        return
    from . import __version__
    click.echo(__version__)
    ctx.exit()


@click.group(context_settings={'help_option_names': ['-h', '--help']})
@click.option('--version', '-v', is_flag=True, callback=print_version,
              expose_value=False, is_eager=True)
def cli():
    pass


@cli.command()
@click.confirmation_option(prompt=u'确定要初始化rtfd吗？')
@click.option('--basedir', '-b', type=click.Path(), help=u'rtfd根目录')
@click.option('--loglevel', '-l', default='INFO', type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"]), help=u'日志级别', show_default=True)
@click.option('--server-url', '-su', default='', help=u'rtfd服务地址，默认是api段的http://host:port')
@click.option('--server-static_url', '-ssu', default='', help=u'rtfd静态资源地址，默认在server-url下')
@click.option('--favicon-url', '-fu', default='https://static.saintic.com/rtfd/favicon.png', help=u'文档HTML页面的默认图标地址', show_default=True)
@click.option('--unallowed-name', '-un', default='', help=u'不允许的文档项目名称，以英文逗号分隔', show_default=True)
@click.option('--nginx-dn', default='localhost.localdomain', help=u'文档生成后用以Nginx访问的顶级域名', show_default=True)
@click.option('--nginx-exec', type=click.Path(exists=True), default='/usr/sbin/nginx', help=u'Nginx管理命令路径', show_default=True)
@click.option('--nginx-ssl/--no-nginx-ssl', default=False, help=u'Nginx开启SSL', show_default=True)
@click.option('--nginx-ssl-crt', type=click.Path(), default='${g:base_dir}/certs/${dn}.crt', help=u'SSL证书', show_default=True)
@click.option('--nginx-ssl-key', type=click.Path(), default='${g:base_dir}/certs/${dn}.key', help=u'SSL证书私钥', show_default=True)
@click.option('--nginx-ssl-hsts-maxage', default=0, type=int, help=u'设置在浏览器收到这个请求后的maxage秒的时间内凡是访问这个域名下的请求都使用HTTPS请求。', show_default=True)
@click.option('--py2', type=click.Path(exists=True), default='/usr/bin/python2', help=u"Python2路径", show_default=True)
@click.option('--py3', type=click.Path(exists=True), default='/usr/bin/python3', help=u"Python3路径", show_default=True)
@click.option('--index', '-i',  type=str, default='https://pypi.org/simple', help=u'pip安装时的默认源', show_default=True)
@click.option('--host', default='127.0.0.1', help=u"Api监听地址", show_default=True)
@click.option('--port', default=5000, type=int, help=u"Api监听端口", show_default=True)
@click.option('--config', '-c', type=click.Path(exists=False), default=DEFAULT_CFG, help=u'rtfd的配置文件（不会覆盖）', show_default=True)
def init(basedir, loglevel, server_url, server_static_url, favicon_url, unallowed_name, nginx_dn, nginx_exec, nginx_ssl, nginx_ssl_crt, nginx_ssl_key, nginx_ssl_hsts_maxage, py2, py3, index, host, port, config):
    """初始化rtfd"""
    _cfg_file = config or DEFAULT_CFG
    if not isfile(_cfg_file):
        if not isfile(py2) or not isfile(py3):
            return echo("This py2 or py3 is error", fg='red')
        if not basedir:
            return echo("This basedir parameter is required", fg='red')
        if not isdir(basedir):
            if not isabs(basedir):
                basedir = abspath(basedir)
            mkdir(basedir)
        if nginx_ssl_hsts_maxage < 0:
            return echo("The nginx-ssl-hsts-maxage is error, it should be greater than 0.")
        nginx_ssl = "on" if nginx_ssl else "off"
        if not server_url:
            server_url = "http://${api:host}:${api:port}"
        if not server_static_url:
            server_static_url = ''
        else:
            if server_static_url[-1] != "/":
                server_static_url += "/"
        #: write default configure
        _cfg_obj = ConfigParser()
        _cfg_obj.add_section("g")
        _cfg_obj.add_section("nginx")
        _cfg_obj.add_section("py")
        _cfg_obj.add_section("api")
        _cfg_obj.set("g", "base_dir", basedir.rstrip('/'))
        _cfg_obj.set("g", "log_level", loglevel)
        _cfg_obj.set("g", "server_url", server_url)
        _cfg_obj.set("g", "server_static_url", server_static_url)
        _cfg_obj.set("g", "favicon_url", favicon_url)
        _cfg_obj.set("g", "unallowed_name", unallowed_name)
        _cfg_obj.set("nginx", "dn", nginx_dn)
        _cfg_obj.set("nginx", "exec", nginx_exec)
        _cfg_obj.set("nginx", "ssl", nginx_ssl)
        _cfg_obj.set("nginx", "ssl_crt", nginx_ssl_crt)
        _cfg_obj.set("nginx", "ssl_key", nginx_ssl_key)
        _cfg_obj.set("nginx", "ssl_hsts_maxage", str(nginx_ssl_hsts_maxage))
        _cfg_obj.set("py", "py2", py2)
        _cfg_obj.set("py", "py3", py3)
        _cfg_obj.set("py", "index", index)
        _cfg_obj.set("api", "host", host)
        _cfg_obj.set("api", "port", str(port))
        with open(_cfg_file, 'w') as fp:
            _cfg_obj.write(fp)
    else:
        return echo("Found configuration file %s" % _cfg_file, fg='green')


@cli.command()
@click.option('--action', '-a', default='get', type=click.Choice(["create", "update", "remove", "get", "list"]), help=u'动作', show_default=True)
@click.option('--url', type=str, help=u'文档项目的git仓库地址，如果是私有仓库，请在url协议后携带编码后的username:password')
@click.option('--latest', default='master', type=str, help=u'latest所指向的分支', show_default=True)
@click.option('--single/--no-single', default=False, help=u'是否开启单一版本功能', show_default=True)
@click.option('--sourcedir', '-s',  type=str, default='docs', help=u'实际文档文件所在目录，目录路径是项目的相对位置', show_default=True)
@click.option('--languages', '-l',  type=str, default='en', help=u'文档语言，支持多种，以英文逗号分隔', show_default=True)
@click.option('--default-language', '-dl', type=str, default='en', help=u'文档默认展示的语言，若默认语言不在languages内，则重置为languages中第一语言', show_default=True)
@click.option('--version', '-v',  type=int, default=2, help=u'Python版本，目前仅支持2、3两个值，对应版本由配置文件定义', show_default=True)
@click.option('--requirements', '-r',  type=str, default='', help=u'需要安装的依赖包文件（文件路径是项目的相对位置），支持多个，以英文逗号分隔')
@click.option('--install/--no-install', default=False, help=u'是否需要安装项目，如果值为true，则会在项目目录执行"pip install ."', show_default=True)
@click.option('--index', '-i',  type=str, default='', help=u'指定pip安装时的pypi源，默认是rtfd配置的源（其默认为官方源）', show_default=True)
@click.option('--show-nav/--no-show-nav', default=True, help=u'是否显示导航', show_default=True)
@click.option('--webhook-secret', '-ws', default='', help=u"Webhook密钥")
@click.option('--custom-domain', '-cd', default='', help=u'文档项目开启自定义域名功能', show_default=True)
@click.option('--ssl/--no-ssl', default=False, help=u'文档项目自定义域名是否开启SSL', show_default=True)
@click.option('--ssl-crt', type=click.Path(exists=True), help=u'自定义域名的SSL证书', show_default=True)
@click.option('--ssl-key', type=click.Path(exists=True), help=u'自定义域名的SSL证书私钥', show_default=True)
@click.option('--ssl-hsts-maxage', default=0, type=int, help=u'设置在浏览器收到这个请求后的maxage秒的时间内凡是访问这个域名下的请求都使用HTTPS请求。', show_default=True)
@click.option('--builder', '-b', type=click.Choice(["html", "dirhtml", "singlehtml"]), default='html', help=u"Sphinx构建器", show_default=True)
@click.option('--update-rule', '-ur', help=u'当action为update时会解析此项，要求是JSON格式，指定要更新的配置内容！')
@click.option('--config', '-c', type=click.Path(exists=True), default=DEFAULT_CFG, help=u'rtfd的配置文件', show_default=True)
@click.argument('name')
def project(action, url, latest, single, sourcedir, languages, default_language, version, requirements, install, index, show_nav, webhook_secret, custom_domain, ssl, ssl_crt, ssl_key, ssl_hsts_maxage, builder, update_rule, config, name):
    """文档项目管理"""
    from .libs import ProjectManager
    from .config import CfgHandler
    from .utils import is_domain, check_giturl, get_git_service_provider
    name = name.lower()
    pm = ProjectManager(config)
    if action == 'get':
        name, key = name.split(":") if ":" in name else (name, None)
        if not pm.has(name):
            return echo("Not found docs project named %s" % name, fg='red')
        data = pm.get(name)
        if key:
            try:
                value = data[key]
            except KeyError:
                echo("")
            else:
                if value is True:
                    value = 'true'
                if value is False:
                    value = 'false'
                if isinstance(value, int):
                    value = str(value)
                echo(value)
        else:
            echo(json.dumps(data, sort_keys=True))
    elif action == 'create':
        c_res = check_giturl(url)
        if not c_res["status"]:
            return echo(c_res["msg"], fg='red')
        url = url[:-4] if url.endswith(".git") else url
        if default_language not in languages.split(","):
            default_language = languages.split(",")[0]
        if not index:
            _cfg = CfgHandler(config)
            index = _cfg.py.get("index", default="https://pypi.org/simple")
        if custom_domain and not is_domain(custom_domain):
            return echo("You have set custom_domain, but the format is "
                        "incorrect, custom_domain does not take effect.",
                        fg="red")
        if pm.has_custom_domain(custom_domain):
            return echo("The domain name is already occupied", fg='red')
        pm.create(
            name, url, latest=latest, single=single, sourcedir=sourcedir,
            languages=languages, default_language=default_language,
            version=version, requirements=requirements, install=install,
            index=index, show_nav=show_nav, webhook_secret=webhook_secret,
            custom_domain=custom_domain if is_domain(custom_domain) else False,
            ssl=ssl, ssl_crt=ssl_crt, ssl_key=ssl_key, builder=builder,
            ssl_hsts_maxage=ssl_hsts_maxage,
            _type=c_res["_type"], _gsp=get_git_service_provider(url),
        )
        #: generate nginx template
        pm.nginx_builder(name)
    elif action == 'update':
        if update_rule[-4:] in (".cfg", ".ini") and isfile(update_rule):
            data = pm.get(name)
            #: 依照文档的rtfd配置文件内容更新项目信息
            urc = CfgHandler(update_rule)
            update_rule = {}
            try:
                project = urc.project
            except AttributeError:
                pass
            else:
                latest = project.get("latest")
                if latest and latest != data.get("latest"):
                    update_rule['latest'] = latest
            try:
                sphinx = urc.sphinx
            except AttributeError:
                pass
            else:
                sourcedir = sphinx.get("sourcedir")
                languages = sphinx.get("languages")
                builder = sphinx.get("builder")
                if sourcedir and sourcedir != data.get("sourcedir"):
                    update_rule["sourcedir"] = sourcedir
                if languages and languages != data.get("languages"):
                    update_rule["languages"] = languages
                if builder and builder != data.get("builder"):
                    update_rule["builder"] = builder
            try:
                py = urc.python
            except AttributeError:
                pass
            else:
                version = py.get("version")
                requirements = py.get("requirements")
                install = py.get("install")
                index = py.get("index")
                if version and version in (2, 3) and version != data.get("version"):
                    update_rule["version"] = version
                if requirements and requirements != data.get("requirements"):
                    update_rule["requirements"] = requirements
                if install and install in (True, "true", "True", False, "False", "false") and install != data.get("install"):
                    update_rule["install"] = install
                if index and index != data.get("index"):
                    update_rule["index"] = index
        else:
            update_rule = json.loads(update_rule)
            if "latest" in update_rule:
                return echo("Unallow update latest with cli", fg="red")
        #: 更新内容检测
        if not isinstance(update_rule, dict):
            return echo("the update rule is error", fg='red')
        #: 检测键值，需要二次更新的存入单独的字典中，迭代完成后更新回update_rule
        _will_update = {}
        for key in update_rule.keys():
            if key.startswith("_") or "-" in key:
                return echo("Found keys that are not allowed to be updated", fg='red')
            if key == "url":
                url = update_rule["url"]
                c_res = check_giturl(url)
                if not c_res["status"]:
                    return echo(c_res["msg"], fg='red')
                url = url[:-4] if url.endswith(".git") else url
                _will_update["url"] = url
                _will_update["_type"] = c_res["_type"]
            if key == "custom_domain":
                custom_domain = update_rule["custom_domain"]
                if custom_domain not in ("false", False, "False"):
                    if not is_domain(custom_domain):
                        return echo("Invalid custom_domain", fg="red")
                    if pm.has_custom_domain(custom_domain):
                        return echo("The domain name is already occupied", fg='red')
        update_rule.update(_will_update)
        pm.update(name, **update_rule)
    elif action == 'remove':
        pm.remove(name)
    elif action == 'list':
        data = pm._cps.list
        param = name
        if param != "raw":
            data = {k: v for k, v in iteritems(data) if pm.has(k)}
        if param == "only":
            data = data.keys()
        echo(json.dumps(data, sort_keys=True))
    else:
        return echo("Invalid action", fg='red')


@cli.command()
@click.option('--branch', '-b', default='master', help=u'文档构建所在的git分支', show_default=True)
@click.option('--config', '-c', type=click.Path(exists=True), default=DEFAULT_CFG, help=u'rtfd的配置文件', show_default=True)
@click.argument('name')
def build(config, branch, name):
    """构建文档"""
    if not isfile(config):
        return echo("Not Found configuration file %s" % config, fg='red')
    from .libs import RTFD_BUILDER
    rb = RTFD_BUILDER(config)
    for _out in rb.build(name, branch, "cli"):
        print(_out)


@cli.command()
@click.option('--host', help=u"Api监听地址", show_default=True)
@click.option('--port', type=int, help=u"Api监听端口", show_default=True)
@click.option('--debug/--no-debug', default=True, help=u'是否开启DEBUG', show_default=True)
@click.option('--config', '-c', type=click.Path(exists=True), default=DEFAULT_CFG, help=u'rtfd的配置文件', show_default=True)
def api(host, port, debug, config):
    """以开发模式运行API"""
    from .app import app
    from .config import CfgHandler
    cfg = CfgHandler(config)
    app.run(
        host=host or cfg.api.get("host", default="127.0.0.1"),
        port=port or cfg.api.get("port", default=5000),
        debug=debug
    )


@cli.command()
@click.option('--config', '-c', type=click.Path(exists=True), default=DEFAULT_CFG, help=u'rtfd的配置文件', show_default=True)
@click.argument('section_item')
def cfg(config, section_item):
    """查询配置文件的配置内容"""
    from .config import CfgHandler
    _cfg = CfgHandler(config)
    if ":" in section_item:
        section, item = section_item.split(":")
        if section in _cfg.sections:
            print(_cfg[section].get(item, default=''))
        else:
            print('')
    else:
        for k, v in _cfg.items(section_item):
            print(k, v)


if __name__ == "__main__":
    cli()
