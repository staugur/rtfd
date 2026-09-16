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

package api

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pkg.tcw.im/rtfd/v2/pkg/lib"
	"pkg.tcw.im/rtfd/v2/pkg/util"
	"pkg.tcw.im/rtfd/v2/vars"

	"github.com/labstack/echo/v4"
	"pkg.tcw.im/gtc"
)

type res struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type resb struct {
	res
	Branch string `json:"branch"`
}
type resp struct {
	res
	Ping string `json:"ping"`
}
type resd struct {
	res
	Data any `json:"data"`
}

// descData 项目描述数据，供文档页面浮动挂件（rtfd.js）渲染
type descData struct {
	// 仓库地址（私有仓库会剥敏）
	URL string `json:"url"`
	// 语言列表
	Lang []string `json:"lang"`
	// 最新版本（分支或tag）
	Latest string `json:"latest"`
	// 自定义域名，未设置时为 false
	DN any `json:"dn"`
	// 文档源目录
	SourceDir string `json:"sourceDir"`
	// 是否单一版本
	Single bool `json:"single"`
	// Sphinx 构建器
	Builder string `json:"builder"`
	// 是否显示挂件导航
	ShowNav bool `json:"showNav"`
	// 系统默认分支
	DefaultBranch string `json:"defaultBranch"`
	// 是否隐藏git入口（非html构建器时恒为true）
	HideGit bool `json:"hideGit"`
	// 是否公开仓库
	Public bool `json:"public"`
	// git 服务商（GitHub/Gitee）
	GSP string `json:"gsp"`
	// 各语言的可用版本：{语言: [版本]}
	Versions map[string][]string `json:"versions"`
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := 200
	msg := err.Error()
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = he.Message.(string)
	}
	c.JSON(code, res{false, msg})
}

// apiDesc 获取项目描述
// @Summary 获取项目描述
// @Description 返回项目 URL、语言、最新版本、可用版本、构建器、是否隐藏git等元数据，供文档页面浮动挂件（rtfd.js）渲染。
// @Description 无需鉴权。路径别名：GET /rtfd/desc/{name}
// @Tags 挂件
// @Produce json
// @Param name path string true "项目名称（不区分大小写）"
// @Success 200 {object} resd{data=descData} "项目元数据，data.versions 为 {语言: [版本]} 映射"
// @Failure default {object} res "项目不存在或数据目录无效（success=false）"
// @Router /rtfd/{name}/desc [get]
func apiDesc(c echo.Context) error {
	name := c.Param("name")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}
	basedir := pm.CFG().BaseDir()
	if basedir == "" || !gtc.IsDir(basedir) {
		return c.JSON(200, res{Message: "invalid data directory"})
	}

	data := descData{
		Lang:          strings.Split(opt.Lang, ","),
		Latest:        opt.Latest,
		DN:            false,
		SourceDir:     opt.SourceDir,
		Single:        opt.Single,
		Builder:       string(opt.Builder),
		ShowNav:       opt.ShowNav,
		DefaultBranch: pm.CFG().DefaultBranch(),
		HideGit:       opt.HideGit,
		Public:        opt.IsPublic,
		GSP:           opt.GSP,
		Versions:      make(map[string][]string),
	}
	if opt.IsPublic {
		data.URL = opt.URL
	} else {
		urlpub, _ := util.PublicGitURL(opt.URL)
		data.URL = urlpub
	}
	if util.IsDomain(opt.CustomDomain) {
		data.DN = opt.CustomDomain
	}
	if opt.Builder != "html" {
		data.HideGit = true
	}

	for _, lang := range strings.Split(opt.Lang, ",") {
		langDir := filepath.Join(basedir, "docs", name, lang)
		if !gtc.IsDir(langDir) {
			continue
		}
		ifs, err := os.ReadDir(langDir)
		if err != nil {
			continue
		}
		vs := []string{"latest"}
		for _, f := range ifs {
			fname := f.Name()
			if f.IsDir() && fname != "" && fname != "." && fname != ".." {
				vs = append(vs, fname)
			}
		}
		if len(vs) == 1 && vs[0] == "latest" {
			continue
		}
		data.Versions[lang] = vs
	}
	return c.JSON(200, resd{res{Success: true}, data})
}

