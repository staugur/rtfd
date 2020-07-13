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
from flask_pluginkit._compat import text_type, string_types, PY2
from .utils import ProjectStorage, run_cmd, run_cmd_stream, is_true, get_now,\
    is_domain, get_public_giturl, get_git_service_provider, is_project_name
from .exceptions import ProjectExistsError, ProjectNotFound, \
    ProjectUnallowedError, CfgNotFound, RTFDError
from .config import CfgHandler
from ._log import Logger


class ProjectManager(object):

    def __init__(self, cfg=None):
        self._cfg_file = cfg or expanduser("~/.rtfd.cfg")
        if not isfile(self._cfg_file):
            raise CfgNotFound("Not found config file: %s" % self._cfg_file)
        self._cfg_handler = CfgHandler(self._cfg_file)
        self._cps = ProjectStorage(self._cfg_file)
        self._logger = Logger("sys", self._cfg_file).getLogger
        self._unallow_names = self._cfg_handler.g.get(
            "unallowed_name", default="").split(",")
        if "www" not in self._unallow_names:
            self._unallow_names.append("www")

    def create(self, name, url, **kwargs):
        name = name.lower().replace("_", "-").replace(" ", "")
        if not is_project_name(name):
            raise ProjectUnallowedError("Invalid project name")
        if name in self._unallow_names:
            raise ProjectUnallowedError("Unallowed project name %s" % name)
        if self.has(name):
            raise ProjectExistsError("This project %s already exists" % name)
        else:
            kwargs.update(
                url=url,
                _dn="%s.%s" % (name, self._cfg_handler.nginx.dn)
            )
            if is_domain(kwargs.get("custom_domain")):
                if kwargs["custom_domain"] == kwargs["_dn"]:
                    raise RTFDError("Custom domain name is duplicated "
                                    "with default domain name")
            self._logger.info(
                "Project.Create: name is %s, create params is %s" %
                (name, kwargs)
            )
            return self._cps.set(name, kwargs)

    def has(self, name):
        name = name.lower()
        if self._cps.get(name):
            return True
        else:
            return False

    def has_custom_domain(self, dn):
        dns = []
        for data in self._cps.list.values():
            if data and isinstance(data, dict):
                if data.get("custom_domain"):
                    dns.append(data["custom_domain"])
                dns.append(data["_dn"])
        return dn in dns

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
        url = data["url"]
        _type = data.get("_type") or "public"
        languages = data.get("languages") or data.get("default_language")
        languages = languages.split(",")
        versions = {}
        for lang in languages:
            lang_dir = join(self._cfg_handler.g.base_dir, "docs", name, lang)
            if isdir(lang_dir):
                _versions = listdir(lang_dir)
                try:
                    _versions.remove(data["latest"])
                except ValueError:
                    pass
                versions[lang] = sorted(_versions, reverse=True)
            else:
                versions[lang] = []
        resp = dict(
            languages=languages, versions=versions, latest=data.get("latest"),
            url=url if _type == "public" else get_public_giturl(url),
            dn=data.get("_dn"),
            custom_dn=data.get("custom_domain") if is_domain(
                data.get("custom_domain")) else False,
            sourcedir=data.get("sourcedir"),
            single=is_true(data.get("single")),
            #: TODO For compatibility, it will be scrapped later
            showNav=is_true(data.get("show_nav", True)),
            showNavGit=is_true(data.get("show_nav_git", True)),
            show_nav=is_true(data.get("show_nav", True)),
            icon='data:image/png;base64,'
            'iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAlUlEQVQ'
            '4T92S0Q0CMQxDnydBtwEbABvcRjAKK7DBscGNwCZGRbSKDigB/uhv4l'
            'c7svjxqeptj8AeWL9hTpJ2dScCLsAqY0hS00WA7+ITcJA0p2AhQgUMw'
            'BHYdAAtxoODYs92hb1k1BhdQMy6hKYAvRukANHB8lYpwB84+DTCVMrz'
            'dQ/ib7ZvsI6Ds6RtmbciZXr/bOcKjCNuESAd+XoAAAAASUVORK5CYII=',
            type=_type, show_nav_git=is_true(data.get("show_nav_git", True)),
            gsp=data.get("_gsp", get_git_service_provider(url)),
        )
        return resp

    def update(self, name, **kwargs):
        name = name.lower()
        if self.has(name):
            data = self.get(name)
            if data and isinstance(data, dict) and kwargs:
                if "languages" in kwargs:
                    if "default_language" not in kwargs:
                        kwargs['default_language'] = data["default_language"]
                if "default_language" in kwargs:
                    lgs = kwargs.get("languages", data["languages"]).split(",")
                    if kwargs["default_language"] not in lgs:
                        kwargs["default_language"] = lgs[0]
                data.update(kwargs)
                self._logger.info(
                    "Project.Update: name is %s, update params is %s" %
                    (name, kwargs)
                )
                self._cps.set(name, data)
                #: update nginx template
                if "languages" in kwargs or "default_language" in kwargs or \
                        "single" in kwargs or "custom_domain" in kwargs or \
                        "ssl" in kwargs or "ssl_crt" in kwargs or \
                        "ssl_key" in kwargs or "ssl_hsts_maxage" in kwargs:
                    self._logger.info("Project.Update: rendering nginx again")
                    self.nginx_builder(name)

    def remove(self, name):
        name = name.lower()
        if self.has(name):
            if PY2 and isinstance(name, text_type):
                name = name.encode("utf-8")
            self._logger.info(
                "Project.Remove: name is %s, will remove docs and nginx, "
                "then reload nginx" % name
            )
            #: 删除文档和nginx
            PROJECT_DOCS = join(self._cfg_handler.g.base_dir, "docs", name)
            NGINX_DIR = join(self._cfg_handler.g.base_dir, "nginx")
            default_nginx_file = join(NGINX_DIR, "%s.conf" % name)
            custom_nginx_file = join(NGINX_DIR, "%s.ext.conf" % name)
            if isdir(PROJECT_DOCS):
                rmtree(PROJECT_DOCS)
            if isfile(default_nginx_file) or isfile(custom_nginx_file):
                if isfile(default_nginx_file):
                    remove(default_nginx_file)
                if isfile(custom_nginx_file):
                    remove(custom_nginx_file)
                self.__reload_nginx()
            return self._cps.remove(name)

    def __reload_nginx(self):
        #: reload nginx
        nginx_exec = self._cfg_handler.nginx.get("exec")
        if nginx_exec:
            if " " in nginx_exec:
                check_cmd = nginx_exec.split(" ") + ["-t"]
                reload_cmd = nginx_exec.split(" ") + ["-s", "reload"]
            else:
                check_cmd = [nginx_exec, "-t"]
                reload_cmd = [nginx_exec, "-s", "reload"]
            exitcode, _, _ = run_cmd(*check_cmd)
            if exitcode == 0:
                exitcode, _, _ = run_cmd(*reload_cmd)
                if exitcode == 0:
                    self._logger.info("Project.Nginx: reload succssfully")
                else:
                    self._logger.warning("Project.Nginx: reload failed")
            else:
                self._logger.warning("Project.Nginx: Syntax check failed")

    def nginx_builder(self, name):
        name = name.lower()
        if not self.has(name):
            raise ProjectNotFound("No such project %s" % name)
        data = self.get(name)
        if not data or not isinstance(data, dict) or \
                "default_language" not in data or \
                "languages" not in data:
            raise ProjectNotFound("The project data of %s is wrong." % name)
        DOCS_DIR = join(self._cfg_handler.g.base_dir, "docs")
        NGINX_DIR = join(self._cfg_handler.g.base_dir, "nginx")
        if not isdir(DOCS_DIR):
            mkdir(DOCS_DIR)
        if not isdir(NGINX_DIR):
            mkdir(NGINX_DIR)
        #: 通用模板，需要参数：
        #: t - string: 当前时间
        #: ssl - true/false: 是否开启ssl
        #: ssl_cfg - string: ssl配置内容
        #: domain_name - string: 域名
        #: docs_dir - path: 文档项目所在的父目录
        #: name - string: 文档项目名
        #: languages - string: 文档语言(remove in 0.4.4)
        #: default_language - string: 文档默认语言
        multi_tpl = '''#: Automatic generated by rtfd at {{ t }}
server {
    listen 80;
    {%- if ssl %}
    listen 443 ssl http2;
    {%- endif %}
    server_name {{ domain_name }};
    charset utf-8;
    root {{ docs_dir }}/{{ name }}/;
    index index.html;
    set $home /{{ default_language }}/latest;
    error_page 403 =404 /404.html;
    {%- if ssl -%}
        {{ ssl_cfg }}
    {%- endif %}
    location / {
        if (-e $document_root$home$document_uri) {
            return 302 $home$document_uri$is_args$args;
        }
    }
}'''
        #: 单一版本的模板，相对通用模板至少一个languages参数
        single_tpl = '''#: Automatic generated by rtfd at {{ t }}
server {
    listen 80;
    {%- if ssl %}
    listen 443 ssl http2;
    {%- endif %}
    server_name {{ domain_name }};
    charset utf-8;
    root {{ docs_dir }}/{{ name }}/{{ default_language }}/latest/;
    index index.html;
    {%- if ssl -%}
        {{ ssl_cfg }}
    {%- endif %}
}'''
        #: SSL模板，需要传递证书、私钥、过期三个参数
        ssl_tpl = '''
    if ($scheme = http) {
        return 301 https://$server_name$request_uri;
    }
    ssl_certificate {{ ssl_crt }};
    ssl_certificate_key {{ ssl_key }};
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
    {%- if ssl_hsts_maxage|int > 0 %}
    add_header Strict-Transport-Security "max-age=%s; preload";
    {%- endif %}'''
        #: 全局默认的域名
        default_dn = "%s.%s" % (name, self._cfg_handler.nginx.dn)
        default_ssl = is_true(self._cfg_handler.nginx.get("ssl"))
        default_ssl_crt = self._cfg_handler.nginx.ssl_crt
        default_ssl_key = self._cfg_handler.nginx.ssl_key
        default_nginx_file = join(NGINX_DIR, "%s.conf" % name)
        default_hstsmaxage = self._cfg_handler.nginx.get("ssl_hsts_maxage", 0)
        #: 项目自定义域名
        custom_dn = data.get("custom_domain")
        custom_ssl = is_true(data.get("ssl"))
        custom_ssl_crt = data.get("ssl_crt")
        custom_ssl_key = data.get("ssl_key")
        cusrom_hstsmaxage = data.get("ssl_hsts_maxage", 0)
        custom_nginx_file = join(NGINX_DIR, "%s.ext.conf" % name)
        #: 项目其他信息
        default_language = data["default_language"]
        languages = data["languages"]
        is_single = is_true(data.get("single"))
        #: ssl默认模板渲染
        default_ssl_cfg = Template(ssl_tpl).render(
            ssl_crt=default_ssl_crt,
            ssl_key=default_ssl_key,
            ssl_hsts_maxage=default_hstsmaxage,
        )
        #: 默认域名模板渲染
        tpl = Template(single_tpl) if is_single else Template(multi_tpl)
        #: 渲染并写入默认域名
        rendered_nginx_conf = tpl.render(
            name=name, domain_name=default_dn, docs_dir=DOCS_DIR,
            languages=languages, default_language=default_language,
            t=get_now(), ssl=default_ssl, ssl_cfg=default_ssl_cfg,
        )
        self._logger.info(
            "Project.Nginx: name is %s, will generate default nginx" % name
        )
        with open(default_nginx_file, "w") as fp:
            fp.write(rendered_nginx_conf)
        #: 渲染并写入自定义域名
        if is_domain(custom_dn):
            #: ssl自定义域名模板渲染
            custom_ssl_cfg = Template(ssl_tpl).render(
                ssl_crt=custom_ssl_crt,
                ssl_key=custom_ssl_key,
                ssl_hsts_maxage=cusrom_hstsmaxage,
            )
            rendered_nginx_conf = tpl.render(
                name=name, domain_name=custom_dn, docs_dir=DOCS_DIR,
                languages=languages, default_language=default_language,
                t=get_now(), ssl=custom_ssl, ssl_cfg=custom_ssl_cfg,
            )
            self._logger.info(
                "Project.Nginx: name is %s, will generate custom nginx" % name
            )
            with open(custom_nginx_file, "w") as fp:
                fp.write(rendered_nginx_conf)
        else:
            #: 自定义域名不存在，如果发现nginx配置文件，则删除
            if isfile(custom_nginx_file):
                self._logger.info(
                    "Project.Nginx: Found custom nginx for "
                    "%s, will remove it" % name
                )
                remove(custom_nginx_file)
        #: reload nginx
        self._logger.info(
            "Project.Nginx: name is %s, will reload nginx" % name
        )
        self.__reload_nginx()


