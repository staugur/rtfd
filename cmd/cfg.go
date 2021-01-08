package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"rtfd/internal/conf"

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

// cfgCmd represents the cfg command
var cfgCmd = &cobra.Command{
	Use:   "cfg",
	Short: "查询配置文件的配置内容",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON, _ := cmd.Flags().GetBool("json")

		c, err := conf.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		switch len(args) {
		// 读取单个section配置
		case 1:
			data := c.SecHash(changeDefaultSection(args[0]))
			printResult(isJSON, data)

		//读取section下的key值
		case 2:
			data := c.GetKey(changeDefaultSection(args[0]), args[1])
			printResult(isJSON, data)

		//读取所有分区
		default:
			data := c.AllHash()
			printResult(isJSON, data)
		}
	},
}

func init() {
	rootCmd.AddCommand(cfgCmd)
	cfgCmd.Flags().BoolP(
		"json", "j", false, "使用JSON格式显示结果",
	)
}

func changeDefaultSection(section string) string {
	if strings.ToLower(section) == "default" {
		return ini.DEFAULT_SECTION
	}
	return section
}

func printResult(isJSON bool, data interface{}) {
	if !isJSON {
		fmt.Printf("%+v\n", data)
		return
	}
	bytes, err := json.Marshal(data)
	if err == nil {
		fmt.Println(string(bytes))
	}
}
