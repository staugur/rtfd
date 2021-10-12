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

// 程序全局（超过两个子包使用）类型、变量、常量及函数

package vars

type (
	// Sender 发起构建来源类型
	Sender string
)

const (
	// APISender 从API接口发起构建
	APISender Sender = "api"
	// CLISender 从命令行发起构建
	CLISender Sender = "cli"
	// WebhookSender 从git webhook发起自动构建
	WebhookSender Sender = "webhook"
)

const (
	// DFT 默认值
	DFT = "default"

	GSPGitHub = "GitHub"
	GSPGitee  = "Gitee"
	GSPNA     = "N/A"

	// 此标记要求重置为默认值
	ResetEmpty = "-"

	WebhookID = "_webhook_id"
	InstallID = "_installation_id"
	GitHubApi = "https://api.github.com"

	PUFMD5 = "_update_file_md5"
)
