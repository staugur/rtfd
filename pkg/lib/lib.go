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

// 对项目管理的封装（操作数据库）

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
	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"

	"github.com/gomodule/redigo/redis"
	homedir "github.com/mitchellh/go-homedir"
	"tcw.im/gtc"
	db "tcw.im/gtc/redigo"
)

type (
	// PyVer Python版本
	PyVer uint8
	// BuilderType 构建器类型
	BuilderType string
	// Path 文件或目录路径
	Path = string
	// URL 包含协议头的地址
	URL = string
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
	// 依赖包文件，以半角逗号分隔多个文件
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
	// 自定义域名开启HTTPS（自动填充）
	SSL bool
	// 自定义域名的ssl公钥
	SSLPublic Path
	// 自定义域名的ssl私钥
	SSLPrivate Path
	// Sphinx构建器，支持html、dirhtml、singlehtml
	Builder BuilderType
	// git服务提供商（自动填充）
	GSP string
	// 是否为公开仓库（原type，自动填充）
	IsPublic bool
	// 构建前的钩子命令
	BeforeHook string
	// 构建成功后的钩子命令
	AfterHook string
	// 额外配置数据
	Meta map[string]string
}

// Result 构建结果
type Result struct {
	// 触发构建的分支或标签
	Branch string
	// 构建结果 passing表示true 其他表示false
	Status bool
	// 发起构建的来源
	Sender vars.Sender
	// 构建完成时间（结束时）
	Btime string
	// 构建总花费时间（单位秒）
	Usedtime int
}

// OptionsWithResult 嵌套了 Options 和 Result 两种结构
type OptionsWithResult struct {
	Options
	Buildset []Result
}

// ProjectManager 项目管理器
type ProjectManager struct {
	path Path
	cfg  *conf.Config
	db   *db.DB
}

// 数据 Key 命名：
// 1. 项目名称写入 GBPK，自定义域名写入 GBDK，类型均为set
// 2. 项目配置写入 BCK，类型为string，内容为json
// 3. 项目构建结果写入 BRK，类型为hash，键为branch/tag
var (
	// GBPK 文档项目名称集合，set类型
	GBPK = "projects"
	// GBDK 所有自定义的域名集合，set类型
	GBDK = "domains"
)

// BCK 生成文档项目配置Key，string类型
func BCK(projectName string) string {
	projectName = strings.ToLower(projectName)
	return "project:" + projectName
}

// BRK 生成构建结果Key，hash类型
func BRK(projectName string) string {
	projectName = strings.ToLower(projectName)
	return "builder:" + projectName
}

// OptionKeyMap 转换 Options 结构体字段名大小写
func OptionKeyMap(key string) string {
	switch strings.ToLower(key) {
	case "url":
		return "URL"
	case "sourcedir":
		return "SourceDir"
	case "shownav":
		return "ShowNav"
	case "hidegit":
		return "HideGit"
	case "defaultdomain":
		return "DefaultDomain"
	case "customdomain":
		return "CustomDomain"
	case "ssl":
		return "SSL"
	case "sslpublic":
		return "SSLPublic"
	case "sslprivate":
		return "SSLPrivate"
	case "gsp":
		return "GSP"
	case "ispublic":
		return "IsPublic"
	case "beforehook":
		return "BeforeHook"
	case "afterhook":
		return "AfterHook"
	default:
		return strings.Title(strings.ToLower(key))
	}
}

// New 新建项目管理器示例，path是rtfd配置文件
func New(path string) (pm *ProjectManager, err error) {
	if strings.HasPrefix(path, "~") {
		path, err = homedir.Expand(path)
		if err != nil {
			return
		}
	}
	if !gtc.IsFile(path) {
		return nil, errors.New("not found config path")
	}
	cfg, err := conf.New(path)
	if err != nil {
		return
	}

	conn, err := db.New(cfg.GetKey(vars.DFT, "redis"))
	if err != nil {
		return
	}
	conn.Prefix = "rtfd:"

	return &ProjectManager{path, cfg, conn}, nil
}

