# -*- coding: utf-8 -*-
"""
    api
    ~~~

    接口（作为Flask-PluginKit的一个插件）

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from flask import request, jsonify, make_response, current_app
from .libs import ProjectManager


def rtfd_api_view():
    """RTFD接口视图"""
    res = dict(code=-1, msg=None)
    if request.method == "GET":
        current_app.logger.info(request.url)
        Action = request.args.get("Action")
        if Action == "describeProject":
            name = request.args.get("name")
            if name:
                cpm = ProjectManager()
                if cpm.has(name):
                    data = cpm.get_for_api(name)
                    if data:
                        res.update(code=0, data=data)
                    else:
                        res.update(code=1, msg="invalid data")
                else:
                    res.update(code=404)
            else:
                res.update(msg='param error')
    response = make_response(jsonify(res))
    response.headers['Access-Control-Allow-Origin'] = '*'
    return response


def register():
    return dict(
        vep=[
            dict(
                rule="/api/rtfd",
                view_func=rtfd_api_view,
                methods=["GET", "POST"]
            )
        ]
    )
