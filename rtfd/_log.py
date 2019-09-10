# -*- coding: utf-8 -*-
"""
    _log
    ~~~~

    日志

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from os import mkdir
from os.path import join, exists
import logging
import logging.handlers
from .config import CfgHandler


class Logger:

    def __init__(self, logName, cfg=None, backupCount=10):
        self.cfg = CfgHandler(cfg)
        self.logName = logName
        self.log_dir = join(self.cfg.g.base_dir, 'logs')
        self.logFile = join(
            self.log_dir,
            '{0}.log'.format(self.logName)
        )
        self._levels = {
            "DEBUG": logging.DEBUG,
            "INFO": logging.INFO,
            "WARN": logging.WARNING,
            "WARNING": logging.WARNING,
            "ERROR": logging.ERROR,
            "CRITICAL": logging.CRITICAL
        }
        self._logfmt = '%Y-%m-%d %H:%M:%S'
        self._logger = logging.getLogger(self.logName)
        if not exists(self.log_dir):
            mkdir(self.log_dir)

        handler = logging.handlers.TimedRotatingFileHandler(
            filename=self.logFile,
            backupCount=backupCount,
            when="midnight"
        )
        handler.suffix = "%Y%m%d"
        formatter = logging.Formatter(
            '[%(levelname)s] %(asctime)s %(filename)s:%(lineno)d %(message)s',
            datefmt=self._logfmt
        )
        handler.setFormatter(formatter)
        if not self._logger.handlers:
            self._logger.addHandler(handler)
        LOGLEVEL = self.cfg.g.get("log_level", default="INFO").upper()
        self._logger.setLevel(self._levels.get(LOGLEVEL))

    @property
    def getLogger(self):
        return self._logger
