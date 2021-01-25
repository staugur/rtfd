// 对项目管理的封装（操作内嵌数据库）

package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	// Sender 发起构建来源类型
	Sender string
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

	// APISender 从API接口发起构建
	APISender Sender = "api"
	// CLISender 从命令行发起构建
	CLISender Sender = "cli"
	// WebhookSender 从git webhook发起自动构建
	WebhookSender Sender = "webhook"
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
	// 自定义域名开启HTTPS（非手动）
	SSL bool
	// 自定义域名的ssl公钥
	SSLPublic Path
	// 自定义域名的ssl私钥
	SSLPrivate Path
	// Sphinx构建器，支持html、dirhtml、singlehtml
	Builder BuilderType
	// git服务提供商
	GSP string
	// 是否为公开仓库（type）
	IsPublic bool
}

// Result 构建结果
type Result struct {
	// 触发构建的分支或标签
	Branch string
	// 构建结果 passing表示true 其他表示false
	Status bool
	Sender Sender
	// 构建完成时间（结束时）
	Btime string
	// 构建总花费时间（单位秒）
	Usedtime int
}

// OptionsWithResult 嵌套了 Options 和 Result 两种结构
type OptionsWithResult struct {
	Options
	Builder []Result
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
	name = strings.ToLower(name)
	return pm.db.SIsMember(vars.GBName, vars.GBPK, s2b(name))
}

// HasCustomDomain 判断是否已有自定义域名
func (pm *ProjectManager) HasCustomDomain(domain string) bool {
	return pm.db.SIsMember(vars.GBName, vars.GBDK, s2b(domain))
}

// GetSourceName 查询名为 name 的文档项目数据存储原数据（不经过解析）
func (pm *ProjectManager) GetSourceName(name string) (value []byte, err error) {
	name = strings.ToLower(name)
	value, err = pm.db.Get(name, vars.BCK)
	if err != nil {
		return
	}
	return value, nil
}

// GetName 查询名为 name 的文档项目数据（解析后）
func (pm *ProjectManager) GetName(name string) (opt Options, err error) {
	value, err := pm.GetSourceName(name)
	if err != nil {
		return
	}
	err = json.Unmarshal(value, &opt)
	if err != nil {
		return
	}
	return opt, nil
}

// GetNameWithBuilder 获取文档项目配置及其构建集详细数据
func (pm *ProjectManager) GetNameWithBuilder(name string) (ropt OptionsWithResult, err error) {
	opt, err := pm.GetName(name)
	if err != nil {
		return
	}
	members, err := pm.ListFullBuilder(name)
	if err != nil {
		return
	}
	return OptionsWithResult{Options: opt, Builder: members}, nil
}

// GetNameOption 获取文档项目某项配置值
func (pm *ProjectManager) GetNameOption(name, key string) (val string, err error) {
	opt, err := pm.GetName(name)
	if err != nil {
		return
	}
	p := reflect.ValueOf(&opt)
	f := p.Elem().FieldByName(key)

	switch key {
	case "Single", "Install", "ShowNav", "HideGit", "SSL", "IsPublic":
		if f.Bool() {
			return "true", nil
		} else {
			return "false", nil
		}
	case "Version":
		return fmt.Sprint(f.Uint()), nil
	default:
		return f.String(), nil
	}
}

// ListFullProject 获取所有项目及其配置选项
func (pm *ProjectManager) ListFullProject() (members []Options, err error) {
	list, err := pm.ListProject()
	if err != nil {
		return
	}
	members = make([]Options, len(list))
	for i, b := range list {
		val, e := pm.GetName(string(b))
		if e != nil {
			err = e
			return
		}
		members[i] = val
	}
	return members, nil
}

// ListProject 获取所有项目
func (pm *ProjectManager) ListProject() (members [][]byte, err error) {
	return pm.db.SMembers(vars.GBName, vars.GBPK)
}

// ListBuilder 获取所有构建集
func (pm *ProjectManager) ListBuilder(name string) (members [][]byte, err error) {
	return pm.db.SMembers(name, vars.BRLK)
}

