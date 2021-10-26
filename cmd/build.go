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
		flagset := cmd.Flags()
		isDebug, _ := flagset.GetBool("debug")
		isLog, _ := flagset.GetBool("log")

		b, err := build.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		if isDebug && isLog {
			err = b.BuildWithAll(name, branch, vars.CLISender)
		} else if isDebug {
			err = b.BuildWithDebug(name, branch, vars.CLISender)
		} else if isLog {
			err = b.BuildWithLog(name, branch, vars.CLISender)
		} else {
			err = b.Build(name, branch, vars.CLISender)
		}
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringP("branch", "b", "", "分支或标签")
	buildCmd.Flags().BoolP("debug", "", false, "使用调试模式运行构建")
	buildCmd.Flags().BoolP("log", "", false, "日志记录构建输出")
}
