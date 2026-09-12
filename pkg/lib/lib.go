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
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"pkg/tcw.im/rtfd/pkg/conf"
	"pkg/tcw.im/rtfd/pkg/store"
	"pkg/tcw.im/rtfd/pkg/util"
	"pkg/tcw.im/rtfd/vars"

	"gorm.io/gorm"
	"pkg.tcw.im/gtc"
)

type (
	// PyVer Python版本，如3、3.10、3.12，需在配置文件py分区中定义
	PyVer string
	// BuilderType 构建器类型
	BuilderType string
	// Path 文件或目录路径
	Path = string
	// URL 包含协议头的地址
	URL = string
)

const (
	// PY3 is Python 3.x（默认版本，已移除Python 2.x支持）
	PY3 PyVer = vars.DefaultPy

	// HTMLBuilder HTML构建器
	HTMLBuilder BuilderType = "html"
	// DirHTMLBuilder 目录式HTML构建器
	DirHTMLBuilder BuilderType = "dirhtml"
	// SingleHTMLBuilder 单页HTML构建器
	SingleHTMLBuilder BuilderType = "singlehtml"
)

// UnmarshalJSON 解析Python版本，兼容历史以数字（2、3）存储的数据
func (p *PyVer) UnmarshalJSON(data []byte) error {
	val := strings.Trim(strings.TrimSpace(string(data)), `"`)
	// 已移除Python2支持，历史数据回退到默认版本
	if val == "" || val == "null" || val == "2" || strings.HasPrefix(val, "2.") {
		*p = PY3
		return nil
	}
	*p = PyVer(val)
	return nil
}

// Options 每个文档项目的配置项
type Options struct {
	// 项目在数据库中唯一标识名
	Name string
	// git地址，可以是包含用户名密码的私有仓库
	URL URL
	// 默认显示的分支
	Latest string
	// 使用的python版本，如3、3.10，需在配置文件py分区中定义
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
	db   *gorm.DB
}

// projectFromOptions 将项目配置转换为数据库模型
func projectFromOptions(opt Options) *store.Project {
	return &store.Project{
		Name:          strings.ToLower(opt.Name),
		URL:           string(opt.URL),
		Latest:        opt.Latest,
		Version:       string(opt.Version),
		Single:        opt.Single,
		SourceDir:     string(opt.SourceDir),
		Lang:          opt.Lang,
		Requirement:   string(opt.Requirement),
		Install:       opt.Install,
		Index:         string(opt.Index),
		ShowNav:       opt.ShowNav,
		HideGit:       opt.HideGit,
		Secret:        opt.Secret,
		DefaultDomain: opt.DefaultDomain,
		CustomDomain:  opt.CustomDomain,
		SSL:           opt.SSL,
		SSLPublic:     string(opt.SSLPublic),
		SSLPrivate:    string(opt.SSLPrivate),
		Builder:       string(opt.Builder),
		GSP:           opt.GSP,
		IsPublic:      opt.IsPublic,
		Meta:          opt.Meta,
	}
}

// optionsFromProject 将数据库模型转换为项目配置
func optionsFromProject(p *store.Project) Options {
	return Options{
		Name:          p.Name,
		URL:           URL(p.URL),
		Latest:        p.Latest,
		Version:       PyVer(p.Version),
		Single:        p.Single,
		SourceDir:     Path(p.SourceDir),
		Lang:          p.Lang,
		Requirement:   Path(p.Requirement),
		Install:       p.Install,
		Index:         URL(p.Index),
		ShowNav:       p.ShowNav,
		HideGit:       p.HideGit,
		Secret:        p.Secret,
		DefaultDomain: p.DefaultDomain,
		CustomDomain:  p.CustomDomain,
		SSL:           p.SSL,
		SSLPublic:     Path(p.SSLPublic),
		SSLPrivate:    Path(p.SSLPrivate),
		Builder:       BuilderType(p.Builder),
		GSP:           p.GSP,
		IsPublic:      p.IsPublic,
		Meta:          p.Meta,
	}
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
	default:
		return util.TitleCase(strings.ToLower(key))
	}
}

