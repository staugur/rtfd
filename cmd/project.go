package cmd

import (
	"github.com/spf13/cobra"
)

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "文档项目管理（可用别名p代替project）",
    Args:  cobra.MinimumNArgs(1),
    Aliases: []string{"p"},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
