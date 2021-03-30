package cmd

import (
	"fmt"
	"os"

	"tcw.im/rtfd/pkg/lib"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:     "remove",
	Short:   "删除文档项目",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"r"},
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if name == "" {
			fmt.Println("invalid name")
			os.Exit(1)
		}

		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(127)
		}
		err = pm.Remove(name)
		if err != nil {
			fmt.Println(err)
			os.Exit(128)
		}
		fmt.Println("removed")

	},
}

func init() {
	projectCmd.AddCommand(removeCmd)
}