// New 新建项目管理器示例，path是rtfd配置文件
func New(path string) (pm *ProjectManager, err error) {
	path = util.ExpandPath(path)
	if !gtc.IsFile(path) {
		return nil, errors.New("not found config path")
	}
	cfg, err := conf.New(path)
	if err != nil {
		return
	}

	conn, err := store.Open(cfg.DatabaseType(), cfg.DatabaseDSN(), cfg.DatabaseDebug())
	if err != nil {
		return
	}

	return &ProjectManager{path, cfg, conn}, nil
}

// CFG 即config实例
func (pm *ProjectManager) CFG() *conf.Config {
	return pm.cfg
}

// DB 即数据库实例
func (pm *ProjectManager) DB() *gorm.DB {
	return pm.db
}

// Close 关闭数据库连接
func (pm *ProjectManager) Close() error {
	return store.Close(pm.db)
}

// HasName 是否存在名为 name 的文档项目
func (pm *ProjectManager) HasName(name string) bool {
	return pm.countProject(strings.ToLower(name)) > 0
}

// HasCustomDomain 判断是否已有自定义域名
func (pm *ProjectManager) HasCustomDomain(domain string) bool {
	if domain == "" {
		return false
	}
	var count int64
	pm.db.Model(&store.Project{}).
		Where("custom_domain = ?", strings.ToLower(domain)).Count(&count)
	return count > 0
}

// countProject 统计名称相同的项目数量
func (pm *ProjectManager) countProject(name string) int64 {
	var count int64
	pm.db.Model(&store.Project{}).Where("name = ?", name).Count(&count)
	return count
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
		Name: name, URL: url, Version: PyVer(pm.cfg.DefaultPyVersion()), Latest: pm.cfg.DefaultBranch(),
		SourceDir: "docs", Lang: "en", ShowNav: true, HideGit: false, GSP: gsp,
		DefaultDomain: name + "." + dn, Builder: HTMLBuilder, IsPublic: isPublic,
	}, nil
}

// SetOption 按照 Options 参数更新key
func (pm *ProjectManager) SetOption(opt *Options, key string, value any) {
	p := reflect.ValueOf(opt)
	f := p.Elem().FieldByName(key)
	switch key {
	case "Single", "Install", "ShowNav", "HideGit", "SSL", "IsPublic":
		f.SetBool(value.(bool))
	case "Version":
		f.SetString(fmt.Sprint(value))
	default:
		f.SetString(value.(string))
	}
}

// optionAlias 参数别名归一，将外部传入的简写字段转为规范字段名
func optionAlias(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "domain":
		return "customdomain"
	case "source":
		return "sourcedir"
	case "sslcrt":
		return "sslpublic"
	case "sslkey", "sslpri":
		return "sslprivate"
	default:
		return key
	}
}

// SetOptionValue 按 Options 字段的实际类型写入参数值（字符串、布尔自动转换），
// 供 API 等外部参数（值类型不确定）场景使用
func (pm *ProjectManager) SetOptionValue(opt *Options, key string, value any) error {
	field := OptionKeyMap(optionAlias(key))
	f := reflect.ValueOf(opt).Elem().FieldByName(field)
	if !f.IsValid() {
		return fmt.Errorf("invalid field: %s", key)
	}
	switch f.Kind() {
	case reflect.Bool:
		f.SetBool(util.ParamBool(value))
	case reflect.String:
		f.SetString(util.ParamString(value))
	default:
		return fmt.Errorf("unsupported field: %s", key)
	}
	return nil
}

