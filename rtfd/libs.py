# -*- coding: utf-8 -*-
"""
    libs
    ~~~~

    核心库

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from os.path import expanduser, dirname, join, abspath
from .utils import ProjectStorage, run_cmd, run_cmd_stream
from .exceptions import ProjectExistsError, ProjectNotFound


class ProjectManager(object):

    def __init__(self, cfg=None):
        self._cps = ProjectStorage(cfg)

    def create(self, name, url, **kwargs):
        name = name.lower()
        if self.has(name):
            raise ProjectExistsError("This project '%s' already exists" % name)
        else:
            kwargs["url"] = url
            return self._cps.set(name, kwargs)

    def has(self, name):
        name = name.lower()
        if self._cps.get(name):
            return True
        else:
            return False

    def get(self, name, default=None):
        name = name.lower()
        if self.has(name):
            return self._cps.get(name, default=default)
        return default

    def update(self, name, **kwargs):
        name = name.lower()
        if self.has(name):
            data = self.get(name)
            if isinstance(data, dict):
                data.update(kwargs)
                return self._cps.set(name, data)

    def remove(self, name):
        name = name.lower()
        if self.has(name):
            return self._cps.set(name, '')


class RTFD_BUILDER(object):

    def __init__(self, cfg=None):
        self._cfg_file = cfg or expanduser("~/.rtfd.cfg")
        self._cpm = ProjectManager(self._cfg_file)
        self._build_sh = join(dirname(abspath(__file__)), "scripts/builder.sh")

    def build(self, name, branch="master", stream=True):
        name = name
        branch = branch
        if not self._cpm.has(name):
            raise ProjectNotFound("Did not find this project '%s'" % name)
        data = self._cpm.get(name)
        if data and isinstance(data, dict) and "url" in data:
            url = data["url"]
            cmd = ['bash', self._build_sh, '-n', name, '-u', url, '-b', branch,
                   '-c', self._cfg_file]
            if stream is True:
                for i in run_cmd_stream(*cmd):
                    print(i)
            else:
                code, out, err = run_cmd(*cmd)
                if code == 0:
                    return True
        else:
            raise ValueError("Not found name, data error for %s" % name)
