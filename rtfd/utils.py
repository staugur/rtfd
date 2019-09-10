# -*- coding: utf-8 -*-
"""
    utils
    ~~~~~

    工具

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

import shelve
from re import compile
from os.path import join
from time import strftime
from urlparse import urlparse
from flask_pluginkit import LocalStorage
from subprocess import Popen, PIPE, STDOUT
from .config import CfgHandler


class ProjectStorage(LocalStorage):

    def __init__(self, cfg=None):
        self.cfg = CfgHandler(cfg)
        self.index = join(self.cfg.g.base_dir, '.rtfd-projects.dat')

    def _open(self, flag="c"):
        return shelve.open(
            filename=self.index,
            flag=flag,
            protocol=2
        )


def run_cmd(*args):
    """
    Execute the external command and get its exitcode, stdout and stderr.
    """
    try:
        proc = Popen(args, stdout=PIPE, stderr=STDOUT)
    except (OSError, Exception) as e:
        out, err, exitcode = (str(e), None, 1)
    else:
        out, err = proc.communicate()
        exitcode = proc.returncode
    finally:
        return exitcode, out, err


def run_cmd_stream(*args):
    proc = Popen(args, stdout=PIPE, stderr=STDOUT)
    for i in iter(proc.stdout.readline, b''):
        yield i.rstrip()


def is_true(value):
    if value and value in (True, "True", "true", "on", 1, "1", "yes"):
        return True
    return False


def is_domain(value):
    if value in ("false", False, "False", "off"):
        return False
    if value and isinstance(value, basestring):
        dn_pat = compile(
            r'^(?=^.{3,255}$)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+$'
        )
        ip_pat = compile(
            r"^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$"
        )
        if dn_pat.match(value) and not ip_pat.match(value):
            return True
    return False


def get_now():
    return strftime('%Y-%m-%d %H:%M:%S')


def check_giturl(url):
    res = dict(status=False, msg=None)
    if url and isinstance(url, basestring) and (url.startswith("http://") or
                                                url.startswith("https://")):
        rst = urlparse(url)
        if rst.hostname in ("github.com", "gitee.com"):
            if rst.username:
                if not rst.password:
                    res.update(
                        msg="The warehouse has set up users but no password"
                    )
                else:
                    res.update(status=True, _type="private")
            else:
                res.update(status=True, _type="public")
        else:
            res.update(msg="Unsupported git service provider")
    else:
        res.update(msg="Invalid url")
    return res


def get_public_giturl(url):
    c_res = check_giturl(url)
    if c_res["status"]:
        if c_res["_type"] == "public":
            return url
        else:
            rst = urlparse(url)
            return rst._replace(netloc=rst.netloc.split("@")[-1]).geturl()
