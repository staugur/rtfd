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
	"regexp"
	"strings"

	"pkg.tcw.im/rtfd/v2/assets"
	"pkg.tcw.im/rtfd/v2/pkg/util"

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
			// 初始化 rtfd 配置文件；可用环境变量预填必填项
			if !gtc.IsFile(cfgFile) {
				content, notes := fillConfigFromEnv(assets.RtfdCFG)
				if err := os.WriteFile(cfgFile, content, 0644); err != nil {
					fmt.Println("failed to generate configuration file")
					os.Exit(129)
				}
				fmt.Printf("Generated rtfd config file: %s\n", cfgFile)
				if notes != "" {
					fmt.Print(notes)
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

	cfg := util.ExpandPath("~/.rtfd.cfg")
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
	// sign 子命令可不依赖配置文件（直接 --secret 传入密钥）
	for _, a := range os.Args[1:] {
		if a == "sign" {
			return
		}
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

// fillConfigFromEnv 依据环境变量预填配置模板中的必填项，返回填充后的内容与提示信息。
// 仅当对应环境变量非空时才替换；未提供则保持模板中的空值（需手动补填）。
//
//	RTFD_API_SERVER_URL -> [api] server_url（对外服务地址，必填）
//	RTFD_SWS_DN       -> [sws] dn（文档托管域名后缀，必填）
func fillConfigFromEnv(content []byte) ([]byte, string) {
	replacers := []struct {
		env string
		key string
	}{
		{"RTFD_API_SERVER_URL", "server_url"},
		{"RTFD_SWS_DN", "dn"},
	}
	s := string(content)
	var filled, missing []string
	for _, r := range replacers {
		v := strings.TrimSpace(os.Getenv(r.env))
		if v == "" {
			missing = append(missing, r.key)
			continue
		}
		re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(r.key) + `\s*=.*$`)
		s = re.ReplaceAllString(s, r.key+" = "+v)
		filled = append(filled, r.key)
	}
	notes := ""
	if len(filled) > 0 {
		notes += "已从环境变量填入: " + strings.Join(filled, ", ") + "\n"
	}
	if len(missing) > 0 {
		notes += "以下必填项未提供环境变量，请在配置文件中手动设置: " + strings.Join(missing, ", ") + "\n"
	}
	return []byte(s), notes
}
