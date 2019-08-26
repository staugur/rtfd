# -*- coding: utf-8 -*-
"""
    libs
    ~~~~

    核心库

    :copyright: (c) 2019 by staugur.
    :license: BSD 3-Clause, see LICENSE for more details.
"""

from os import mkdir, remove, listdir
from shutil import rmtree
from os.path import expanduser, dirname, join, abspath, isdir, isfile
from jinja2 import Template
from time import strftime
from .utils import ProjectStorage, run_cmd, run_cmd_stream
from .exceptions import ProjectExistsError, ProjectNotFound
from .config import CfgHandler
from ._log import Logger


class ProjectManager(object):

    def __init__(self, cfg=None):
        self._cfg_file = cfg or expanduser("~/.rtfd.cfg")
        self._cfg_handler = CfgHandler(self._cfg_file)
        self._cps = ProjectStorage(self._cfg_file)
        self._logger = Logger("sys", self._cfg_file).getLogger

    def create(self, name, url, **kwargs):
        name = name.lower()
        if self.has(name):
            raise ProjectExistsError("This project '%s' already exists" % name)
        else:
            kwargs.update(url=url, _dn="%s.%s" %
                          (name, self._cfg_handler.nginx.dn))
            self._logger.info(
                "Project.Create: name is %s, create params is %s" % (name, kwargs))
            return self._cps.set(name, kwargs)

    def has(self, name):
        name = name.lower()
        if self._cps.get(name):
            return True
        else:
            return False

    def get(self, name, default=None):
        name = name.lower()
        if self.has(name):
            return self._cps.get(name, default=default)
        return default

    def get_for_badge(self, name, branch="latest"):
        name = name.lower()
        if self.has(name):
            data = self.get(name)
            if branch == "latest":
                branch = data["latest"]
            key = "_build_%s" % branch
            build_info = data.get(key)
            if build_info and isinstance(build_info, dict):
                return build_info.get("status", "unknown")
            else:
                return "unknown"
        else:
            return "unknown"

    def get_for_api(self, name):
        data = self.get(name)
        languages = data.get("languages") or data.get("default_language")
        languages = languages.split(",")
        versions = {}
        for lang in languages:
            lang_dir = join(self._cfg_handler.g.base_dir, "docs", name, lang)
            if isdir(lang_dir):
                _versions = listdir(lang_dir)
                _versions.remove(data["latest"])
                versions[lang] = _versions
            else:
                versions[lang] = []
        resp = dict(languages=languages, versions=versions,
                    latest=data.get("latest"), url=data.get("url"),
                    dn=data.get("_dn"), sourcedir=data.get("sourcedir"),
                    single=True if data.get("single") in (
                        True, "true", "True") else False,
                    showNav=True if data.get("showNav", True) in (True, "true", "True") else False)
        return resp

    def update(self, name, **kwargs):
        name = name.lower()
        if self.has(name):
            data = self.get(name)
            if isinstance(data, dict):
                if "default_language" in kwargs:
                    if kwargs["default_language"] not in data["languages"].split(","):
                        kwargs["default_language"] = data["languages"].split(",")[
                            0]
                data.update(kwargs)
                for k, v in data.iteritems():
                    if isinstance(v, unicode):
                        v.encode("utf-8")
                    data[k] = v
                self._logger.info(
                    "Project.Update: name is %s, update params is %s" % (name, kwargs))
                return self._cps.set(name, data)

    def remove(self, name):
        name = name.lower()
        if self.has(name):
            self._logger.info(
                "Project.Remove: name is %s, will remove docs and nginx itself, then reload nginx" % name)
            #: 删除文档和nginx
            PROJECT_DOCS = join(self._cfg_handler.g.base_dir, "docs", name)
            NGINX_FILE = join(self._cfg_handler.g.base_dir,
                              "nginx", "%s.conf" % name)
            if isdir(PROJECT_DOCS):
                rmtree(PROJECT_DOCS)
            if isfile(NGINX_FILE):
                remove(NGINX_FILE)
                self.__reload_nginx()
            return self._cps.set(name, '')

    def __reload_nginx(self):
        #: reload nginx
        nginx_exec = self._cfg_handler.nginx.get("exec")
        if nginx_exec:
            exitcode, _, _ = run_cmd(nginx_exec, '-t')
            if exitcode == 0:
                run_cmd(nginx_exec, '-s', 'reload')
            else:
                self._logger.warning("Project.Nginx: Syntax check failed")

    def nginx_builder(self, name):
        name = name.lower()
        data = self.get(name)
        if not data or not isinstance(data, dict):
            raise ProjectNotFound("Did not find this project '%s'" % name)
        DOCS_DIR = join(self._cfg_handler.g.base_dir, "docs")
        NGINX_DIR = join(self._cfg_handler.g.base_dir, "nginx")
        NGINX_DN = self._cfg_handler.nginx.dn
        NGINX_SSL = True if self._cfg_handler.nginx.get(
            "ssl") in ("on", "true", "True", True) else False
        if not isdir(DOCS_DIR):
            mkdir(DOCS_DIR)
        if not isdir(NGINX_DIR):
            mkdir(NGINX_DIR)
        #: 通用模板
        multi_lang_tpl = '''#: Automatic generated by rtfd in {{ t }}
server {
    listen 80;
    {% if ssl %}
    listen 443 ssl http2;
    {% endif %}
    server_name {{ name }}.{{ nginx_dn }};
    charset utf-8;
    root {{ docs_dir }}/{{ name }}/;
    index index.html;
    error_page 403 =404 /404.html;
    {% if ssl %}
        {{ ssl_tpl }}
    {% endif %}
    location = / {
        return 302 /{{ default_language }}/latest/$is_args$args;
    }
    {% for lang in languages.split(",") %}
    location /{{ lang }}/latest/ {
        alias {{ docs_dir }}/{{ name }}/{{ lang }}/latest/;
    }
    {% endfor %}
}'''
        #: 单一版本的模板
        single_lang_tpl = '''#: Automatic generated by rtfd in {{ t }}
server {
    listen 80;
    {% if ssl %}
    listen 443 ssl http2;
    {% endif %}
    server_name {{ name }}.{{ nginx_dn }};
    charset utf-8;
    root {{ docs_dir }}/{{ name }}/{{ default_language }}/latest/;
    index index.html;
    {% if ssl %}
        {{ ssl_tpl }}
    {% endif %}
}'''
        #: SSL模板
        if NGINX_SSL:
            nginx_tpl = '''
    ssl_certificate %s;
    ssl_certificate_key %s;
    ssl_stapling on;
    ssl_stapling_verify on;
    resolver 8.8.8.8 114.114.114.114 valid=300s;
    resolver_timeout 5s;
    ssl_session_cache builtin:1000 shared:SSL:10m;
    ssl_session_tickets on;
    ssl_session_timeout  10m;
    ssl_protocols TLSv1 TLSv1.1 TLSv1.2 TLSv1.3;
    ssl_ciphers TLS13-AES-256-GCM-SHA384:TLS13-CHACHA20-POLY1305-SHA256:TLS13-AES-128-GCM-SHA256:TLS13-AES-128-CCM-8-SHA256:TLS13-AES-128-CCM-SHA256:EECDH+CHACHA20:EECDH+CHACHA20-draft:EECDH+ECDSA+AES128:EECDH+aRSA+AES128:RSA+AES128:EECDH+ECDSA+AES256:EECDH+aRSA+AES256:RSA+AES256:EECDH+ECDSA+3DES:EECDH+aRSA+3DES:RSA+3DES:!MD5;
    ssl_prefer_server_ciphers on;
    add_header Strict-Transport-Security "max-age=%s; preload";
''' % (self._cfg_handler.nginx.ssl_crt, self._cfg_handler.nginx.ssl_key, self._cfg_handler.nginx.get("ssl_hsts_maxage") or 31536000)
        else:
            nginx_tpl = ''
        default_language = data.get("default_language") or "en"
        languages = data.get("languages") or default_language
        sgl = True if data.get("single") in (True, "true", "True") else False
        tpl = Template(single_lang_tpl) if sgl else Template(multi_lang_tpl)
        rendered_nginx_conf = tpl.render(
            t=strftime('%Y-%m-%d %H:%M:%S'), name=name, nginx_dn=NGINX_DN,
            docs_dir=DOCS_DIR, languages=languages,
            default_language=default_language, ssl=NGINX_SSL, ssl_tpl=nginx_tpl
        )
        self._logger.info(
            "Project.Nginx: name is %s, will render nginx configure" % name)
        with open(join(NGINX_DIR, "%s.conf" % name), "w") as fp:
            fp.write(rendered_nginx_conf)
        #: reload nginx
        self._logger.info(
            "Project.Nginx: name is %s, will reload nginx" % name)
        self.__reload_nginx()


