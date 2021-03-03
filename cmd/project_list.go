package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"tcw.im/rtfd/pkg/lib"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有文档项目信息",
	Run: func(cmd *cobra.Command, args []string) {
		flagset := cmd.Flags()
		verbose, err := flagset.GetBool("verbose")
		if err != nil {
			fmt.Printf("invalid param(verbose): %v\n", verbose)
			fmt.Println(err)
			os.Exit(1)
		}

		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(127)
		}

		list, err := pm.ListProject()
		if err != nil {
			fmt.Println(err)
			os.Exit(128)
		}
		members := make([]interface{}, len(list))
		if verbose == true {
			list, err := pm.ListFullProject()
			if err != nil {
				fmt.Println(err)
				os.Exit(128)
			}
			for i, b := range list {
				members[i] = b
			}
		} else {
			for i, b := range list {
				members[i] = string(b)
			}
		}

		data, _ := json.Marshal(members)
		fmt.Println(string(data))
	},
}

func init() {
	projectCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("verbose", "v", false, "显示项目详情")
}
