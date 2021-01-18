package cmd

import (
	"fmt"
	"os"

	"tcw.im/rtfd/pkg/lib"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "显示文档项目信息",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if name == "" {
			fmt.Println("invalid name")
			os.Exit(127)
		}
		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(128)
		}
		defer pm.Close()
		val, err := pm.GetName(name)
		if err != nil {
			fmt.Println(err)
			os.Exit(129)
		}
		fmt.Println(val)

	},
}

func init() {
	projectCmd.AddCommand(getCmd)
}
