package lib

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tcw.im/rtfd/pkg/util"

	"tcw.im/ufc"
)

// 更新文档项目配置结构体
type updateHook struct {
	pm     *ProjectManager
	opt    *Options
	render bool // 重新渲染nginx
}

// 根据要更新的字段选择对应处理函数
func (u *updateHook) handle(field string) (fn func(value interface{}) error, err error) {
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
	default:
		err = errors.New("invalid field")
	}
	return
}

func (u *updateHook) url(value interface{}) error {
	rawurl := value.(string)

	typ, err := util.CheckGitURL(rawurl)
	if err != nil {
		return err
	}

	isPublic := false
	if typ == "public" {
		isPublic = true
	}

	if strings.HasSuffix(rawurl, ".git") {
		rawurl = strings.TrimSuffix(rawurl, ".git")
	}
	gsp, err := util.GitServiceProvider(rawurl)
	if err != nil {
		return err
	}

	u.opt.URL = rawurl
	u.opt.IsPublic = isPublic
	u.opt.GSP = gsp
	return nil
}

func (u *updateHook) latest(value interface{}) error {
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

func (u *updateHook) version(value interface{}) error {
	v, err := strconv.Atoi(value.(string))
	if err != nil {
		return err
	}
	ver := PyVer(v)
	if ver != PY2 && ver != PY3 {
		return errors.New("invalid version value")
	}
	u.opt.Version = ver
	return nil
}

func (u *updateHook) single(value interface{}) error {
	u.opt.Single = ufc.IsTrue(value.(string))
	u.render = true
	return nil
}

func (u *updateHook) sourceDir(value interface{}) error {
	sd := value.(string)
	// 检测sd，避免安全风险
	if strings.HasPrefix(sd, "/") || strings.HasPrefix(sd, "..") {
		return errors.New("illegal sourcedir")
	}
	u.opt.SourceDir = sd
	return nil
}

func (u *updateHook) lang(value interface{}) error {
	u.opt.Lang = value.(string)
	u.render = true
	return nil
}

func (u *updateHook) requirement(value interface{}) error {
	req := value.(string)
	// 检测req，避免安全风险
	if strings.HasPrefix(req, "/") || strings.HasPrefix(req, "..") {
		return errors.New("illegal requirement")
	}
	u.opt.Requirement = req
	return nil
}

func (u *updateHook) install(value interface{}) error {
	u.opt.Install = ufc.IsTrue(value.(string))
	return nil
}

func (u *updateHook) index(value interface{}) error {
	u.opt.Index = value.(string)
	return nil
}

func (u *updateHook) showNav(value interface{}) error {
	u.opt.ShowNav = ufc.IsTrue(value.(string))
	return nil
}

func (u *updateHook) hideGit(value interface{}) error {
	u.opt.HideGit = ufc.IsTrue(value.(string))
	return nil
}

func (u *updateHook) secret(value interface{}) error {
	u.opt.Secret = value.(string)
	return nil
}

func (u *updateHook) customDomain(value interface{}) error {
	dn := strings.ToLower(value.(string))

	// 清除自定义域名
	if ufc.IsFalse(dn) {
		odn := u.opt.CustomDomain
		if u.pm.HasCustomDomain(odn) {
			_, err := u.pm.db.SRem(GBDK, odn)
			if err != nil {
				return err
			}
		}
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

	_, err := u.pm.db.SAdd(GBDK, dn)
	if err != nil {
		return err
	}

	u.opt.CustomDomain = dn
	u.render = true
	return nil
}

func (u *updateHook) builder(value interface{}) error {
	u.opt.Builder = value.(BuilderType)
	return nil
}

func (u *updateHook) beforeHook(value interface{}) error {
	u.opt.BeforeHook = value.(string)
	return nil
}

func (u *updateHook) afterHook(value interface{}) error {
	u.opt.AfterHook = value.(string)
	return nil
}

func (u *updateHook) ssl(value interface{}) error {
	v := value.(string)

	// 取消自定义域名SSL
	if ufc.IsFalse(v) {
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
	if !ufc.IsFile(pub) || !ufc.IsFile(pri) {
		return errors.New("not found ssl file")
	}
	u.opt.SSL = true
	u.opt.SSLPublic = pub
	u.opt.SSLPrivate = pri
	u.render = true
	return nil
}
