# -*- coding: utf-8 -*-
"""
    app
    ~~~

    Flask App

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from flask_pluginkit import PluginManager, Flask

app = Flask(__name__)
PluginManager(app, plugin_packages=["rtfd"])
