# -*- coding: utf-8 -*-
"""
    config
    ~~~~~~

    配置

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from os.path import isfile, expanduser
from configparser import ConfigParser, ExtendedInterpolation

__all__ = ["CfgHandler"]


class SectionHandler(object):

    def __init__(self, cfg, section):
        self._cfg_file = cfg._cfg_file
        self._cfg_obj = cfg._cfg_obj
        self._section = section

    def __str__(self):
        return "<%s object at %s, the section is %s(in %s)>" % (
            self.__class__.__name__, hex(id(self)),
            self._section, self._cfg_file
        )

    __repr__ = __str__

    def __getattr__(self, option):
        option = option.lower()
        if self._cfg_obj.has_option(self._section, option):
            value = self._cfg_obj.get(self._section, option)
            if value in ("true", "True"):
                value = True
            elif value in ("false", "False"):
                value = False
            return value
        raise AttributeError(
            "No option %s in section: %s" % (option, self._section)
        )

    __getitem__ = __getattr__

    def get(self, option, default=None, converter=None, err_ignore=True):
        try:
            value = getattr(self, option)
        except AttributeError:
            if err_ignore is True:
                return default
            else:
                raise
        else:
            if callable(converter):
                value = converter(value)
            return value

    def __len__(self):
        return len(self._cfg_obj.options(self._section))


class CfgHandler(object):

    def __init__(self, cfg=None):
        self._cfg_file = cfg or expanduser("~/.rtfd.cfg")
        self._cfg_obj = ConfigParser(interpolation=ExtendedInterpolation())
        if isfile(self._cfg_file):
            self._cfg_obj.read(self._cfg_file)

    def __str__(self):
        return "<%s object at %s, the config file is %s>" % (
            self.__class__.__name__, hex(id(self)), self._cfg_file
        )

    __repr__ = __str__

    def __getattr__(self, section):
        section = section.lower()
        if self._cfg_obj.has_section(section):
            return SectionHandler(self, section)
        raise AttributeError("No section: %s" % section)

    __getitem__ = __getattr__

    @property
    def sections(self):
        return self._cfg_obj.sections()

    def options(self, section):
        return self._cfg_obj.options(section)

    def items(self, section):
        return self._cfg_obj.items(section)
