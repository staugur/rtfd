# -*- coding: utf-8 -*-
"""
    api
    ~~~

    接口（作为Flask-PluginKit的一个插件）

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

import hmac
from hashlib import sha1
from collections import deque
from flask import request, jsonify, make_response, render_template_string,\
    current_app, Blueprint
from flask_pluginkit._compat import PY2, text_type
from .libs import ProjectManager, RTFD_BUILDER
from .utils import is_true
if PY2:
    from thread import start_new_thread
else:
    from _thread import start_new_thread

#: Build message map
MESSAGE_MAP = {}

#: Blueprint(bep) instead of vep
#:
#: ..versionadded:: 0.4.2
bp = Blueprint(
    "rtfd", "rtfd", static_folder="static", static_url_path="/assets"
)


@bp.route("/api", methods=["GET", "POST"])
def api_view():
    """RTFD接口视图"""
    res = dict(code=-1, msg=None)
    cfg = current_app.config.get("RTFD_CFG")
    Action = request.args.get("Action")
    if request.method == "GET":
        if Action == "describeProject":
            name = request.args.get("name")
            if name:
                cpm = ProjectManager(cfg)
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
            name = request.args.get("name")
            try:
                msg = MESSAGE_MAP[name].popleft()
            except (IndexError, KeyError):
                msg = ""
            #: 重置res响应数据
            isRaw = is_true(request.args.get("raw", True))
            if isRaw:
                res = msg
            else:
                res = dict(code=0, msg=msg)
    else:
        if Action == "buildProject":
            rb = RTFD_BUILDER(cfg)
            name = request.form.get("name", request.args.get("name"))
            branch = request.form.get(
                "branch", request.args.get("branch")
            ) or "master"
            if rb._cpm.has(name):
                def build(name, branch):
                    if name not in MESSAGE_MAP:
                        MESSAGE_MAP[name] = deque()
                    for _out in rb.build(name, branch, "api"):
                        MESSAGE_MAP[name].append(_out)
                start_new_thread(build, (name, branch))
                res.update(code=0, branch=branch, msg="ok")
            else:
                res.update(msg="Did not find this project %s" % name)
    response = make_response(jsonify(res))
    response.headers['Access-Control-Allow-Origin'] = '*'
    return response


@bp.route("/badge/<string:name>")
def badge_view(name):
    """RTFD徽章视图"""
    cpm = ProjectManager(current_app.config.get("RTFD_CFG"))
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


@bp.route("/webhook/<string:name>", methods=["POST"])
def webhook_view(name):
    """基于webhook自动构建文档"""
    res = dict(code=1, msg=None)
    agent = request.headers.get("User-Agent") or ""
    if agent.startswith("GitHub-Hookshot"):
        gst = "github"
    elif agent == "git-oschina-hook":
        gst = "gitee"
    else:
        res.update(msg="Unsupported git service provider")
        return jsonify(res)

    #: check params
    data = request.json
    if not data:
        res.update(msg="Invalid json data")
        return jsonify(res)
    cfg = current_app.config.get("RTFD_CFG")
    pm = ProjectManager(cfg)
    if not pm.has(name):
        res.update(code=404, msg="Not found project")
        return jsonify(res)

    def build(_name, _branch):
        rb = RTFD_BUILDER(cfg)
        if name not in MESSAGE_MAP:
            MESSAGE_MAP[name] = deque()
        for _out in rb.build(_name, _branch, "webhook"):
            MESSAGE_MAP[name].append(_out)

    #: hook handler for github or gitee
    if gst == "github":
        event = request.headers.get("X-GitHub-Event")
        if event == "ping":
            res.update(code=0, ping="pong")
        elif event in ("push", "release"):
            docsInfo = pm.get(name)
            secret = docsInfo.get("webhook_secret")
            sign_passing = True
            if secret:
                if isinstance(secret, text_type):
                    secret = secret.encode("utf-8")
                sign_passing = False
                signature = request.headers.get("X-Hub-Signature")
                if signature:
                    sign_method, sign_ret = signature.split("=")
                    if sign_method == "sha1":
                        if hmac.new(
                                secret,
                                request.data,
                                sha1).hexdigest() == sign_ret:
                            sign_passing = True
                        else:
                            res.update(msg="Verify signature failed")
                    else:
                        res.update(msg="Invalid signature method")
                else:
                    res.update(msg="Invalid signature header")
            if sign_passing is True:
                allow_build = True
                if event == "push":
                    branch = data.get("ref", "").split("/")[-1] or "master"
                else:
                    if data["action"] == "published":
                        branch = data["release"]["tag_name"]
                    else:
                        allow_build = False
                        res.update(
                            code=0,
                            msg="This action is ignored in the release event"
                        )
                if allow_build is True:
                    start_new_thread(build, (name, branch))
                    res.update(code=0, msg="ok", branch=branch)
        else:
            res.update(msg="Invalid event type")
    elif gst == "gitee":
        event = request.headers.get("X-Gitee-Event")
        if is_true(request.headers.get("X-Gitee-Ping")):
            event = "ping"
        else:
            if event == "Push Hook":
                event = "push"
            elif event == "Tag Push Hook":
                event = "release"
        if event == "ping":
            res.update(code=0, ping="pong")
        elif event in ("push", "release"):
            docsInfo = pm.get(name)
            secret = docsInfo.get("webhook_secret")
            sign_passing = True
            if secret:
                sign_passing = False
                token = request.headers.get("X-Gitee-Token")
                if token and secret == token:
                    sign_passing = True
                else:
                    res.update(msg="Verify token failed")
            if sign_passing is True:
                if event == "push":
                    branch = data.get("ref", "").split("/")[-1] or "master"
                else:
                    branch = data["ref"].split("/")[-1]
                start_new_thread(build, (name, branch))
                res.update(code=0, msg="ok", branch=branch)
        else:
            res.update(msg="Invalid event type")

    pm._logger.debug("Trigger webhook with %s event, from %s" % (event, gst))
    return jsonify(res)


def register():
    return dict(
        bep=dict(
            blueprint=bp,
            prefix="/rtfd"
        )
    )
