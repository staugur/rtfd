package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// cfgCmd represents the cfg command
var cfgCmd = &cobra.Command{
	Use:   "cfg",
	Short: "查询配置文件的配置内容",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cfg called")
	},
}

func init() {
	rootCmd.AddCommand(cfgCmd)
}