// CFG 即config实例
func (pm *ProjectManager) CFG() *conf.Config {
	return pm.cfg
}

// DB 即db实例
func (pm *ProjectManager) DB() *db.DB {
	return pm.db
}

// HasName 是否存在名为 name 的文档项目
func (pm *ProjectManager) HasName(name string) bool {
	name = strings.ToLower(name)
	has, err := pm.db.SIsMember(GBPK, name)
	if err != nil {
		panic(err)
	}
	return has
}

// HasCustomDomain 判断是否已有自定义域名
func (pm *ProjectManager) HasCustomDomain(domain string) bool {
	has, err := pm.db.SIsMember(GBDK, domain)
	if err != nil {
		panic(err)
	}
	return has
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

	url = strings.TrimSuffix(url, ".git")
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
		Name: name, URL: url, Version: PY3, Latest: pm.cfg.DefaultBranch(),
		SourceDir: "docs", Lang: "en", ShowNav: true, HideGit: false, GSP: gsp,
		DefaultDomain: name + "." + dn, Builder: HTMLBuilder, IsPublic: isPublic,
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
	if name == "www" || gtc.StrInSlice(name, strings.Split(unallow, ",")) {
		return errors.New("not allowed name")
	}
	if pm.HasName(name) {
		return errors.New("this project name already exists")
	}
	//校验必选项
	if opt.URL == "" || opt.DefaultDomain == "" || opt.Latest == "" || opt.Lang == "" ||
		(opt.Builder != HTMLBuilder && opt.Builder != DirHTMLBuilder && opt.Builder != SingleHTMLBuilder) ||
		(opt.Version != PY2 && opt.Version != PY3) || opt.SourceDir == "" {
		return errors.New("required fields are missing")
	}
	domain := opt.CustomDomain
	if domain != "" {
		domain = strings.ToLower(domain)
		if !util.IsDomain(domain) {
			return errors.New("invalid custom domain")
		}
		if pm.HasCustomDomain(domain) {
			return errors.New("this domain name already exists")
		}
		opt.CustomDomain = domain
	}
	if opt.SSLPublic != "" && opt.SSLPrivate != "" {
		if !gtc.IsFile(opt.SSLPublic) || !gtc.IsFile(opt.SSLPrivate) {
			return errors.New("not found ssl file")
		}
		opt.SSL = true
	} else {
		opt.SSL = false
	}

	// 生成nginx配置并重载
	err := pm.renderNginx(&opt)
	if err != nil {
		return err
	}

	// 基本数据生成完毕，写入数据库
	val, err := json.Marshal(opt)
	if err != nil {
		return err
	}

	// 使用管道批量事务提交
	tc := pm.db.Pipeline()
	// 新增文档项目成功，以下分别是：添加到全局项目集合中、写入配置、添加到全局自定义域名键中
	tc.SAdd(GBPK, name)
	tc.Set(BCK(name), string(val)) //配置写入的是JSON格式
	if domain != "" {
		tc.SAdd(GBDK, domain)
	}
	_, err = tc.Execute()
	if err != nil {
		return err
	}

	// 已创建项目后的处理，无所谓成功
	// 创建 GHApp 实例，使用接口获取安装id再换取token
	if opt.GSP == vars.GSPGitHub {
		gh, err := NewGHApp(pm)
		if err == nil {
			err = gh.cliSetWebhook(&opt)
			if err != nil {
				fmt.Printf("failed to automatically create webhook: %s\n", err)
			}
		}
	}
	return nil
}

