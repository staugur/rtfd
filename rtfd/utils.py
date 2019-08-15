# -*- coding: utf-8 -*-
"""
    utils
    ~~~~~

    工具

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

import shelve
from os.path import join
from flask_pluginkit import LocalStorage
from .config import cfg
from ._log import Logger

logger = Logger("sys").getLogger


class ProjectStorage(LocalStorage):

    def __init__(self):
        self.index = join(cfg.g.base_dir, '.rtfd-projects.dat')

    def _open(self, flag="c"):
        return shelve.open(
            filename=self.index,
            flag=flag,
            protocol=2
        )