class RTFD_BUILDER(object):

    def __init__(self, cfg=None):
        self._cfg_file = cfg or expanduser("~/.rtfd.cfg")
        self._cpm = ProjectManager(self._cfg_file)
        self._build_sh = join(dirname(abspath(__file__)), "scripts/builder.sh")
        self._logger = Logger("sys", self._cfg_file).getLogger

    def build(self, name, branch="master", stream=True, sponsor=None):
        res = dict(code=1)
        if isinstance(name, unicode):
            name = name.encode("utf8")
        if isinstance(branch, unicode):
            branch = branch.encode("utf8")
        if not self._cpm.has(name):
            res.update(_err="Did not find this project %s" % name)
            return res
        data = self._cpm.get(name)
        if data and isinstance(data, dict) and "url" in data:
            if branch == "latest":
                branch = data["latest"]
            msg = "RTFD.Builder: build %s with branch %s" % (name, branch)
            self._logger.debug(msg)
            cmd = ['bash', self._build_sh, '-n', name, '-u', data["url"],
                   '-b', branch, '-c', self._cfg_file]
            #: 响应信息
            status = "failing"
            if stream is True:
                out = []
                for i in run_cmd_stream(*cmd):
                    if "Build Successfully" in i:
                        status = "passing"
                        res.update(code=0)
                    print(i)
                    out.append(i)
                res.update(_output="\n".join(out))
            else:
                code, out, err = run_cmd(*cmd)
                res.update(code=code, _output=out, _err=err)
                if code == 0 and "Build Successfully" in out:
                    status = "passing"
            #: 更新构建信息
            _build_info = {"_build_%s" % branch: dict(
                btime=strftime('%Y-%m-%d %H:%M:%S'), status=status, sponsor=sponsor)
            }
            self._cpm.update(name, **_build_info)
        else:
            res.update(_err="Not found name, data error for %s" % name)
        #: 当code为0基本上表示构建成功了
        print(res)
        return res
