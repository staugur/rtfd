package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "创建文档项目",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create called")
	},
}

func init() {
	projectCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("name", "n", "", "名称")
	createCmd.Flags().StringP("url", "", "", "文档项目的git仓库地址，如果是私有仓库，请在url协议后携带编码后的username:password")
	createCmd.Flags().StringP("latest", "", "master", "latest所指向的分支")
	createCmd.Flags().BoolP("single", "", false, "是否为单一版本")
	createCmd.Flags().StringP("sourcedir", "s", "docs", "实际文档文件所在目录，目录路径是项目的相对位置")
	createCmd.Flags().StringP("lang", "l", "en", "文档语言，支持多种，以英文逗号分隔")
	createCmd.Flags().IntP("version", "v", 2, "Python版本，2或3")
	createCmd.Flags().StringP("requirement", "r", "", "需要安装的依赖包文件（文件路径是项目的相对位置），支持多个，以英文逗号分隔")
	createCmd.Flags().BoolP("install", "", false, "是否需要安装项目")
	createCmd.Flags().StringP("index", "i", "", "指定pip安装时的pypi源")
	createCmd.Flags().BoolP("nav", "", false, "是否显示导航")
	createCmd.Flags().StringP("secret", "", "", "Webhook密钥")
	createCmd.Flags().StringP("domain", "", "", "自定义域名")
	createCmd.Flags().StringP("sslcrt", "", "", "自定义域名的SSL证书公钥")
	createCmd.Flags().StringP("sslkey", "", "", "自定义域名的SSL证书私钥")
	createCmd.Flags().StringP("builder", "b", "html", "Sphinx构建器")
}
