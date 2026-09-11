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

package main

import (
	"log"

	_ "pkg/tcw.im/rtfd/assets"
	"pkg/tcw.im/rtfd/cmd"
)

// @title Rtfd API
// @version 2.0.0
// @description rtfd 是自托管的 Sphinx 文档构建与托管服务，本文档描述其 HTTP API（版本号与 assets/VERSION 保持一致）。
// @description 所有接口挂 /rtfd 前缀，响应统一为 {"success":bool,"message":string,"data":any}，
// @description 业务错误同样以该结构返回（HTTP 200），由 customHTTPErrorHandler 统一处理。
// @description 鉴权：管理接口（项目管理、配置查询、导入导出）需请求头 X-Rtfd-Sign，值为 MD5(配置 [api] secret)；
// @description 项目级接口（详情/更新/删除/导出）额外接受 MD5(项目 secret)；构建与 webhook 接口使用项目 secret。
// @description 多数接口兼容两种路径风格：/rtfd/{name}/xxx 与 /rtfd/xxx/{name}（文档只列出前者）。
// @license.name Apache-2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0
// @host localhost:5000
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey RtfdSign
// @in header
// @name X-Rtfd-Sign
// @description 密钥签名：MD5([api] secret) 或 MD5(项目 secret)
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cmd.Execute()
}
