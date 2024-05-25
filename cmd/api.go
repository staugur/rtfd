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
	"strconv"

	"pkg/tcw.im/rtfd/api"
	"pkg/tcw.im/rtfd/pkg/conf"

	"github.com/spf13/cobra"
)

// apiCmd represents the api command
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "运行API服务",
	Run: func(cmd *cobra.Command, args []string) {
		flagset := cmd.Flags()
		host, err := flagset.GetString("host")
		if err != nil {
			fmt.Printf("invalid param(host): %v\n", host)
			fmt.Println(err)
			os.Exit(1)
		}
		port, err := flagset.GetUint("port")
		if err != nil {
			fmt.Printf("invalid param(port): %v\n", port)
			fmt.Println(err)
			os.Exit(1)
		}

		c, err := conf.New(cfgFile)
		if err != nil {
			fmt.Println(err)
			return
		}
		if host == "" {
			host = c.GetKey("api", "host")
		}
		if port == 0 {
			p := c.GetKey("api", "port")
			pi, err := strconv.Atoi(p)
			if err != nil {
				fmt.Printf("invalid param(port): %v\n", port)
				fmt.Println(err)
				os.Exit(1)
			}
			port = uint(pi)
		}

		api.Start(host, port, cfgFile)
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().StringP("host", "", "", "Api监听地址")
	apiCmd.Flags().UintP("port", "", 0, "Api监听端口")
}
