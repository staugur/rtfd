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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"pkg/tcw.im/rtfd/pkg/lib"
	"pkg/tcw.im/rtfd/vars"
)

var transferDesc = `转储（导入、导出）文档项目

可以使用此之命令在一台服务器上将项目配置导出为 base64 编码的字符串，
在另一台服务器上导入，或者在本地导入（相当于复制项目，需要设置别名）。

导出：

    $ rtfd p t -e <NAME>
    // Output: base64-encoded

导入：

    $ rtfd p t -i <base64-encoded>

    // 因为导出选项包含名称，如果导入时rtfd已经有此名称则会失败，
    // 此时可以设置别名覆盖原名称。
    $ rtfd p t -i <base64-encoded> <New-Name>
    // Output: imported (if success)
`

// transferCmd represents the transfer command
var transferCmd = &cobra.Command{
	Use:     "transfer",
	Short:   "转储（导入、导出）文档项目",
	Long:    transferDesc,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"t"},
	Run: func(cmd *cobra.Command, args []string) {
		flagset := cmd.Flags()
		EX, err := flagset.GetBool("export")
		if err != nil {
			fmt.Printf("invalid param(export): %v\n", EX)
			fmt.Println(err)
			os.Exit(1)
		}
		IM := cmd.Flag("import").Value.String()
		IMdebug, err := flagset.GetBool("import-debug")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// to import or export
		pm, err := lib.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(127)
		}
		if EX {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}
			name := args[0]
			if name == "" {
				fmt.Println("empty name")
				os.Exit(1)
			}
			opt, err := pm.GetName(name)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if esm, _ := flagset.GetBool("export-sys-meta"); !esm {
				meta := opt.Meta
				if meta == nil {
					meta = make(map[string]string)
				}
				for k := range meta {
					if strings.HasPrefix(k, "_") {
						err = opt.UpdateMeta(k, vars.ResetEmpty)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
					}
				}
			}
			val, err := json.Marshal(opt)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			encode := base64.StdEncoding.EncodeToString(val)
			fmt.Println(encode)
		} else {
			if IM == "" {
				cmd.Help()
				os.Exit(1)
			}
			IMjson, err := base64.StdEncoding.DecodeString(IM)
			if err != nil {
				fmt.Println("import decode fail")
				fmt.Println(err)
				os.Exit(129)
			}
			if IMdebug {
				fmt.Println(string(IMjson))
				os.Exit(0)
			}

			//ready to create a new project
			var opt lib.Options
			err = json.Unmarshal(IMjson, &opt)
			if err != nil {
				fmt.Println(err)
				os.Exit(128)
			}
			name := opt.Name
			if len(args) > 0 {
				// override option Name
				name = args[0]
				opt.Name = name
			}

			if pm.HasName(name) {
				fmt.Println("the name already exists")
				fmt.Println("but you can overwrite it: rtfd p t -i <BASE64> <Name>")
				os.Exit(128)
			}

			// override default option
			dn := pm.CFG().GetKey("nginx", "dn")
			if dn == "" {
				panic("invalid nginx dn")
			}
			opt.DefaultDomain = name + "." + dn

			err = pm.Create(name, opt)
			if err != nil {
				fmt.Println(err)
				os.Exit(130)
			}
			fmt.Println("imported")
		}
	},
}

func init() {
	projectCmd.AddCommand(transferCmd)
	transferCmd.Flags().BoolP("export", "e", false, "导出（格式为 base64 编码）项目")
	transferCmd.Flags().BoolP("export-sys-meta", "", false, "是否导出Meta内置字段")
	transferCmd.Flags().StringP("import", "i", "", "导入（格式为 base64 编码）项目")
	transferCmd.Flags().BoolP("import-debug", "d", false, "不导入项目，仅查看选项")
}
