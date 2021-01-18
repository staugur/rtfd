// 程序全局变量固定值

package vars

var (
	// DFT 默认值
	DFT = "default"

	// GBName 特殊的 桶 的名称
	GBName = "global"

	// GBPK 特殊 桶 下的文档项目名称集合Key
	GBPK = []byte("projects")
	// GBDK 特殊 桶 下所有自定义的域名集合Key
	GBDK = []byte("domains")
	// BCK 文档项目 桶 下的配置索引Key
	BCK = []byte("config")
)
