# -*- coding: utf-8 -*-
"""
    api
    ~~~

    接口（作为Flask-PluginKit的一个插件）

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from thread import start_new_thread
from collections import deque
from flask import request, jsonify, make_response, render_template_string
from .libs import ProjectManager, RTFD_BUILDER

#: Build message queue
_queue = deque()


def rtfd_api_view():
    """RTFD接口视图"""
    res = dict(code=-1, msg=None)
    Action = request.args.get("Action")
    if request.method == "GET":
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
        elif Action == "queryBuildmsg":
            try:
                msg = _queue.popleft()
            except IndexError:
                msg = ""
            #: 重置res响应数据
            isRaw = True if request.args.get("raw", True) in (
                1, "1", "on", True, "True", "true") else False
            if isRaw:
                res = msg
            else:
                res = dict(code=0, msg=msg)
    else:
        if Action == "buildProject":
            rb = RTFD_BUILDER()
            name = request.form.get("name", request.args.get("name"))
            branch = request.form.get(
                "branch", request.args.get("branch")) or "latest"
            if rb._cpm.has(name):
                def build(name, branch):
                    for _out in rb.build(name, branch, "api"):
                        _queue.append(_out)
                start_new_thread(build, (name, branch))
                res.update(code=0, msg="Already submitted asynchronous task")
            else:
                res.update(msg="Did not find this project %s" % name)
    response = make_response(jsonify(res))
    response.headers['Access-Control-Allow-Origin'] = '*'
    return response


def rtfd_badge_view(name):
    """RTFD徽章视图"""
    cpm = ProjectManager()
    status = cpm.get_for_badge(
        name, branch=request.args.get("branch") or "latest"
    )
    if status == "passing":
        statusText = '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="86" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="86" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#4c1" d="M35 0h51v20H35z"/><path fill="url(#b)" d="M0 0h86v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="595" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="410">passing</text><text x="595" y="140" transform="scale(.1)" textLength="410">passing</text></g> </svg>'
    elif status == "failing":
        statusText = '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="78" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="78" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#e05d44" d="M35 0h43v20H35z"/><path fill="url(#b)" d="M0 0h78v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="555" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="330">failing</text><text x="555" y="140" transform="scale(.1)" textLength="330">failing</text></g> </svg>'
    else:
        statusText = '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="96" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="96" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#dfb317" d="M35 0h61v20H35z"/><path fill="url(#b)" d="M0 0h96v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="645" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="510">unknown</text><text x="645" y="140" transform="scale(.1)" textLength="510">unknown</text></g> </svg>'
    resp = make_response(render_template_string(statusText))
    resp.headers["Content-Type"] = "image/svg+xml; charset=utf-8"
    return resp


def register():
    return dict(
        vep=[
            dict(
                rule="/api/rtfd",
                view_func=rtfd_api_view,
                methods=["GET", "POST"]
            ),
            dict(
                rule="/badge/rtfd/<string:name>",
                view_func=rtfd_badge_view
            )
        ]
    )
