# -*- coding: utf-8 -*-
"""
    exceptions
    ~~~~~~~~~~

    Exception Classes

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""


class RTFDError(Exception):
    pass


class ProjectExistsError(RTFDError):
    pass


class ProjectNotFound(RTFDError):
    pass


class ProjectUnallowedError(RTFDError):
    pass


class CfgNotFound(RTFDError):
    pass
