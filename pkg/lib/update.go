/*
   Copyright 2021 Hiroshi.tao

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pkg/tcw.im/rtfd/pkg/conf"
	"pkg/tcw.im/rtfd/pkg/util"
	"pkg/tcw.im/rtfd/vars"

	"pkg.tcw.im/gtc"
)

// allowEmptyFields 值可以为 - 表示重置为空的字段
var allowEmptyFields = []string{"requirement", "index", "secret", "before", "after"}

// 更新文档项目配置结构体
type updateHook struct {
	pm     *ProjectManager
	opt    *Options
	render bool // 重新渲染nginx
}

// 根据要更新的字段选择对应处理函数
func (u *updateHook) handle(field string) (fn func(value any) error, err error) {
	switch strings.ToLower(field) {
	case "url":
		fn = u.url
	case "latest":
		fn = u.latest
	case "version":
		fn = u.version
	case "single":
		fn = u.single
	case "sourcedir", "source":
		fn = u.sourceDir
	case "lang":
		fn = u.lang
	case "requirement":
		fn = u.requirement
	case "install":
		fn = u.install
	case "index":
		fn = u.index
	case "shownav":
		fn = u.showNav
	case "hidegit":
		fn = u.hideGit
	case "secret":
		fn = u.secret
	case "customdomain", "domain":
		fn = u.customDomain
	case "builder":
		fn = u.builder
	case "beforehook", "before":
		fn = u.beforeHook
	case "afterhook", "after":
		fn = u.afterHook
	case "ssl":
		fn = u.ssl
	case "meta":
		fn = u.meta
	default:
		err = errors.New("invalid field")
	}
	return
}

func (u *updateHook) url(value any) error {
	rawurl := value.(string)

	typ, err := util.CheckGitURL(rawurl)
	if err != nil {
		return err
	}

	isPublic := false
	if typ == "public" {
		isPublic = true
	}

	rawurl = strings.TrimSuffix(rawurl, ".git")
	gsp, err := util.GitServiceProvider(rawurl)
	if err != nil {
		return err
	}

	u.opt.URL = rawurl
	u.opt.IsPublic = isPublic
	u.opt.GSP = gsp
	return nil
}

func (u *updateHook) latest(value any) error {
	br := value.(string)
	// 检测br，避免安全风险
	if strings.HasPrefix(br, "/") || strings.HasPrefix(br, ".") {
		return errors.New("illegal latest")
	}
	pd := filepath.Join(u.pm.cfg.BaseDir(), "docs", u.opt.Name)
	for _, lang := range strings.Split(u.opt.Lang, ",") {
		ln := filepath.Join(pd, lang, "latest")
		src := filepath.Join(pd, lang, br)
		if _, err := os.Lstat(ln); err == nil {
			os.Remove(ln)
		}
		err := os.Symlink(src, ln)
		if err != nil {
			return err
		}
	}
	u.opt.Latest = br
	return nil
}

func (u *updateHook) version(value any) error {
	ver := strings.TrimSpace(value.(string))
	if ver == "" {
		return errors.New("invalid version value")
	}
	if !u.pm.CFG().HasPyVersion(ver) {
		return fmt.Errorf(
			"unsupported python version: %s, available: %s",
			ver, strings.Join(u.pm.CFG().PyVersions(), ", "),
		)
	}
	u.opt.Version = PyVer(ver)
	return nil
}

func (u *updateHook) single(value any) error {
	u.opt.Single = gtc.IsTrue(value.(string))
	u.render = true
	return nil
}

func (u *updateHook) sourceDir(value any) error {
	sd := value.(string)
	// 检测sd，避免安全风险
	if strings.HasPrefix(sd, "/") || strings.HasPrefix(sd, "..") {
		return errors.New("illegal sourcedir")
	}
	u.opt.SourceDir = sd
	return nil
}

func (u *updateHook) lang(value any) error {
	u.opt.Lang = value.(string)
	u.render = true
	return nil
}

func (u *updateHook) requirement(value any) error {
	req := value.(string)
	// 检测req，避免安全风险
	if strings.HasPrefix(req, "/") || strings.HasPrefix(req, "..") {
		return errors.New("illegal requirement")
	}
	u.opt.Requirement = req
	return nil
}

func (u *updateHook) install(value any) error {
	u.opt.Install = gtc.IsTrue(value.(string))
	return nil
}

func (u *updateHook) index(value any) error {
	u.opt.Index = value.(string)
	return nil
}

func (u *updateHook) showNav(value any) error {
	u.opt.ShowNav = gtc.IsTrue(value.(string))
	return nil
}

func (u *updateHook) hideGit(value any) error {
	u.opt.HideGit = gtc.IsTrue(value.(string))
	return nil
}

func (u *updateHook) secret(value any) error {
	u.opt.Secret = value.(string)
	return nil
}

func (u *updateHook) customDomain(value any) error {
	dn := strings.ToLower(value.(string))

	// 清除自定义域名（域名占用记录随项目配置一起落库）
	if gtc.IsFalse(dn) {
		u.opt.CustomDomain = ""
		u.render = true
		return nil
	}

	if !util.IsDomain(dn) {
		return errors.New("invalid custom domain")
	}
	if u.pm.HasCustomDomain(dn) {
		return errors.New("this domain name already exists")
	}

	u.opt.CustomDomain = dn
	u.render = true
	return nil
}

func (u *updateHook) builder(value any) error {
	// 兼容字符串与 BuilderType，并校验取值范围
	bt := BuilderType(strings.TrimSpace(util.ParamString(value)))
	switch bt {
	case HTMLBuilder, DirHTMLBuilder, SingleHTMLBuilder:
		u.opt.Builder = bt
		return nil
	}
	return errors.New("invalid builder")
}

func (u *updateHook) beforeHook(value any) error {
	u.opt.BeforeHook = value.(string)
	return nil
}

func (u *updateHook) afterHook(value any) error {
	u.opt.AfterHook = value.(string)
	return nil
}

func (u *updateHook) ssl(value any) error {
	v := value.(string)

	// 取消自定义域名SSL
	if gtc.IsFalse(v) {
		u.opt.SSL = false
		u.opt.SSLPublic = ""
		u.opt.SSLPrivate = ""
		u.render = true
		return nil
	}

	cert := strings.Split(v, ",")
	if len(cert) != 2 {
		return errors.New("invalid ssl")
	}
	pub := cert[0]
	pri := cert[1]
	if !gtc.IsFile(pub) || !gtc.IsFile(pri) {
		return errors.New("not found ssl file")
	}
	u.opt.SSL = true
	u.opt.SSLPublic = pub
	u.opt.SSLPrivate = pri
	u.render = true
	return nil
}

func (u *updateHook) meta(value any) error {
	// value format key=value, update only one at a time
	v := value.(string)
	ms := strings.Split(v, "=")
	if len(ms) != 2 {
		return fmt.Errorf("invalid meta field: %s", v)
	}
	key := strings.ToLower(ms[0])
	val := ms[1]
	if strings.HasPrefix(key, "_") {
		return errors.New("cannot change system reserved fields")
	}
	return u.opt.UpdateMeta(key, val)
}

// ParseUpdateRule 解析text形式的更新规则，返回字段->值的映射（CLI 与 API 共用）。
// 格式：Field:Value,Field:Value（分隔符由 sep 指定，缺省是英文冒号）；
// 其中 sslcrt 与 sslpri 合并为 ssl 字段（值以英文逗号分隔），
// requirement、index、secret、before、after的值可以为 - 表示重置为空。
func ParseUpdateRule(text, sep string) (rule map[string]any, err error) {
	if sep == "" {
		sep = ":"
	}
	rule = make(map[string]any)
	var ssl string
	for _, kv := range strings.Split(text, ",") {
		kvs := strings.Split(kv, sep)
		if len(kvs) != 2 {
			return nil, fmt.Errorf("invalid %s", kv)
		}
		field := kvs[0]
		value := kvs[1]
		if field == "" || value == "" {
			continue
		}
		switch field {
		case "sslcrt":
			ssl = value
		case "sslpri":
			ssl += "," + value
		case "ssl":
			// ssl不在上方合并列表中，仅支持取消（值为0、false、off）
			if !gtc.IsFalse(value) {
				return nil, errors.New("invalid ssl")
			}
			ssl = value
		default:
			if value == vars.ResetEmpty && gtc.StrInSlice(field, allowEmptyFields) {
				value = ""
			}
			rule[field] = value
		}
	}
	if ssl != "" {
		rule["ssl"] = ssl
	}
	if len(rule) == 0 {
		err = errors.New("empty rule")
	}
	return
}

// ParseUpdateFile 解析仓库内的 .rtfd.ini 规则文件（构建期回写），
// 仅白名单字段参与更新，同时返回文件内容MD5供调用方判断是否需要更新
func ParseUpdateFile(path string) (rule map[string]any, md5 string, err error) {
	if !gtc.IsFile(path) {
		err = errors.New("not found file")
		return
	}
	cfg, err := conf.New(path)
	if err != nil {
		return
	}
	md5, _ = gtc.MD5File(path)

	rule = make(map[string]any)
	for k, v := range cfg.SecHash("project") {
		if gtc.StrInSlice(k, []string{"latest"}) {
			rule[k] = v
		}
	}
	for k, v := range cfg.SecHash("sphinx") {
		if gtc.StrInSlice(k, []string{"sourcedir", "lang", "builder"}) {
			rule[k] = v
		}
	}
	for k, v := range cfg.SecHash("python") {
		if gtc.StrInSlice(k, []string{"version", "requirement", "install", "index"}) {
			rule[k] = v
		}
	}
	if len(rule) == 0 {
		err = errors.New("empty rule")
	}
	return
}
