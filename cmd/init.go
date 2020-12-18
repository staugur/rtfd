package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化rtfd配置文件",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
        yes, err := cmd.Flags().GetBool("yes")
        if err != nil {
            fmt.Println(err)
            os.Exit(1)
        }
        if yes {
            renderConfig()
        }
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("yes", "y", false, "确定要初始化rtfd吗？")
}

func renderConfig() {
    fmt.Println("render ok")
}