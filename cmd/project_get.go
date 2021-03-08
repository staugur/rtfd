package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"tcw.im/rtfd/pkg/lib"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "显示文档项目信息",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		flagset := cmd.Flags()

		name := args[0]
		if name == "" {
			fmt.Println("invalid name")
			os.Exit(1)
		}
		build, err := flagset.GetBool("build")
		if err != nil {
			fmt.Printf("invalid param(build): %v\n", build)
			os.Exit(1)
		}
		var key string
		if strings.Count(name, ":") > 0 {
			build = false
			ns := strings.Split(name, ":")
			name = ns[0]
			key = ns[1]
			if key == "" {
				fmt.Printf("%s: invalid field\n", name)
				os.Exit(1)
			}
		}

		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(128)
		}

		var data []byte
		if build {
			bs, err := pm.GetNameWithBuilder(name)
			if err != nil {
				fmt.Println(err)
				os.Exit(127)
			}
			data, _ = json.Marshal(bs)
		} else {
			if key != "" {
				// rtfd p get {Name}:{Option}
				val, err := pm.GetNameOption(name, key)
				if err != nil {
					fmt.Println(err)
					os.Exit(130)
				}
				fmt.Println(val)
				os.Exit(0)
			} else {
				val, err := pm.GetSourceName(name)
				if err != nil {
					fmt.Println(err)
					os.Exit(129)
				}
				data = val
			}
		}

		fmt.Println(string(data))
	},
}

func init() {
	projectCmd.AddCommand(getCmd)
	getCmd.Flags().BoolP("build", "b", false, "是否显示构建结果")
}
