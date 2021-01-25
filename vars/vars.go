// 程序全局变量固定值

package vars

var (
	// DFT 默认值
	DFT = "default"

	// GBName 特殊的 桶 的名称
	GBName = "global"
	// GBPK 特殊 桶 下的文档项目名称集合Key，set类型
	GBPK = []byte("projects")
	// GBDK 特殊 桶 下所有自定义的域名集合Key，set类型
	GBDK = []byte("domains")

	// BCK 文档项目 桶 下的配置索引Key，hash类型
	BCK = []byte("config")
	// BRLK 文档项目 桶 下的构建索引Key（构建的分支、标签索引），set类型
	BRLK = []byte("builders")
)

// BRK 构建结果 桶 ，与文档项目并列
// 平行结构：
//   - project: Bucket -> name , key -> config, builder;  value
//   - builder: Bucket -> name:Project, key -> branch, tag; Result struct
func BRK(projectName string) string {
	return projectName + ":builder"
}
