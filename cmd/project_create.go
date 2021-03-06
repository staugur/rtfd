package cmd

import (
	"fmt"
	"os"

	"tcw.im/rtfd/pkg/lib"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "创建文档项目",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		flagset := cmd.Flags()

		name := args[0]
		if name == "" {
			fmt.Println("empty name")
			os.Exit(1)
		}
		url := cmd.Flag("url").Value.String()
		if url == "" {
			fmt.Println("empty url")
			os.Exit(1)
		}
		latest := cmd.Flag("latest").Value.String()
		single, err := flagset.GetBool("single")
		if err != nil {
			fmt.Printf("invalid param(single): %v\n", single)
			fmt.Println(err)
			os.Exit(1)
		}
		source := cmd.Flag("sourcedir").Value.String()
		lang := cmd.Flag("lang").Value.String()
		pyver, err := flagset.GetUint8("version")
		req := cmd.Flag("requirement").Value.String()
		install, err := flagset.GetBool("install")
		if err != nil {
			fmt.Printf("invalid param(install): %v\n", install)
			fmt.Println(err)
			os.Exit(1)
		}
		index := cmd.Flag("index").Value.String()
		secret := cmd.Flag("secret").Value.String()
		domain := cmd.Flag("domain").Value.String()
		sslcrt := cmd.Flag("sslcrt").Value.String()
		sslkey := cmd.Flag("sslkey").Value.String()
		builder := cmd.Flag("builder").Value.String()

		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(127)
		}
		defer pm.Close()

		if pm.HasName(name) == true {
			fmt.Println("the name already exists")
			os.Exit(128)
		}

		opt, err := pm.GenerateOption(name, url)
		if err != nil {
			fmt.Println(err)
			os.Exit(129)
		}

		// 需要更新值的key
		optBind := make(map[string]interface{})
		optBind["Latest"] = latest
		optBind["Version"] = pyver
		optBind["Single"] = single
		optBind["SourceDir"] = source
		optBind["Lang"] = lang
		optBind["Requirement"] = req
		optBind["Install"] = install
		optBind["Index"] = index
		optBind["ShowNav"] = true
		optBind["Secret"] = secret
		optBind["CustomDomain"] = domain
		optBind["SSLPublic"] = sslcrt
		optBind["SSLPrivate"] = sslkey
		optBind["Builder"] = builder

		for k, v := range optBind {
			pm.SetOption(&opt, k, v)
		}

		fmt.Println(opt)
		err = pm.Create(name, opt)
		if err != nil {
			fmt.Println(err)
			os.Exit(130)
		}
		fmt.Println("created")
	},
}

func init() {
	createCmd.Flags().SortFlags = false
	projectCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("url", "", "", "文档项目的git仓库地址，如果是私有仓库，请在url协议后携带编码后的 username:password")
	createCmd.Flags().StringP("latest", "", "master", "latest所指向的分支")
	createCmd.Flags().BoolP("single", "", false, "是否为单一版本")
	createCmd.Flags().StringP("sourcedir", "s", "docs", "实际文档文件所在目录，目录路径是项目的相对位置")
	createCmd.Flags().StringP("lang", "l", "en", "文档语言，支持多种，以英文逗号分隔")
	createCmd.Flags().Uint8P("version", "v", 3, "Python版本，2或3")
	createCmd.Flags().StringP("requirement", "r", "", "需要安装的依赖包文件（文件路径是项目的相对位置），支持多个，以英文逗号分隔")
	createCmd.Flags().BoolP("install", "", false, "是否需要安装项目")
	createCmd.Flags().StringP("index", "i", "", "指定pip安装时的pypi源")
	createCmd.Flags().StringP("secret", "", "", "Webhook密钥")
	createCmd.Flags().StringP("domain", "", "", "自定义域名")
	createCmd.Flags().StringP("sslcrt", "", "", "自定义域名的SSL证书公钥")
	createCmd.Flags().StringP("sslkey", "", "", "自定义域名的SSL证书私钥")
	createCmd.Flags().StringP("builder", "b", "html", "Sphinx构建器，可选html、dirhtml、singlehtml")
}
