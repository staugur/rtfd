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

// 项目管理接口：为web管理端提供与CLI project子命令一致的增删改查能力

package api

import (
	"errors"
	"strings"

	"pkg.tcw.im/rtfd/v2/pkg/lib"
	"pkg.tcw.im/rtfd/v2/pkg/util"
	"pkg.tcw.im/rtfd/v2/vars"

	"github.com/labstack/echo/v4"
	"pkg.tcw.im/gtc"
)

// reservedParams 管理接口的保留参数，不作为项目配置字段
var reservedParams = []string{
	"name", "key", "text", "file", "sep", "verbose", "build", "debug", "log", "sysmeta", "export", "data",
}

type resup struct {
	res
	// 更新成功的字段
	Updated []string `json:"updated"`
	// 更新失败的字段（格式 field:reason）
	Failed []string `json:"failed"`
}

// keyValue 单项查询结果（项目字段、系统配置项）
type keyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// exportData 导出项目配置
type exportData struct {
	// 项目名称
	Name string `json:"name"`
	// base64 编码的配置
	Export string `json:"export"`
}

// importData 导入项目配置
type importData struct {
	// 导入后的项目名称
	Name string `json:"name"`
}

// apiProjectList 项目列表：GET /rtfd/projects[?verbose=1]
// apiProjectList 项目列表
// @Summary 项目列表
// @Description 返回全部项目名称；verbose=1 时返回完整项目配置数组。需管理密钥。
// @Tags 项目管理
// @Produce json
// @Param verbose query string false "为 true/on/1 时返回完整配置（[]lib.OptionsWithResult）"
// @Security RtfdSign
// @Success 200 {object} resd{data=[]string} "项目名称数组；verbose 时为完整配置数组"
// @Failure default {object} res "密钥未配置或校验失败"
// @Router /rtfd/projects [get]
func apiProjectList(c echo.Context) error {
	if err := checkAPISecret(c); err != nil {
		return err
	}
	if gtc.IsTrue(getArg(c, "verbose")) {
		members, err := pm.ListFullProject()
		if err != nil {
			return err
		}
		return c.JSON(200, resd{res{Success: true}, members})
	}
	members, err := pm.ListProject()
	if err != nil {
		return err
	}
	return c.JSON(200, resd{res{Success: true}, members})
}

// apiProjectCreate 创建项目：POST /rtfd/projects
// 参数与CLI一致（url必需），其余字段如 latest/version/single/sourcedir/lang/
// requirement/install/index/builder/secret/domain/sslcrt/sslkey 可选
// apiProjectCreate 创建项目
// @Summary 创建项目
// @Description 参数与 CLI `rtfd project create` 一致：url 必需，其余字段可选、空值沿用系统默认；创建后自动汇总渲染 Caddy 配置。
// @Description 参数可放在表单、query 或 JSON body（键不区分大小写）；需管理密钥。
// @Tags 项目管理
// @Accept x-www-form-urlencoded
// @Produce json
// @Param name formData string true "项目名称（仅小写字母数字与-_.）"
// @Param url formData string true "git 仓库地址（https 或 ssh）"
// @Param latest formData string false "分支或tag，缺省用系统 default_branch"
// @Param version formData string false "Python 版本（需在 [py] 分区中定义，如 3.10）"
// @Param lang formData string false "语言，多个用英文逗号分隔，如 zh_CN,en"
// @Param single formData string false "是否单一版本（true/false）"
// @Param sourcedir formData string false "文档源目录，默认 docs"
// @Param requirement formData string false "依赖文件相对路径（相对文档源目录）"
// @Param install formData string false "是否安装依赖（true/false）"
// @Param index formData string false "pip 源索引地址"
// @Param builder formData string false "Sphinx 构建器：html/dirhtml/singlehtml"
// @Param secret formData string false "项目密钥（构建与 webhook 签名用）"
// @Param domain formData string false "自定义域名（需已解析到本机）"
// @Param sslcrt formData string false "自定义域名证书路径（与 sslkey 成对）"
// @Param sslkey formData string false "自定义域名证书私钥路径（与 sslcrt 成对）"
// @Security RtfdSign
// @Success 201 {object} resd{data=lib.Options} "创建成功，data 为项目配置"
// @Failure default {object} res "参数非法、名称/自定义域名已存在或密钥校验失败"
// @Router /rtfd/projects [post]
func apiProjectCreate(c echo.Context) error {
	if err := checkAPISecret(c); err != nil {
		return err
	}
	params, err := getFormParams(c)
	if err != nil {
		return err
	}
	name := util.ParamString(params["name"])
	if name == "" {
		return errors.New("empty name")
	}
	opt, err := pm.CreateProject(name, ruleFromParams(params))
	if err != nil {
		return err
	}
	return c.JSON(201, resd{res{Success: true}, opt})
}

