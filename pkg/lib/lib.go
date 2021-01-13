// 对项目管理的封装（操作内嵌数据库）

package lib

import (
	"errors"
	"reflect"
	"strings"

	"rtfd/pkg/conf"
	"rtfd/pkg/util"

	"github.com/xujiajun/nutsdb"
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
	db   *nutsdb.DB
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

	opt := nutsdb.DefaultOptions
	opt.Dir = cfg.BaseDir()
	db, err := nutsdb.Open(opt)
	if err != nil {
		return
	}

	return &ProjectManager{path, cfg, db}, nil
}

// NewOption 创建一个通用的默认选项
func (pm *ProjectManager) NewOption(name, url, domain string) (opt Options, err error) {
	if domain != "" && !util.IsDomain(domain) {
		err = errors.New("invalid custom domain")
		return
	}
	// check has custom domain

	isPublic := false
	typ, err := util.CheckGitURL(url)
	if err != nil {
		return
	}
	if typ == "public" {
		isPublic = true
	}

	if strings.HasSuffix(url, ".git") {
		url = strings.TrimRight(url, ".git")
	}
	util.GitServiceProvider(url)

	dn := pm.cfg.GetKey("nginx", "dn")
	if dn == "" {
		err = errors.New("invalid nginx dn")
		return
	}
	return Options{
		Name: name, URL: url, Latest: "master", Version: PY3,
		SourceDir: "docs", Lang: "en", ShowNav: true, HideGit: false,
		DefaultDomain: name + "." + dn, Builder: "html", IsPublic: isPublic,
	}, nil
}

// SetOption 按照 Options 参数更新key
func (pm *ProjectManager) SetOption(opt *Options, key string, value interface{}) {
	p := reflect.ValueOf(&opt)
	f := p.Elem().FieldByName(key)
	switch key {
	case "Single", "Install", "ShowNav", "HideGit", "SSL", "IsPublic":
		f.SetBool(value.(bool))
	case "Version":
		f.SetUint(value.(uint64))
	default:
		f.SetString(value.(string))
	}
}

// Create 新建一个文档项目
func (pm *ProjectManager) Create(name string, opt Options) {
	//
}
