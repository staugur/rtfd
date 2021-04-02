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

	ResetEmpty = `\`

	WebhookID = "_webhook_id"
	InstallID = "_installation_id"
	GitHubApi = "https://api.github.com"
)