class RTFD_BUILDER(object):

    def __init__(self, cfg=None):
        self._cfg_file = cfg or expanduser("~/.rtfd.cfg")
        self._cpm = ProjectManager(self._cfg_file)
        self._build_sh = join(dirname(abspath(__file__)), "scripts/builder.sh")
        self._logger = self._cpm._logger

    def build(self, name, branch="master", sender=None):
        if PY2 and isinstance(name, text_type):
            name = name.encode("utf-8")
        if not self._cpm.has(name):
            yield "Did not find this project %s" % name
            return
        data = self._cpm.get(name)
        if data and isinstance(data, dict) and "url" in data:
            if branch == "latest":
                branch = data["latest"]
            if not PY2 and not isinstance(branch, string_types):
                branch = branch.decode("utf-8")
            msg = "RTFD.Builder: build %s with branch %s" % (name, branch)
            self._logger.debug(msg)
            cmd = [
                'bash', self._build_sh, '-n', name, '-u', data["url"],
                '-b', branch, '-c', self._cfg_file
            ]
            #: 响应信息
            status = "failing"
            usedtime = -1
            for i in run_cmd_stream(*cmd):
                if "Build Successfully" in i:
                    status = "passing"
                    try:
                        usedtime = int(i.split(" ")[2])
                    except (ValueError, TypeError):
                        pass
                yield i
            #: 更新构建信息
            _build_info = {"_build_%s" % branch: dict(
                btime=get_now(),
                status=status,
                sender=sender,
                usedtime=usedtime,
            )}
            self._cpm.update(name, **_build_info)
        else:
            yield "Not found name, data error for %s" % name
