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
        self.cps = ProjectStorage(cfg)

    def create(self, name, url, **kwargs):
        name = name.lower()
        if name in self.cps.list:
            raise ProjectExistsError("This project '%s' already exists" % name)
        else:
            kwargs["url"] = url
            self.cps.set(name, kwargs)

    def has(self, name):
        name = name.lower()
        return name in self.cps.list

    def get(self, name, default=None):
        name = name.lower()
        if self.has(name):
            return self.cps.get(name, default=default)

    def update(self):
        pass

    def remove(self):
        pass


class RTFD_BUILDER(object):

    def __init__(self, cfg=None):
        self.cfg_file = cfg or expanduser("~/.rtfd.cfg")
        self.cpm = ProjectManager(self.cfg_file)
        self.build_sh = join(dirname(abspath(__file__)), "scripts/builder.sh")

    def build(self, name, branch="master"):
        name = name
        branch = branch
        if not self.cpm.has(name):
            raise ProjectNotFound("Did not find this project '%s'" % name)
        data = self.cpm.get(name)
        if data and isinstance(data, dict) and "url" in data:
            url = data["url"]
            cmd = ['bash', self.build_sh, '-n', name, '-u', url, '-b', branch,
                   '-c', self.cfg_file]
            '''
            code, out, err = run_cmd()
            print out,err
            if code == 0:
                return True
            '''
            for i in run_cmd_stream(*cmd):
                print i
        else:
            raise ValueError("Not found name, data error for %s" % name)
