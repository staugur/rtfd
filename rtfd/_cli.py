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
@click.option('--py2', default='/usr/bin/python2', help=u"Python2路径", show_default=True)
@click.option('--py3', default='/usr/bin/python3', help=u"Python3路径", show_default=True)
@click.option('--host', default='127.0.0.1', help=u"Api监听地址", show_default=True)
@click.option('--port', default=5000, type=int, help=u"Api监听端口", show_default=True)
@click.option('--config', '-c', default=DEFAULT_CFG, help=u'rtfd的配置文件（不会覆盖）', show_default=True)
def init(basedir, loglevel, py2, py3, host, port, config):
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
        _cfg_obj.set("py", "py2", py2)
        _cfg_obj.set("py", "py3", py3)
        _cfg_obj.set("api", "host", host)
        _cfg_obj.set("api", "port", str(port))
        with open(_cfg_file, 'wb') as fp:
            _cfg_obj.write(fp)
    else:
        return echo("Found configuration file %s" % _cfg_file, fg='green')


@cli.command()
def project():
    """文档项目管理"""
    pass


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