// apiProjectInfo 项目详情：GET /rtfd/:name/info[?key=Field][&build=1]
// apiProjectInfo 项目详情
// @Summary 项目详情
// @Description 默认返回项目完整配置；key 指定单个字段；build=1 时附带构建集。需管理密钥或项目密钥。
// @Description 路径别名：GET /rtfd/info/{name}
// @Tags 项目管理
// @Produce json
// @Param name path string true "项目名称"
// @Param key query string false "字段名，如 lang、secret、url，返回 data.{key,value}"
// @Param build query string false "为 true/on/1 时附带构建集（data.buildset）"
// @Security RtfdSign
// @Success 200 {object} resd{data=lib.OptionsWithResult} "项目配置（key 时为 data.{key,value}）"
// @Failure default {object} res "项目不存在或密钥校验失败"
// @Router /rtfd/{name}/info [get]
func apiProjectInfo(c echo.Context) error {
	if err := checkProjectSecret(c); err != nil {
		return err
	}
	name := c.Param("name")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	if key := getArg(c, "key"); key != "" {
		val, err := pm.GetNameOption(name, key)
		if err != nil {
			return err
		}
		return c.JSON(200, resd{res{Success: true}, keyValue{key, val}})
	}
	if gtc.IsTrue(getArg(c, "build")) {
		bs, err := pm.GetNameWithBuildset(name)
		if err != nil {
			return err
		}
		return c.JSON(200, resd{res{Success: true}, bs})
	}
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}
	return c.JSON(200, resd{res{Success: true}, opt})
}

// apiProjectUpdate 更新项目配置：POST /rtfd/:name/update
// 支持三种方式：text（Field:Value,...）、file（服务端规则文件路径）、
// 以及直接以字段名传参（如 lang=zh_CN&single=on）
// apiProjectUpdate 更新项目配置
// @Summary 更新项目配置
// @Description 三种传参方式（按优先级）：text（Field:Value 逗号分隔，sep 可自定义分隔符）、file（服务端 .rtfd.ini 路径，内容未变化则跳过）、
// @Description 或直接以字段名传参（如 lang=zh_CN&single=on）。值 - 表示重置为空（requirement/index/secret）。
// @Description 字段名同创建项目；需管理密钥或项目密钥。路径别名：POST /rtfd/update/{name}
// @Tags 项目管理
// @Accept x-www-form-urlencoded
// @Produce json
// @Param name path string true "项目名称"
// @Param text formData string false "更新规则，如 lang:zh_CN,single:on"
// @Param sep formData string false "text 的字段分隔符，默认英文逗号"
// @Param file formData string false "服务端规则文件路径（.rtfd.ini）"
// @Param url formData string false "直接传字段：见创建项目参数列表"
// @Param lang formData string false "直接传字段：语言，如 en"
// @Param single formData string false "直接传字段：是否单一版本"
// @Param latest formData string false "直接传字段：分支或tag"
// @Param version formData string false "直接传字段：Python 版本"
// @Param sourcedir formData string false "直接传字段：文档源目录"
// @Param requirement formData string false "直接传字段：依赖文件路径，- 重置"
// @Param install formData string false "直接传字段：是否安装依赖"
// @Param index formData string false "直接传字段：pip 源索引，- 重置"
// @Param builder formData string false "直接传字段：Sphinx 构建器"
// @Param secret formData string false "直接传字段：项目密钥，- 重置"
// @Param domain formData string false "直接传字段：自定义域名，- 取消"
// @Param sslcrt formData string false "直接传字段：证书路径（需与 sslkey 同时提供）"
// @Param sslkey formData string false "直接传字段：证书私钥路径"
// @Param shownav formData string false "直接传字段：是否显示挂件导航"
// @Param hidegit formData string false "直接传字段：是否隐藏git入口"
// @Security RtfdSign
// @Success 200 {object} resup "更新结果，updated/failed 为字段列表；规则文件内容未变化时 updated/failed 为空且 message=not updated"
// @Failure default {object} res "项目不存在、规则为空或密钥校验失败"
// @Router /rtfd/{name}/update [post]
func apiProjectUpdate(c echo.Context) error {
	if err := checkProjectSecret(c); err != nil {
		return err
	}
	name := c.Param("name")
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}
	params, err := getFormParams(c)
	if err != nil {
		return err
	}

	text := util.ParamString(params["text"])
	file := util.ParamString(params["file"])
	var rule map[string]any
	switch {
	case text != "":
		rule, err = lib.ParseUpdateRule(text, util.ParamString(params["sep"]))
		if err != nil {
			return err
		}
	case file != "":
		var fileMD5 string
		rule, fileMD5, err = lib.ParseUpdateFile(file)
		if err != nil {
			return err
		}
		// 规则文件内容未变化则跳过更新
		if md5 := opt.GetMeta(vars.PUFMD5); md5 != "" && fileMD5 != "" && md5 == fileMD5 {
			return c.JSON(200, res{Success: true, Message: "not updated"})
		}
		if err = opt.UpdateMeta(vars.PUFMD5, fileMD5); err != nil {
			return err
		}
	default:
		rule = ruleFromParams(params)
		if len(rule) == 0 {
			return errors.New("empty rule")
		}
	}

	ok, fail, err := pm.Update(&opt, rule)
	if err != nil {
		return err
	}
	return c.JSON(200, resup{res{Success: true, Message: "updated"}, ok, fail})
}

