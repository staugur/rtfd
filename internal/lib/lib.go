// 对项目管理的封装（操作内嵌数据库）

package lib

import (
	"errors"

	"rtfd/internal/conf"

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
	name string
	// git地址，可以是包含用户名密码的私有仓库
	url URL
	// 默认显示的分支
	latest string
	// 使用的python版本，2或3
	version PyVer
	// 是否单一版本
	single bool
	// 文档源文件路径
	sourcedir Path
	// 文档语言，以半角逗号分隔多种语言
	languages string
	// 依赖包文件
	requirements Path
	// 是否安装项目
	install bool
	// pypi仓库
	index URL
	// 是否显示导航
	nav bool
	// webhook secret
	secret string
	// 默认域名
	defaultDomain string
	// 自定义域名
	customDomain string
	// 自定义域名开启HTTPS
	ssl bool
	// ssl公钥
	sslPublic Path
	// ssl私钥
	sslPrivate Path
	// Sphinx构建器，支持html、dirhtml、singlehtml
	builder BuilderType
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

	baseDir := cfg.GetKey("default", "base_dir")
	opt := nutsdb.DefaultOptions
	opt.Dir = baseDir
	db, err := nutsdb.Open(opt)
	if err != nil {
		return
	}

	return &ProjectManager{path, cfg, db}, nil
}

// NewOption 创建一个通用的默认选项
func (pm *ProjectManager) NewOption(name, url string) (opt Options, err error) {
	dn := pm.cfg.GetKey("nginx", "dn")
	if dn == "" {
		return opt, errors.New("invalid nginx dn")
	}
	return Options{
		name: name, url: url, latest: "master", version: PY3,
		sourcedir: "docs", languages: "en", nav: true,
		defaultDomain: name + "." + dn, builder: "html",
	}, nil
}

// Create 新建一个文档项目
func (pm *ProjectManager) Create(name string, opt Options) {
	//
}
