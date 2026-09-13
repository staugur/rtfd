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

	_ "pkg.tcw.im/rtfd/v2/assets"
	"pkg.tcw.im/rtfd/v2/cmd"
)

// @title Rtfd API
// @version 2.0.0
// @description rtfd 是自托管的 Sphinx 文档构建与托管服务，本文档描述其 HTTP API（版本号与 assets/VERSION 保持一致）。
// @description 所有接口挂 /rtfd 前缀，响应统一为 {"success":bool,"message":string,"data":any}，
// @description 业务错误同样以该结构返回（HTTP 200），由 customHTTPErrorHandler 统一处理。
// @description 鉴权：管理接口（项目管理、配置查询、导入导出）与项目级接口（详情/更新/删除/导出）采用 HMAC-SHA256 动态签名，
// @description 需随请求携带 X-Rtfd-Ts（Unix 秒）、X-Rtfd-Nonce（随机串）、X-Rtfd-Sign（HMAC-SHA256 签名）三个头；
// @description 签名串 = ts + "\n" + nonce（以 secret 为 HMAC 密钥，与 method/path/body 无关），密钥为 [api] secret 或项目 secret（二者其一即可）。
// @description 可使用 `rtfd sign` 命令生成签名，或在 Swagger UI 首页填入密钥后自动签名；构建与 webhook 接口使用项目 secret（HMAC/Token）。
// @description 多数接口兼容两种路径风格：/rtfd/{name}/xxx 与 /rtfd/xxx/{name}（文档只列出前者）。
// @license.name Apache-2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0
// @host localhost:5000
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey RtfdSign
// @in header
// @name X-Rtfd-Sign
// @description 动态签名：随请求同时携带 X-Rtfd-Ts、X-Rtfd-Nonce、X-Rtfd-Sign（HMAC-SHA256，密钥为 [api] secret 或项目 secret）
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cmd.Execute()
}
