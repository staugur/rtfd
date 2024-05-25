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
	"fmt"
	"os"

	"pkg/tcw.im/rtfd/pkg/lib"

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
