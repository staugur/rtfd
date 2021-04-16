package cmd

import (
	"fmt"
	"os"
	"strings"

	"tcw.im/rtfd/pkg/conf"
	"tcw.im/rtfd/pkg/lib"
	"tcw.im/rtfd/vars"

	"github.com/spf13/cobra"
	"tcw.im/gtc"
)

var updateDesc = `更新文档项目配置

第一种方式，通过 text 选项：

    仅可更新部分字段，参考如下列表（即Field，解释说明处小括号为字段类型，无则默认为string）：

    url：        文档项目的git仓库地址
    latest：     latest所指向的分支
    version：    构建文档所用的Python版本，2或3（int）
    single：     是否单一版本（bool）
    source：     文档源文件所在目录
    lang：       文档语言
    requirement：依赖包需求文件，支持多个，以逗号分隔
    install：    是否安装项目（bool）
    index：      pypi源
    builder：    sphinx构建器
    shownav：    是否显示导航（bool）
    hidegit：    导航中是否隐藏git信息（bool）
    secret：     api/webhook密钥
    domain：     自定义域名
    sslcrt：     自定义域名开启HTTPS时的证书公钥
    sslpri：     自定义域名开启HTTPS时的证书私钥
    before：     构建前的钩子命令
    after：      执行构建成功后的钩子命令
    meta：       额外配置数据，每次仅能更新一条，格式是 key=value（key不区分大小写）

    可一次更新一个或多个字段，格式是 -> Field:Value,Field:Value,...,Field:Value
    分隔符可用 sep 选项设置，更新成功或失败的字段均会打印。
    请按照字段类型（如int、bool）填写值，否则可能导致异常。
    请注意：
        # bool类型仅当值为1、true、on时表示true，其他表示false
        # domain字段值为0、false、off时表示取消自定义域名（不更改SSL相关配置）
        # 额外字段ssl（不在列表中）值为0、false、off时表示取消自定义域名SSL
        # 部分更新失败的字段亦可能已造成破坏性更改（如lang、latest、domain）
        # 部分字段仅在下一次构建时生效
        # 特殊字段meta系统内置字段：
            # _sep: 当meta内部字段的值为多值类型时，指定其分隔符，默认是 |

第二种方式，通过 file 选项：

    通常用于构建时更新，编写 rtfd.ini 规则文件放到源码仓库中，在构建时 rtfd 会读取此文件，
    结合系统存储配置（优先级低于规则文件）进行参数化文档构建。

    不过相对于第一种方式，此方式可更新字段较少，仅为构建时参数。
`

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "更新文档项目配置",
	Long:    updateDesc,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"u"},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if name == "" {
			fmt.Println("empty name")
			os.Exit(1)
		}

		sep := cmd.Flag("sep").Value.String()
		text := cmd.Flag("text").Value.String()
		file := cmd.Flag("file").Value.String()
		if text == "" && file == "" {
			fmt.Println("empty text or file")
			os.Exit(1)
		}

		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(127)
		}

		opt, err := pm.GetName(name)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		rule := make(map[string]interface{})
		var isUpFile bool
		var fileMD5 string
		if text != "" {
			var ssl string
			for _, kv := range strings.Split(text, ",") {
				kvs := strings.Split(kv, sep)
				if len(kvs) != 2 {
					fmt.Printf("invalid %s\n", kv)
					os.Exit(1)
				}
				field := kvs[0]
				value := kvs[1]
				if field == "" || value == "" {
					continue
				}
				if field == "sslcrt" {
					ssl = value
				} else if field == "sslpri" {
					ssl += "," + value
				} else if field == "ssl" {
					if gtc.IsFalse(value) {
						ssl = value
					} else {
						fmt.Println("invalid ssl")
						os.Exit(1)
					}
				} else {
					rule[field] = value
				}
			}
			if ssl != "" {
				rule["ssl"] = ssl
			}
		} else {
			if !gtc.IsFile(file) {
				fmt.Println("not found file")
				os.Exit(1)
			}
			//Check if it needs to be updated
			md5 := opt.GetMeta(vars.PUFMD5)
			isUpFile = true
			fileMD5, _ = gtc.MD5File(file)
			if md5 != "" && fileMD5 != "" && fileMD5 == md5 {
				fmt.Println("not updated")
				return
			}
			cfg, err := conf.New(file)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			for k, v := range cfg.SecHash("project") {
				if gtc.StrInSlice(k, []string{"latest"}) {
					rule[k] = v
				}
			}
			for k, v := range cfg.SecHash("sphinx") {
				if gtc.StrInSlice(k, []string{"sourcedir", "lang", "builder"}) {
					rule[k] = v
				}
			}
			for k, v := range cfg.SecHash("python") {
				if gtc.StrInSlice(k, []string{"version", "requirement", "install", "index"}) {
					rule[k] = v
				}
			}
		}

		if len(rule) <= 0 {
			fmt.Println("empty rule")
			os.Exit(1)
		}

		if isUpFile {
			(&opt).UpdateMeta(vars.PUFMD5, fileMD5)
		}
		ok, fail, err := pm.Update(&opt, rule)
		if err != nil {
			fmt.Println(err)
			os.Exit(128)
		}

		fmt.Println("updated")
		if len(ok) > 0 {
			fmt.Printf("成功：%+v\n", strings.Join(ok, ", "))
		}
		if len(fail) > 0 {
			fmt.Printf("失败：%+v\n", strings.Join(fail, ", "))
		}
	},
}

func init() {
	projectCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringP("sep", "s", ":", "设定 Field、Value 之间的分隔符")
	updateCmd.Flags().StringP("text", "t", "", "更新规则文本，格式是 Field:Value,Field:Value")
	updateCmd.Flags().StringP("file", "f", "", "更新规则文件")
}
