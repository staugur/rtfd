/*
   Copyright 2021 Hiroshi.tao

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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
	Use:     "list",
	Short:   "列出所有文档项目信息",
	Aliases: []string{"l"},
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
		if verbose {
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