// CreateProject 创建文档项目（CLI 与 API 共用入口）：
// rule 使用与 project update 一致的字段名（url、latest、version、single、sourcedir、
// lang、requirement、install、index、secret、domain、sslcrt、sslkey、builder、before、after等），
// 其中 url 必需，其余字段为空或未提供时沿用系统默认值。
func (pm *ProjectManager) CreateProject(name string, rule map[string]any) (opt Options, err error) {
	name = strings.ToLower(strings.TrimSpace(name))
	rawurl := util.ParamString(rule["url"])
	if rawurl == "" {
		err = errors.New("empty url")
		return
	}
	if pm.HasName(name) {
		err = errors.New("the name already exists")
		return
	}

	opt, err = pm.GenerateOption(name, rawurl)
	if err != nil {
		return
	}

	for key, value := range rule {
		field := OptionKeyMap(optionAlias(key))
		// 名称与地址由 GenerateOption 处理
		if field == "Name" || field == "URL" {
			continue
		}
		// 空值表示不覆盖默认配置
		if util.ParamString(value) == "" {
			continue
		}
		if err = pm.SetOptionValue(&opt, key, value); err != nil {
			return
		}
	}

	err = pm.Create(name, opt)
	return
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
		opt.SourceDir == "" {
		return errors.New("required fields are missing")
	}
	// 校验python版本是否已配置
	if !pm.cfg.HasPyVersion(string(opt.Version)) {
		return fmt.Errorf(
			"unsupported python version: %s, available: %s",
			opt.Version, strings.Join(pm.cfg.PyVersions(), ", "),
		)
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
	err = pm.db.Create(projectFromOptions(opt)).Error
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

// GetSourceName 查询名为 name 的文档项目数据（序列化为JSON格式返回）
func (pm *ProjectManager) GetSourceName(name string) (value []byte, err error) {
	opt, err := pm.GetName(name)
	if err != nil {
		return
	}
	return json.Marshal(opt)
}

// GetName 查询名为 name 的文档项目数据
func (pm *ProjectManager) GetName(name string) (opt Options, err error) {
	var p store.Project
	err = pm.db.Where("name = ?", strings.ToLower(name)).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("not found project")
		}
		return
	}
	return optionsFromProject(&p), nil
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
	err = pm.db.Model(&store.Project{}).Order("name").Pluck("name", &members).Error
	return
}

// ListBuildset 获取所有构建集
func (pm *ProjectManager) ListBuildset(name string) (builders []Result, err error) {
	var rows []store.BuildResult
	err = pm.db.Where("project = ?", strings.ToLower(name)).
		Order("branch").Find(&rows).Error
	if err != nil {
		return
	}
	builders = make([]Result, 0, len(rows))
	for _, row := range rows {
		builders = append(builders, resultFromBuildResult(&row))
	}
	return builders, nil
}

// GetBuildset 获取某个构建结果
func (pm *ProjectManager) GetBuildset(name, branch string) (builder Result, err error) {
	var row store.BuildResult
	err = pm.db.Where("project = ? AND branch = ?", strings.ToLower(name), branch).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("not found branch")
		}
		return
	}
	return resultFromBuildResult(&row), nil
}

// resultFromBuildResult 将构建结果模型转换为 Result
func resultFromBuildResult(row *store.BuildResult) Result {
	return Result{
		Branch:   row.Branch,
		Status:   row.Status,
		Sender:   vars.Sender(row.Sender),
		Btime:    row.Btime,
		Usedtime: row.Usedtime,
	}
}

