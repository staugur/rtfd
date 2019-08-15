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


@click.group()
def cli():
    pass


@cli.command()
@click.confirmation_option(prompt=u'确定要初始化rtfd吗？')
@click.option('--basedir', '-b', help=u'rtfd根目录')
@click.option('--loglevel', '-l', default='INFO', type=click.Choice(["DEBUG", "INFO", "WARNING", "ERROR"]), help=u'日志级别')
@click.option('--py2', default='/usr/bin/python2', help=u"Python2路径")
@click.option('--py3', default='/usr/bin/python3', help=u"Python3路径")
@click.option('--host', default='127.0.0.1', help=u"Api监听地址")
@click.option('--port', default=5000, type=int, help=u"Api监听端口")
def init(basedir, loglevel, py2, py3, host, port):
    """初始化rtfd"""
    _cfg_file = expanduser("~/.rtfd.cfg")
    if not isfile(_cfg_file):
        if not isfile(py2) or nor isfile(py3):
            return click.echo(
                click.style("This py2 or py3 is error", fg='red')
            )
        if not basedir:
            return click.echo(
                click.style("This basedir parameter is required", fg='red')
            )
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
        click.echo(
            click.style("Found configuration file %s" % _cfg_file, fg='green')
        )


if __name__ == "__main__":
    cli()