// ListFullBuilder 获取构建集及其详情
func (pm *ProjectManager) ListFullBuilder(name string) (members []Result, err error) {
	list, err := pm.ListBuilder(name)
	if err != nil {
		return
	}
	members = make([]Result, len(list))
	for i, b := range list {
		val, e := pm.db.Get(vars.BRK(name), b)
		if e != nil {
			err = e
			return
		}
		var rst Result
		e = json.Unmarshal(val, &rst)
		if e != nil {
			err = e
			return
		}
		members[i] = rst
	}
	return members, nil
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

// Create 新建一个文档项目（唯一入口，必须通过GenerateOption方法生成选项）
func (pm *ProjectManager) Create(name string, opt Options) error {
	name = strings.ToLower(name)
	unallow := pm.cfg.GetKey(vars.DFT, "unallowed_name")
	if name == "www" || ufc.StrInSlice(name, strings.Split(unallow, ",")) {
		return errors.New("not allowed name")
	}
	if pm.HasName(name) {
		return errors.New("this project name already exists")
	}
	//校验必选项
	if opt.URL == "" || opt.DefaultDomain == "" || opt.Latest == "" || opt.Lang == "" || opt.Builder == "" || (opt.Version != 2 && opt.Version != 3) || opt.SourceDir == "" {
		return errors.New("required fields are missing")
	}
	domain := opt.CustomDomain
	if domain != "" {
		if !util.IsDomain(domain) {
			return errors.New("invalid custom domain")
		}
		if pm.HasCustomDomain(domain) {
			return errors.New("this domain name already exists")
		}
		domain = strings.ToLower(domain)
		opt.CustomDomain = domain
	}
	if opt.SSLPublic != "" && opt.SSLPrivate != "" {
		if !ufc.IsFile(opt.SSLPublic) || !ufc.IsFile(opt.SSLPrivate) {
			return errors.New("not found ssl file")
		}
		opt.SSL = true
	} else {
		opt.SSL = false
	}

	val, err := json.Marshal(opt)
	if err != nil {
		return err
	}

	// 使用管道批量事务提交
	tc, err := pm.db.Pipeline()
	if err != nil {
		return err
	}
	// 新增文档项目成功，以下分别是：添加到全局项目集合中、写入配置、添加到全局自定义域名键中
	tc.SAdd(vars.GBName, vars.GBPK, s2b(name))
	tc.Set(name, vars.BCK, val) //配置写入的是JSON格式
	if domain != "" {
		tc.SAdd(vars.GBName, vars.GBDK, s2b(domain))
	}
	err = tc.Execute()
	if err != nil {
		return err
	}

	// 生成nginx配置
	err = pm.renderNginx(name)
	if err != nil {
		return err
	}
	return nil
}

func (pm *ProjectManager) renderNginx(name string) error {
	name = strings.ToLower(name)
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}
	if opt.Lang == "" {
		return errors.New("empty language cannot render nginx")
	}
	basedir := pm.cfg.BaseDir()
	DocsDir := filepath.Join(basedir, "docs")
	NginxDir := filepath.Join(basedir, "nginx")
	if !ufc.IsDir(DocsDir) {
		err := ufc.CreateDir(DocsDir)
		if err != nil {
			return err
		}
	}
	if !ufc.IsDir(NginxDir) {
		err := ufc.CreateDir(NginxDir)
		if err != nil {
			return err
		}
	}
	// 渲染默认域名的nginx配置
	dftLang := strings.Split(opt.Lang, ",")[0]
	dftNgxFile := filepath.Join(NginxDir, fmt.Sprintf("%s.conf", name))
	cstNgxFile := filepath.Join(NginxDir, fmt.Sprintf("%s.ext.conf", name))
	dftSSLCrt := pm.cfg.GetKey("nginx", "ssl_crt")
	dftSSLKey := pm.cfg.GetKey("nginx", "ssl_key")
	ngxopt := &nginxOptions{
		Name: name, Lang: dftLang, Domain: opt.DefaultDomain, DocsDir: DocsDir,
		Single: opt.Single, SSLCrt: dftSSLCrt, SSLKey: dftSSLKey,
	}
	dftConf, err := ngxopt.render()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dftNgxFile, []byte(dftConf), 0644)
	if err != nil {
		return err
	}

	// 渲染自定义域名的nginx配置
	if util.IsDomain(opt.CustomDomain) {
		ngxopt.Domain = opt.CustomDomain
		ngxopt.SSLCrt = opt.SSLPublic
		ngxopt.SSLKey = opt.SSLPrivate
		cstConf, err := ngxopt.render()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(cstNgxFile, []byte(cstConf), 0644)
		if err != nil {
			return err
		}
	} else {
		if ufc.IsFile(cstNgxFile) {
			os.Remove(cstNgxFile)
		}
	}

	err = pm.reloadNginx()
	if err != nil {
		return err
	}
	return nil
}

func (pm *ProjectManager) reloadNginx() error {
	cmd := pm.cfg.GetKey("nginx", "exec")
	sudo := ufc.IsTrue(pm.cfg.GetKey("nginx", "sudo"))
	var (
		name       string
		testArgs   []string
		reloadArgs []string
	)
	if sudo == true {
		name = "sudo"
		testArgs = []string{cmd, "-t"}
		reloadArgs = []string{cmd, "-s", "reload"}
	} else {
		name = cmd
		testArgs = []string{"-t"}
		reloadArgs = []string{"-s", "reload"}
	}

	exitCode, _, err := util.RunCmd(name, testArgs...)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return errors.New("nginx test configuration failed")
	}
	exitCode, _, err = util.RunCmd(name, reloadArgs...)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return errors.New("nginx reload configuration failed")
	}
	return nil
}

// BuildRecord 记录构建结果
func (pm *ProjectManager) BuildRecord(name string, branchOrTag string, result interface{}) error {
	// 使用管道批量事务提交
	tc, err := pm.db.Pipeline()
	if err != nil {
		return err
	}
	// 新增文档项目成功，以下分别是：添加到构建索引中和构建结果记录中
	bot := s2b(branchOrTag)
	rst, err := json.Marshal(result)
	if err != nil {
		return err
	}
	tc.SAdd(name, vars.BRLK, bot)
	tc.Set(vars.BRK(name), bot, rst)

	err = tc.Execute()
	if err != nil {
		return err
	}
	return nil
}
