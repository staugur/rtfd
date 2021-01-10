package cmd

import (
	"fmt"

	"rtfd/internal/build"

	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "构建文档",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("build called")
		b, err := build.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		b.build()
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("name", "n", "", "名称")
	buildCmd.Flags().StringP("branch", "b", "", "分支")
}