// GetSourceName 查询名为 name 的文档项目数据存储原数据（不经过解析，即JSON格式）
func (pm *ProjectManager) GetSourceName(name string) (value []byte, err error) {
	name = strings.ToLower(name)
	r, err := pm.db.Get(BCK(name))
	if err != nil {
		return
	}
	return []byte(r), nil
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

// GetNameWithBuildset 获取文档项目配置及其构建集详细数据
func (pm *ProjectManager) GetNameWithBuildset(name string) (ropt OptionsWithResult, err error) {
	opt, err := pm.GetName(name)
	if err != nil {
		return
	}
	members, err := pm.ListBuildset(name)
	if err != nil {
		return
	}
	return OptionsWithResult{Options: opt, Buildset: members}, nil
}

// GetNameOption 获取文档项目某项配置值
func (pm *ProjectManager) GetNameOption(name, key string) (val string, err error) {
	key = OptionKeyMap(key)

	opt, err := pm.GetName(name)
	if err != nil {
		return
	}

	if strings.HasPrefix(key, "Meta") {
		ks := strings.Split(key, "@")
		if len(ks) < 2 {
			return "", errors.New("invalid meta key")
		}
		field := strings.ToLower(ks[1])
		return opt.GetMeta(field), nil
	}

	p := reflect.ValueOf(&opt)
	f := p.Elem().FieldByName(key)
	switch key {
	case "Single", "Install", "ShowNav", "HideGit", "SSL", "IsPublic":
		if f.Bool() {
			return "true", nil
		}
		return "false", nil
	case "Version":
		return fmt.Sprint(f.Uint()), nil
	default:
		if f.IsValid() {
			return f.String(), nil
		}
		return "", nil
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
		val, e := pm.GetName(b)
		if e != nil {
			err = e
			return
		}
		members[i] = val
	}
	return members, nil
}

// ListProject 获取所有项目
func (pm *ProjectManager) ListProject() (members []string, err error) {
	return pm.db.SMembers(GBPK)
}

// ListBuildset 获取所有构建集
func (pm *ProjectManager) ListBuildset(name string) (builders []Result, err error) {
	hash, err := pm.db.HGetAll(BRK(name))
	if err != nil {
		return
	}
	builders = make([]Result, 0, len(hash))
	for _, val := range hash {
		var rst Result
		e := json.Unmarshal([]byte(val), &rst)
		if e != nil {
			err = e
			return
		}
		builders = append(builders, rst)
	}
	return builders, nil
}

// GetBuildset 获取某个构建结果
func (pm *ProjectManager) GetBuildset(name, branch string) (builder Result, err error) {
	val, err := pm.db.HGet(BRK(name), branch)
	if err != nil {
		if err == redis.ErrNil {
			err = errors.New("not found branch")
		}
		return
	}

	var rst Result
	err = json.Unmarshal([]byte(val), &rst)
	if err != nil {
		return
	}
	return rst, nil
}

func (pm *ProjectManager) renderNginx(opt *Options) error {
	name := opt.Name
	if opt.Lang == "" {
		return errors.New("empty language cannot render nginx")
	}
	basedir := pm.cfg.BaseDir()
	DocsDir := filepath.Join(basedir, "docs")
	NginxDir := filepath.Join(basedir, "nginx")
	if !gtc.IsDir(basedir) {
		err := gtc.CreateDir(basedir)
		if err != nil {
			return err
		}
	}
	if !gtc.IsDir(DocsDir) {
		err := gtc.CreateDir(DocsDir)
		if err != nil {
			return err
		}
	}
	if !gtc.IsDir(NginxDir) {
		err := gtc.CreateDir(NginxDir)
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
		if gtc.IsFile(cstNgxFile) {
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
	sudo := gtc.IsTrue(pm.cfg.GetKey("nginx", "sudo"))
	var (
		name       string
		testArgs   []string
		reloadArgs []string
	)
	if sudo {
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
	if exitCode != 0 || err != nil {
		return errors.New("nginx reload service failed")
	}
	return nil
}

// BuildRecord 记录构建结果
func (pm *ProjectManager) BuildRecord(name string, branchOrTag string, result Result) error {
	name = strings.ToLower(name)
	rst, err := json.Marshal(result)
	if err != nil {
		return err
	}

	_, err = pm.db.HSet(BRK(name), branchOrTag, string(rst))
	if err != nil {
		return err
	}

	return nil
}

// Remove 删除一个文档项目及其数据
func (pm *ProjectManager) Remove(name string) error {
	name = strings.ToLower(name)
	if !pm.HasName(name) {
		return errors.New("not found project")
	}
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}

	basedir := pm.cfg.BaseDir()
	DocsDir := filepath.Join(basedir, "docs", name)
	NginxDir := filepath.Join(basedir, "nginx")
	dftNgxFile := filepath.Join(NginxDir, fmt.Sprintf("%s.conf", name))
	cstNgxFile := filepath.Join(NginxDir, fmt.Sprintf("%s.ext.conf", name))

	if gtc.IsDir(DocsDir) {
		err = os.RemoveAll(DocsDir)
		if err != nil {
			return err
		}
	}
	if gtc.IsFile(dftNgxFile) || gtc.IsFile(cstNgxFile) {
		os.Remove(dftNgxFile)
		os.Remove(cstNgxFile)
		err = pm.reloadNginx()
		if err != nil {
			fmt.Printf("failed to automatically remove webhook: %s\n", err)
		}
	}

	// try remove webhook with github apps
	if opt.GSP == vars.GSPGitHub {
		gh, err := NewGHApp(pm)
		if err == nil {
			err = gh.cliRemoveWebhook(opt)
			if err != nil {
				fmt.Printf("remove webhook fail: %s\n", err)
			}
		}
	}

	tc := pm.db.Pipeline()
	tc.SRem(GBPK, name)
	domain := opt.CustomDomain
	if domain != "" {
		tc.SRem(GBDK, domain)
	}
	tc.Del(BCK(name))
	tc.Del(BRK(name))
	_, err = tc.Execute()
	if err != nil {
		return err
	}
	return nil
}

// Update 更新文档项目配置
func (pm *ProjectManager) Update(opt *Options, rule map[string]interface{}) (ok []string, fail []string, err error) {
	name := opt.Name
	if !pm.HasName(name) {
		err = errors.New("not found project")
		return
	}

	uh := &updateHook{pm: pm, opt: opt}
	for field, value := range rule {
		fn, e := uh.handle(field)
		if e != nil {
			fail = append(fail, fmt.Sprintf("%s:%s", field, e.Error()))
			continue
		}
		e = fn(value)
		if e != nil {
			fail = append(fail, fmt.Sprintf("%s:%s", field, e.Error()))
			continue
		}
		ok = append(ok, field)
	}

	val, err := json.Marshal(opt)
	if err != nil {
		return
	}
	_, err = pm.db.Set(BCK(name), string(val)) //配置写入的是JSON格式
	if err != nil {
		return
	}

	if uh.render {
		err = pm.renderNginx(opt)
		if err != nil {
			return
		}
	}
	return
}

// GetMeta 专门读取 Options 结构体 Meta 字段的值
func (opt Options) GetMeta(key string) string {
	val := opt.Meta[key]
	return val
}

// MustMeta 专门读取 Options 结构体 Meta 字段的值，可设置默认值
func (opt Options) MustMeta(key, defaultValue string) string {
	val := opt.Meta[key]
	if val == "" {
		return defaultValue
	}
	return val
}

// UpdateMeta 专门更新 Meta 字段 （如果key以下划线开头表示系统数据）
func (opt *Options) UpdateMeta(key, val string) error {
	if key == "" || val == "" {
		return errors.New("invalid meta key or value")
	}
	if !util.LLPat.MatchString(key) {
		return errors.New("illegal format key")
	}
	meta := opt.Meta
	if meta == nil {
		meta = make(map[string]string)
	}
	if val == vars.ResetEmpty {
		val = ""
	}
	meta[key] = val
	opt.Meta = meta
	return nil
}

// Writeback 配置写入数据库
func (opt Options) Writeback(pm *ProjectManager) error {
	name := strings.ToLower(opt.Name)
	if !pm.HasName(name) {
		return errors.New("not found project")
	}
	val, err := json.Marshal(opt)
	if err != nil {
		return err
	}
	ok, err := pm.DB().Set(BCK(name), string(val))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("write fail")
	}
	return nil
}