// apiBuild 触发构建
// @Summary 触发构建
// @Description 异步触发指定项目的文档构建，接口立即返回 201，构建结果可经项目详情（build=1）查询。
// @Description 路径别名：POST /rtfd/build/{name}；鉴权：项目 secret（未设置则免鉴权）
// @Tags 构建
// @Accept json
// @Produce json
// @Param name path string true "项目名称"
// @Param branch query string false "分支或tag，缺省用项目最新版本"
// @Param debug query string false "为 true/on/1 时输出完整构建日志（BuildWithAll）"
// @Security RtfdSign
// @Success 201 {object} resb "已异步启动，branch 回显实际分支"
// @Failure default {object} res "项目不存在或签名校验失败"
// @Router /rtfd/{name}/build [post]
func apiBuild(c echo.Context) error {
	if ok, err := checkSecret(c); !ok {
		return err
	}
	name := c.Param("name")
	branch := getArg(c, "branch")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}

	// 复用API启动时创建的构建器（同一数据库连接）
	isDebug := gtc.IsTrue(getArg(c, "debug"))
	go func() {
		if isDebug {
			bld.BuildWithAll(name, branch, vars.APISender)
		} else {
			bld.BuildWithLog(name, branch, vars.APISender)
		}
	}()
	return c.JSON(201, resb{res{Success: true}, branch})
}

// webhookBuild git仓库webhook
// @Summary git仓库webhook
// @Description 接收 GitHub（校验 X-Hub-Signature HMAC-SHA1）与 Gitee（校验 X-Gitee-Token）的 push/release 事件，
// @Description 校验通过后异步构建；ping 事件直接返回 pong；命中项目 meta 的 excluded_branch 时忽略。
// @Description 路径别名：POST /rtfd/webhook/{name}
// @Tags webhook
// @Accept json
// @Produce json
// @Param name path string true "项目名称"
// @Success 201 {object} resb "已异步启动构建（命中 excluded_branch 时返回 200 且 success=false）"
// @Success 200 {object} resp "ping 事件回 pong（success=true，ping=pong）"
// @Failure default {object} res "来源、事件类型或签名校验失败"
// @Router /rtfd/{name}/webhook [post]
func webhookBuild(c echo.Context) error {
	var gst, event string

	H := c.Request().Header
	agent := H.Get("User-Agent")
	if strings.HasPrefix(agent, "GitHub-Hookshot") {
		gst = vars.GSPGitHub
		event = H.Get("X-GitHub-Event")
	} else if agent == "git-oschina-hook" {
		gst = vars.GSPGitee
		evt := H.Get("X-Gitee-Event")
		ping := H.Get("X-Gitee-Ping")
		if gtc.IsTrue(ping) {
			event = "ping"
		} else if evt == "Push Hook" {
			event = "push"
		} else if evt == "Tag Push Hook" {
			event = "release"
		}
	} else {
		return errors.New("unsupported provider")
	}
	if event == "" {
		return errors.New("invalid event type")
	}
	if event == "ping" {
		return c.JSON(200, resp{res{Success: true}, "pong"})
	}
	if !gtc.StrInSlice(event, []string{"ping", "push", "release"}) {
		return errors.New("unsupported webhook event")
	}

	name := c.Param("name")
	branch := ""
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}

	var body map[string]any
	RawBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(RawBody, &body); err != nil {
		return err
	}

	if gst == vars.GSPGitHub {
		if err := checkGitHubWebhook(c, opt, RawBody); err != nil {
			return err
		}
		if event == "push" {
			ref := strings.Split(body["ref"].(string), "/")
			branch = ref[len(ref)-1]
		} else {
			action := body["action"].(string)
			if action == "released" {
				release := body["release"].(map[string]any)
				branch = release["tag_name"].(string)
			} else {
				return errors.New("the action is ignored in the release event")
			}
		}
	} else if gst == vars.GSPGitee {
		if err := checkGiteeWebhook(c, opt); err != nil {
			return err
		}
		ref := strings.Split(body["ref"].(string), "/")
		branch = ref[len(ref)-1]
	} else {
		return errors.New("unsupported git service provider")
	}

	sep := opt.GetMeta("excluded_sep")
	if sep == "" {
		sep = opt.MustMeta("_sep", "|")
	}
	if gtc.StrInSlice(branch, strings.Split(opt.GetMeta("excluded_branch"), sep)) {
		return c.JSON(200, resb{res{false, "excluded branch"}, branch})
	}

	go bld.BuildWithLog(name, branch, vars.WebhookSender)
	return c.JSON(201, resb{res{Success: true}, branch})
}

