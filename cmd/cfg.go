package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"tcw.im/rtfd/pkg/conf"

	"github.com/spf13/cobra"
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
			data := c.SecHash(args[0])
			printResult(isJSON, data)

		//读取section下的key值
		case 2:
			data := c.GetKey(args[0], args[1])
			if strings.ToLower(args[0]) == "default" && args[1] == "base_dir" {
				data = c.BaseDir()
			}
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