func (pm *ProjectManager) renderNginx(opt *Options) error {
	name := opt.Name
	if opt.Lang == "" {
		return errors.New("empty language cannot render nginx")
	}
	basedir := pm.cfg.BaseDir()
	DocsDir := filepath.Join(basedir, "docs")
	dftNginxDir := filepath.Join(basedir, "nginx")
	NginxDir := pm.cfg.MustPath("nginx", "conf_dir", dftNginxDir)
	NginxExtDir := pm.cfg.MustPath("nginx", "conf_ext_dir", dftNginxDir)
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
	if !gtc.IsDir(NginxExtDir) {
		err := gtc.CreateDir(NginxExtDir)
		if err != nil {
			return err
		}
	}
	// 渲染默认域名的nginx配置
	dftLang := strings.Split(opt.Lang, ",")[0]
	dftNgxFile := filepath.Join(NginxDir, fmt.Sprintf("%s.conf", name))
	cstNgxFile := filepath.Join(NginxExtDir, fmt.Sprintf("%s.conf", name))
	dftSSLCrt := pm.cfg.GetKey("nginx", "ssl_crt")
	dftSSLKey := pm.cfg.GetKey("nginx", "ssl_key")
	// 兼容旧版本的配置，如果渲染时存在则自动删除
	dftNgxFileOld := filepath.Join(dftNginxDir, fmt.Sprintf("%s.conf", name))
	cstNgxFileOld := filepath.Join(dftNginxDir, fmt.Sprintf("%s.ext.conf", name))
	if gtc.IsFile(dftNgxFileOld) {
		os.Remove(dftNgxFileOld)
	}
	if gtc.IsFile(cstNgxFileOld) {
		os.Remove(cstNgxFileOld)
	}
	// 静态资源缓存时间，非正整数表示不设置缓存头
	staticExpires := pm.cfg.MustKey("nginx", "static_expires", "3600")
	if n, e := strconv.Atoi(staticExpires); e != nil || n <= 0 {
		staticExpires = ""
	}
	ngxopt := &nginxOptions{
		Name: name, Lang: dftLang, Domain: opt.DefaultDomain, DocsDir: DocsDir,
		Single: opt.Single, SSLCrt: dftSSLCrt, SSLKey: dftSSLKey,
		StaticExpires: staticExpires,
		HTMLNoCache:   gtc.IsTrue(pm.cfg.MustKey("nginx", "html_nocache", "on")),
		OpenFileCache: gtc.IsTrue(pm.cfg.MustKey("nginx", "open_file_cache", "on")),
	}
	dftConf, err := ngxopt.render()
	if err != nil {
		return err
	}
	err = os.WriteFile(dftNgxFile, []byte(dftConf), 0644)
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
		err = os.WriteFile(cstNgxFile, []byte(cstConf), 0644)
		if err != nil {
			return err
		}
	} else {
		// conf_ext_dir 未单独配置时与默认配置同路径，此时不能删除
		if cstNgxFile != dftNgxFile && gtc.IsFile(cstNgxFile) {
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

// BuildRecord 记录构建结果（同项目同分支仅保留最新一条）
func (pm *ProjectManager) BuildRecord(name string, branchOrTag string, result Result) error {
	name = strings.ToLower(name)
	row := store.BuildResult{
		Project:  name,
		Branch:   branchOrTag,
		Status:   result.Status,
		Sender:   string(result.Sender),
		Btime:    result.Btime,
		Usedtime: result.Usedtime,
	}
	// 使用主键做upsert：存在则更新，不存在则新增
	var old store.BuildResult
	err := pm.db.Where("project = ? AND branch = ?", name, branchOrTag).First(&old).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return pm.db.Create(&row).Error
	}
	row.ID = old.ID
	row.CreatedAt = old.CreatedAt
	return pm.db.Model(&old).Select(
		"status", "sender", "btime", "usedtime",
	).Updates(row).Error
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
	dftNginxDir := filepath.Join(basedir, "nginx")
	NginxDir := pm.cfg.MustPath("nginx", "conf_dir", dftNginxDir)
	NginxExtDir := pm.cfg.MustPath("nginx", "conf_ext_dir", dftNginxDir)
	dftNgxFile := filepath.Join(NginxDir, fmt.Sprintf("%s.conf", name))
	cstNgxFile := filepath.Join(NginxExtDir, fmt.Sprintf("%s.conf", name))
	// 兼容旧版本的扩展配置，如果渲染时存在则自动删除
	dftNgxFileOld := filepath.Join(dftNginxDir, fmt.Sprintf("%s.conf", name))
	cstNgxFileOld := filepath.Join(dftNginxDir, fmt.Sprintf("%s.ext.conf", name))
	if gtc.IsFile(dftNgxFileOld) {
		os.Remove(dftNgxFileOld)
	}
	if gtc.IsFile(cstNgxFileOld) {
		os.Remove(cstNgxFileOld)
	}
	if gtc.IsDir(DocsDir) {
		err = os.RemoveAll(DocsDir)
		if err != nil {
			return err
		}
	}
	if gtc.IsFile(dftNgxFile) || gtc.IsFile(cstNgxFile) || gtc.IsFile(cstNgxFileOld) || gtc.IsFile(dftNgxFileOld) {
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

	// 删除项目及其构建结果（事务）
	err = pm.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("name = ?", name).Delete(&store.Project{}).Error; e != nil {
			return e
		}
		return tx.Where("project = ?", name).Delete(&store.BuildResult{}).Error
	})
	return err
}

// Update 更新文档项目配置
func (pm *ProjectManager) Update(opt *Options, rule map[string]any) (ok []string, fail []string, err error) {
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

	err = pm.SaveOptions(opt)
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
	return pm.SaveOptions(&opt)
}

// SaveOptions 将项目配置整体写回数据库
func (pm *ProjectManager) SaveOptions(opt *Options) error {
	name := strings.ToLower(opt.Name)
	var p store.Project
	err := pm.db.Where("name = ?", name).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("not found project")
		}
		return err
	}

	data := projectFromOptions(*opt)
	data.ID = p.ID
	data.CreatedAt = p.CreatedAt
	return pm.db.Save(data).Error
}
