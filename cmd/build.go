package cmd

import (
	"fmt"

	"tcw.im/rtfd/pkg/build"
	"tcw.im/rtfd/vars"

	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "构建文档",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		branch := cmd.Flag("branch").Value.String()

		b, err := build.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = b.Build(name, branch, vars.CLISender)
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("branch", "b", "", "分支或标签")
}