// apiProjectRemove 删除项目：POST|DELETE /rtfd/:name/remove
// apiProjectRemove 删除项目
// @Summary 删除项目
// @Description 删除项目配置与构建结果，并从 Caddy 配置中移除对应站点。需管理密钥或项目密钥。
// @Description 路径别名：/rtfd/remove/{name}（POST 与 DELETE 均可）
// @Tags 项目管理
// @Produce json
// @Param name path string true "项目名称"
// @Security RtfdSign
// @Success 200 {object} res "删除成功"
// @Failure default {object} res "项目不存在或密钥校验失败"
// @Router /rtfd/{name}/remove [delete]
// @Router /rtfd/{name}/remove [post]
func apiProjectRemove(c echo.Context) error {
	if err := checkProjectSecret(c); err != nil {
		return err
	}
	name := c.Param("name")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	if err := pm.Remove(name); err != nil {
		return err
	}
	return c.JSON(200, res{Success: true, Message: "removed"})
}

// apiProjectExport 导出项目配置：GET /rtfd/:name/export[?sysmeta=1]
// apiProjectExport 导出项目配置
// @Summary 导出项目配置
// @Description 返回 base64 编码的项目配置，可通过导入接口或 CLI 在其它 rtfd 实例中还原。
// @Description sysmeta=1 时保留系统 meta（如 _installation_id、_webhook_id）。需管理密钥或项目密钥。
// @Description 路径别名：GET /rtfd/export/{name}
// @Tags 项目管理
// @Produce json
// @Param name path string true "项目名称"
// @Param sysmeta query string false "为 true/on/1 时保留系统 meta"
// @Security RtfdSign
// @Success 200 {object} resd{data=exportData} "data.export 为 base64 配置"
// @Failure default {object} res "项目不存在或密钥校验失败"
// @Router /rtfd/{name}/export [get]
func apiProjectExport(c echo.Context) error {
	if err := checkProjectSecret(c); err != nil {
		return err
	}
	name := c.Param("name")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	encode, err := pm.Export(name, gtc.IsTrue(getArg(c, "sysmeta")))
	if err != nil {
		return err
	}
	return c.JSON(200, resd{res{Success: true}, exportData{strings.ToLower(name), encode}})
}

// apiProjectImport 导入项目配置：POST /rtfd/import
// 参数 export 为base64编码的配置（必需），name 可选（默认取配置中的名称）
// apiProjectImport 导入项目配置
// @Summary 导入项目配置
// @Description 导入 base64 编码的项目配置（由导出接口或 CLI `rtfd project transfer -e` 生成）；
// @Description name 可选，用于改名导入（缺省取配置中的名称）；导入走创建流程，会重新校验并渲染 Caddy 配置。需管理密钥。
// @Tags 项目管理
// @Accept x-www-form-urlencoded
// @Produce json
// @Param export formData string true "base64 编码的项目配置"
// @Param name formData string false "新项目名称，缺省用配置中的名称"
// @Security RtfdSign
// @Success 201 {object} resd{data=importData} "data.name 为导入后的项目名"
// @Failure default {object} res "配置非法、名称已存在或密钥校验失败"
// @Router /rtfd/import [post]
func apiProjectImport(c echo.Context) error {
	if err := checkAPISecret(c); err != nil {
		return err
	}
	params, err := getFormParams(c)
	if err != nil {
		return err
	}
	encoded := util.ParamString(params["export"])
	if encoded == "" {
		encoded = util.ParamString(params["data"])
	}
	if encoded == "" {
		return errors.New("empty export data")
	}
	opt, err := pm.DecodeExport(encoded)
	if err != nil {
		return err
	}
	name, err := pm.Import(opt, util.ParamString(params["name"]))
	if err != nil {
		return err
	}
	return c.JSON(201, resd{res{Success: true}, importData{name}})
}

// ruleFromParams 提取参数中的项目配置字段（排除保留参数），值统一转为字符串
func ruleFromParams(params map[string]any) map[string]any {
	rule := make(map[string]any)
	for k, v := range params {
		if gtc.StrInSlice(k, reservedParams) {
			continue
		}
		rule[k] = util.ParamString(v)
	}
	return rule
}
