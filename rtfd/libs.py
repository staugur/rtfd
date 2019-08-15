# -*- coding: utf-8 -*-
"""
    libs
    ~~~~

    核心库

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from os.path import expanduser
from .utils import ProjectStorage
from .exceptions import ProjectExistsError, ProjectNotFound


class ProjectManager(object):

    def __init__(self):
        self.cps = ProjectStorage()

    def create(self, name, url, **kwargs):
        name = name.lower()
        if name in self.cps.list:
            raise ProjectExistsError("This project '%s' already exists" % name)
        else:
            kwargs["url"] = url
            self.cps.set(name, kwargs)

    def update(self):
        pass

    def remove(self):
        pass


class RTFD_BUILDER(object):

    def __init__(self, name, branch="master", config=None):
        self.name = name
        self.branch = branch
        self.cps = ProjectStorage()
        if self.name not in self.cps.list:
            raise ProjectNotFound("Did not find this project '%s'" % self.name)
        #: 用户级rtfd配置文件
        self.cfg_file = config or expanduser("~/.rtfd.cfg")

    def build(self):
        pass
