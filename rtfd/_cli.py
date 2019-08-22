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
from os.path import expanduser, isfile, isdir

DEFAULT_CFG = expanduser("~/.rtfd.cfg")


def echo(msg, fg=None):
    return click.echo(
        click.style(msg, fg=fg)
    )


@click.group()
def cli():
    pass


@cli.command()
@click.confirmation_option(prompt=u'确定要初始化rtfd吗？')
@click.option('--basedir', '-b', help=u'rtfd根目录')
@click.option('--loglevel', '-l', default='INFO', type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"]), help=u'日志级别', show_default=True)
@click.option('--nginxdn', default='localhost.localdomain', help=u'文档生成后用以Nginx访问的顶级域名', show_default=True)
@click.option('--nginxexec', default='/usr/sbin/nginx', help=u'Nginx管理命令路径', show_default=True)
@click.option('--py2', default='/usr/bin/python2', help=u"Python2路径", show_default=True)
@click.option('--py3', default='/usr/bin/python3', help=u"Python3路径", show_default=True)
@click.option('--host', default='127.0.0.1', help=u"Api监听地址", show_default=True)
@click.option('--port', default=5000, type=int, help=u"Api监听端口", show_default=True)
@click.option('--config', '-c', default=DEFAULT_CFG, help=u'rtfd的配置文件（不会覆盖）', show_default=True)
def init(basedir, loglevel, nginxdn, nginxexec, py2, py3, host, port, config):
    """初始化rtfd"""
    _cfg_file = config or DEFAULT_CFG
    if not isfile(_cfg_file):
        if not isfile(py2) or not isfile(py3):
            return echo("This py2 or py3 is error", fg='red')
        if not basedir:
            return echo("This basedir parameter is required", fg='red')
        if not isdir(basedir):
            mkdir(basedir)
        #: write default configure
        _cfg_obj = ConfigParser()
        _cfg_obj.add_section("g")
        _cfg_obj.add_section("py")
        _cfg_obj.add_section("api")
        _cfg_obj.set("g", "base_dir", basedir)
        _cfg_obj.set("g", "log_level", loglevel)
        _cfg_obj.set("g", "nginx_dn", nginxdn)
        _cfg_obj.set("g", "nginx_exec", nginxexec)
        _cfg_obj.set("py", "py2", py2)
        _cfg_obj.set("py", "py3", py3)
        _cfg_obj.set("api", "host", host)
        _cfg_obj.set("api", "port", str(port))
        with open(_cfg_file, 'wb') as fp:
            _cfg_obj.write(fp)
    else:
        return echo("Found configuration file %s" % _cfg_file, fg='green')


@cli.command()
@click.option('--action', '-a', default='get', type=click.Choice(["create", "update", "remove", "get"]), help=u'动作', show_default=True)
@click.option('--url', type=str, help=u'文档项目的git仓库地址')
@click.option('--latest', default='master', type=str, help=u'latest所指向的分支', show_default=True)
@click.option('--single/--no-single', default=False, help=u'是否开启单一版本功能', show_default=True)
@click.option('--sourcedir', '-s',  type=str, default='docs', help=u'实际文档文件所在目录，目录路径是项目的相对位置', show_default=True)
@click.option('--languages', '-l',  type=str, default='en', help=u'文档语言，支持多种，以英文逗号分隔', show_default=True)
@click.option('--version', '-v',  type=int, default=2, help=u'Python版本，目前仅支持2、3两个值，对应版本由配置文件定义', show_default=True)
@click.option('--requirements', '-r',  type=str, default='', help=u'需要安装的依赖包文件（文件路径是项目的相对位置），支持多个，以英文逗号分隔')
@click.option('--install/--no-install', default=False, help=u'是否需要安装项目，如果值为true，则会在项目目录执行"pip install ."', show_default=True)
@click.option('--index', '-i',  type=str, default='https://pypi.org/simple', help=u'指定pip安装时的pypi源', show_default=True)
@click.option('--update-rule', help=u'当action为update时会解析此项，要求是JSON格式，指定要更新的配置内容！')
@click.option('--config', '-c', default=DEFAULT_CFG, help=u'rtfd的配置文件', show_default=True)
@click.argument('name')
def project(action, url, latest, single, sourcedir, languages, version, requirements, install, index, update_rule, config, name):
    """文档项目管理"""
    from .libs import ProjectManager
    name = name.lower().encode('utf-8')
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
            echo(json.dumps(data))
    elif action == 'create':
        if not url:
            return echo("url is required", fg='red')
        pm.create(name, url, latest=latest, single=single, sourcedir=sourcedir, languages=languages,
                  version=version, requirements=requirements, install=install, index=index)
        #: generate nginx template
        pm.nginx_builder(name)
    elif action == 'update':
        update_rule = json.loads(update_rule)
        pm.update(name, **update_rule)
        #: update nginx template
        if "languages" in update_rule or "single" in update_rule:
            pm.nginx_builder(name)
    elif action == 'remove':
        pm.remove(name)
    else:
        return echo("Invalid action", fg='red')


@cli.command()
@click.option('--config', '-c', default=DEFAULT_CFG, help=u'rtfd的配置文件', show_default=True)
@click.option('--branch', '-b', default='master', help=u'文档构建所在的git分支', show_default=True)
@click.argument('name')
def build(config, branch, name):
    """构建文档"""
    if not isfile(config):
        return echo("Not Found configuration file %s" % config, fg='red')
    from .libs import RTFD_BUILDER
    rb = RTFD_BUILDER(config)
    rb.build(name, branch)


if __name__ == "__main__":
    cli()
