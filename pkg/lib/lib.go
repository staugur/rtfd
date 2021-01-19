// 对项目管理的封装（操作内嵌数据库）

package lib

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"tcw.im/rtfd/pkg/conf"
	"tcw.im/rtfd/pkg/db"
	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"

	"tcw.im/ufc"
)

type (
	// PyVer Python版本
	PyVer uint8
	// BuilderType 构建器类型
	BuilderType string
)

const (
	// PY2 is Python 2.x
	PY2 PyVer = 2
	// PY3 is Python 3.x
	PY3 PyVer = 3

	// HTMLBuilder HTML构建器
	HTMLBuilder BuilderType = "html"
	// DirHTMLBuilder 目录式HTML构建器
	DirHTMLBuilder BuilderType = "dirhtml"
	// SingleHTMLBuilder 单页HTML构建器
	SingleHTMLBuilder BuilderType = "singlehtml"
)

type (
	// Path 文件或目录路径
	Path = string
	// URL 包含协议头的地址
	URL = string
)

// Options 每个文档项目的配置项
type Options struct {
	// 项目在数据库中唯一标识名
	Name string
	// git地址，可以是包含用户名密码的私有仓库
	URL URL
	// 默认显示的分支
	Latest string
	// 使用的python版本，2或3
	Version PyVer
	// 是否单一版本
	Single bool
	// 文档源文件路径
	SourceDir Path
	// 文档语言，以半角逗号分隔多种语言
	Lang string
	// 依赖包文件
	Requirement Path
	// 是否安装项目
	Install bool
	// pypi仓库
	Index URL
	// 是否显示导航
	ShowNav bool
	// 隐藏git
	HideGit bool
	// webhook secret
	Secret string
	// 默认域名
	DefaultDomain string
	// 自定义域名
	CustomDomain string
	// 自定义域名开启HTTPS
	SSL bool
	// ssl公钥
	SSLPublic Path
	// ssl私钥
	SSLPrivate Path
	// Sphinx构建器，支持html、dirhtml、singlehtml
	Builder BuilderType
	// git服务提供商
	GSP string
	// 是否为公开仓库
	IsPublic bool
}

// ProjectManager 项目管理器
type ProjectManager struct {
	path Path
	cfg  *conf.Config
	db   *db.DB
}

func s2b(s string) []byte {
	return []byte(s)
}

// New 新建项目管理器示例，path是rtfd配置文件
func New(path string) (pm *ProjectManager, err error) {
	if !ufc.IsFile(path) {
		return nil, errors.New("not found config path")
	}
	cfg, err := conf.New(path)
	if err != nil {
		return
	}

	conn, err := db.New(path)
	if err != nil {
		return
	}

	return &ProjectManager{path, cfg, conn}, nil
}

// Close 关闭DB连接
func (pm *ProjectManager) Close() error {
	err := pm.db.Close()
	if err != nil {
		return err
	}
	return nil
}

// HasName 是否存在名为 name 的文档项目
func (pm *ProjectManager) HasName(name string) bool {
	return pm.db.SIsMember(vars.GBName, vars.GBPK, s2b(name))
}

// HasCustomDomain 判断是否已有自定义域名
func (pm *ProjectManager) HasCustomDomain(domain string) bool {
	return pm.db.SIsMember(vars.GBName, vars.GBDK, s2b(domain))
}

// GetName 查询名为 name 的文档项目数据
func (pm *ProjectManager) GetName(name string) (value []byte, err error) {
	value, err = pm.db.Get(name, vars.BCK)
	if err != nil {
		return
	}
	return value, nil
}

// GenerateOption 创建一个通用的默认选项（不作参数的系统级别检测）
func (pm *ProjectManager) GenerateOption(name, url string) (opt Options, err error) {
	name = strings.ToLower(name)
	if !util.IsProjectName(name) {
		err = errors.New("invalid name")
		return
	}

	typ, err := util.CheckGitURL(url)
	if err != nil {
		return
	}

	isPublic := false
	if typ == "public" {
		isPublic = true
	}

	if strings.HasSuffix(url, ".git") {
		url = strings.TrimRight(url, ".git")
	}
	gsp, err := util.GitServiceProvider(url)
	if err != nil {
		return
	}

	dn := pm.cfg.GetKey("nginx", "dn")
	if dn == "" {
		err = errors.New("invalid nginx dn")
		return
	}
	return Options{
		Name: name, URL: url, Latest: "master", Version: PY3, GSP: gsp,
		SourceDir: "docs", Lang: "en", ShowNav: true, HideGit: false,
		DefaultDomain: name + "." + dn, Builder: "html", IsPublic: isPublic,
	}, nil
}

// SetOption 按照 Options 参数更新key
func (pm *ProjectManager) SetOption(opt *Options, key string, value interface{}) {
	p := reflect.ValueOf(opt)
	f := p.Elem().FieldByName(key)
	switch key {
	case "Single", "Install", "ShowNav", "HideGit", "SSL", "IsPublic":
		f.SetBool(value.(bool))
	case "Version":
		f.SetUint(uint64(value.(uint8)))
	default:
		f.SetString(value.(string))
	}
}

// Create 新建一个文档项目
func (pm *ProjectManager) Create(name string, opt Options) error {
	name = strings.ToLower(name)
	unallow := pm.cfg.GetKey(vars.DFT, "unallowed_name")
	if ufc.StrInSlice(name, strings.Split(unallow, ",")) {
		return errors.New("not allowed name")
	}
	if pm.HasName(name) {
		return errors.New("this project name already exists")
	}
	domain := opt.CustomDomain
	if domain != "" {
		if !util.IsDomain(domain) {
			return errors.New("invalid custom domain")
		}
		if pm.HasCustomDomain(domain) {
			return errors.New("this domain name already exists")
		}
	}
	if opt.SSLPublic != "" && opt.SSLPrivate != "" {
		if !ufc.IsFile(opt.SSLPublic) || !ufc.IsFile(opt.SSLPrivate) {
			return errors.New("not found ssl file")
		}
		opt.SSL = true
	}

	val, err := json.Marshal(opt)
	if err != nil {
		return err
	}
	// 使管道批量事务提交
	tc, err := pm.db.Pipeline()
	if err != nil {
		return err
	}

	// 新增文档项目成功，以下分别是：添加到全局项目集合中、写入配置、添加到全局自定义域名键中
	tc.SAdd(vars.GBName, vars.GBPK, s2b(name))
	tc.Set(name, vars.BCK, val)
	if domain != "" {
		tc.SAdd(vars.GBName, vars.GBDK, s2b(domain))
	}

	err = tc.Execute()
	if err != nil {
		return err
	}

	return nil
}