// apiBadge 文档状态徽章
// @Summary 文档状态徽章
// @Description 返回 SVG 状态徽章（passing/failing/unknown），可直接内嵌到 README。无需鉴权。
// @Description 路径别名：GET /rtfd/badge/{name}
// @Tags 挂件
// @Produce image/svg+xml
// @Param name path string true "项目名称"
// @Param branch query string false "分支或tag，缺省用项目最新版本；latest 亦表示最新版本"
// @Success 200 {string} string "SVG 内容"
// @Failure default {object} res "查询构建结果失败"
// @Router /rtfd/{name}/badge [get]
func apiBadge(c echo.Context) error {
	passing := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="86" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="86" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#4c1" d="M35 0h51v20H35z"/><path fill="url(#b)" d="M0 0h86v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="595" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="410">passing</text><text x="595" y="140" transform="scale(.1)" textLength="410">passing</text></g> </svg>`
	failing := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="78" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="78" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#e05d44" d="M35 0h43v20H35z"/><path fill="url(#b)" d="M0 0h78v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="555" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="330">failing</text><text x="555" y="140" transform="scale(.1)" textLength="330">failing</text></g> </svg>`
	unknown := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="96" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="96" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#dfb317" d="M35 0h61v20H35z"/><path fill="url(#b)" d="M0 0h96v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="645" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="510">unknown</text><text x="645" y="140" transform="scale(.1)" textLength="510">unknown</text></g> </svg>`

	name := c.Param("name")
	branch := strings.ToLower(getArg(c, "branch"))
	status := ""

	if !pm.HasName(name) {
		return badgeRes(c, unknown)
	}

	if branch == "" || branch == "latest" {
		opt, err := pm.GetName(name)
		if err != nil {
			return err
		}
		branch = opt.Latest
	}

	builder, err := pm.GetBuildset(name, branch)
	if err != nil {
		if strings.HasPrefix(err.Error(), "not found branch") {
			return badgeRes(c, unknown)
		}
		return err
	}
	if status == "" {
		if builder.Status {
			status = passing
		} else {
			status = failing
		}
	}
	return badgeRes(c, status)
}

// ghApp GitHub App 事件回调
// @Summary GitHub App 事件回调
// @Description 接收 installation / installation_repositories 事件（需 header X-GitHub-Hook-Installation-Target-Type=integration），
// @Description 校验 App ID 后同步项目 meta 与仓库级 webhook
// @Tags webhook
// @Accept json
// @Produce json
// @Success 200 {object} res "处理成功"
// @Failure default {object} res "事件类型、目标类型或 App ID 校验失败"
// @Router /rtfd/github/app [post]
func ghApp(c echo.Context) error {
	H := c.Request().Header
	evt := H.Get("X-GitHub-Event")
	hiti := H.Get("X-GitHub-Hook-Installation-Target-ID")
	hitt := H.Get("X-GitHub-Hook-Installation-Target-Type")
	if !gtc.StrInSlice(evt, []string{"installation", "installation_repositories"}) {
		return errors.New("unsupported event")
	}
	if hitt != "integration" {
		return errors.New("unsupported type")
	}
	hitiU, err := strconv.ParseUint(hiti, 10, 64)
	if err != nil {
		return err
	}

	gh, err := lib.NewGHApp(pm)
	if err != nil {
		return err
	}
	gh.BaseURL(getBaseURL(c))

	var data lib.AppWebhook
	if err := c.Bind(&data); err != nil {
		return err
	}
	if hitiU != data.Installation.AppID {
		return errors.New("not match installation app")
	}

	err = gh.Dispatch(data)
	if err != nil {
		return err
	}
	return c.JSON(200, res{Success: true})
}
