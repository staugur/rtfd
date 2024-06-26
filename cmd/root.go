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

	"pkg/tcw.im/rtfd/assets"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"pkg.tcw.im/gtc"
)

var (
	// a global config file of rtfd, default is ~/.rtfd.cfg
	cfgFile string = os.Getenv("RTFD_CFG")

	// commitID is git commit hash when building
	commitID string
	// built is UTC time when building
	built string

	showVersion bool
	showVerbose bool
	newInit     bool
)

var rootCmd = &cobra.Command{
	Use:   "rtfd",
	Short: "Build, read your exclusive and fuck docs.",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if showVerbose {
			fmt.Printf("v%s commit/%s built/%s\n", assets.AppVersion, commitID, built)
		} else if showVersion {
			fmt.Println(assets.AppVersion)
		} else if newInit {
			//新增rtfd配置文件
			if !gtc.IsFile(cfgFile) {
				err := os.WriteFile(cfgFile, assets.RtfdCFG, 0644)
				if err != nil {
					fmt.Println("failed to generate configuration file")
					os.Exit(129)
				}
			} else {
				fmt.Printf("The rtfd config file(%s) already exists\n", cfgFile)
			}
		} else {
			cmd.Help()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	cfg, err := homedir.Expand("~/.rtfd.cfg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if cfgFile != "" {
		cfg = cfgFile
	}

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().StringVarP(
		&cfgFile, "config", "c", cfg, "rtfd配置文件",
	)
	rootCmd.Flags().BoolVarP(
		&showVersion, "version", "v", false, "显示版本",
	)
	rootCmd.Flags().BoolVarP(
		&showVerbose, "info", "i", false, "显示版本与构建信息",
	)
	rootCmd.Flags().BoolVarP(
		&newInit, "init", "", false, "初始化rtfd配置文件",
	)
}

func initConfig() {
	if showVersion || showVerbose || newInit {
		return
	}
	// 除 -h/help 和根命令 -v/-i/--init 选项外，其他子命令均需配置文件存在
	if cfgFile == "" || !gtc.IsFile(cfgFile) {
		fmt.Printf(
			"No valid configuration file: %s\n"+
				"Please use `rtfd --init` to initialize it.\n", cfgFile,
		)
		os.Exit(127)
	}
}
