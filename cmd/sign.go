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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"pkg.tcw.im/rtfd/v2/pkg/conf"
	"pkg.tcw.im/rtfd/v2/pkg/util"

	"github.com/spf13/cobra"
)

var (
	signSecret   string
	signMethod   string
	signPath     string
	signBody     string
	signBodyFile string
	signTs       string
	signNonce    string
	signBaseURL  string
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "生成 rtfd 管理接口的 HMAC-SHA256 签名头（便于调试 / curl 调用）",
	Long: `rtfd 管理/项目接口采用动态签名鉴权，需随请求携带 X-Rtfd-Ts、X-Rtfd-Nonce、X-Rtfd-Sign 三个头。
签名仅由 secret、时间戳、nonce 三者决定（HMAC-SHA256(secret, ts+"\n"+nonce)），与 method/path/body 无关。
本命令输出可直接粘贴的请求头，或（提供 --url 时）输出完整 curl 命令；其中 --method/--path/--body 仅用于拼装 curl，不参与签名计算。
示例：
  rtfd sign --secret 'yoursecret' --method POST --path /rtfd/projects --body 'name=Demo&url=https://github.com/staugur/rtfd'
  rtfd sign --method GET --path /rtfd/projects --url https://docs.example.com
  rtfd sign --secret 'yoursecret' --method POST --path /rtfd/demo/update --body-file ./req.txt --url https://docs.example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		secret := signSecret
		if secret == "" {
			cfg, err := conf.New(util.ExpandPath(cfgFile))
			if err != nil {
				fmt.Printf("无法读取配置 %s：%v\n请使用 --secret 直接传入密钥\n", cfgFile, err)
				os.Exit(1)
			}
			secret = cfg.APISecret()
		}
		if secret == "" {
			fmt.Println("secret 为空：请通过 --secret 传入，或在 rtfd.cfg 的 [api] secret 中配置")
			os.Exit(1)
		}

		method := strings.ToUpper(signMethod)
		if method == "" {
			method = "GET"
		}
		path := signPath
		if path == "" {
			path = "/rtfd/projects"
		}

		var body []byte
		switch {
		case signBodyFile != "":
			f := strings.TrimPrefix(signBodyFile, "@")
			b, err := os.ReadFile(f)
			if err != nil {
				fmt.Printf("读取 body 文件失败：%v\n", err)
				os.Exit(1)
			}
			body = b
		case strings.HasPrefix(signBody, "@"):
			b, err := os.ReadFile(signBody[1:])
			if err != nil {
				fmt.Printf("读取 body 文件失败：%v\n", err)
				os.Exit(1)
			}
			body = b
		default:
			body = []byte(signBody)
		}

		ts := signTs
		if ts == "" {
			ts = fmt.Sprintf("%d", time.Now().Unix())
		}
		nonce := signNonce
		if nonce == "" {
			buf := make([]byte, 8)
			if _, err := rand.Read(buf); err != nil {
				fmt.Printf("生成 nonce 失败：%v\n", err)
				os.Exit(1)
			}
			nonce = hex.EncodeToString(buf)
		}

		sign := util.SignAPIRequest(secret, ts, nonce)
		fmt.Println("X-Rtfd-Ts: " + ts)
		fmt.Println("X-Rtfd-Nonce: " + nonce)
		fmt.Println("X-Rtfd-Sign: " + sign)

		if signBaseURL != "" {
			curl := fmt.Sprintf("curl -X %s '%s%s'", method, strings.TrimRight(signBaseURL, "/"), path)
			curl += fmt.Sprintf(" -H 'X-Rtfd-Ts: %s' -H 'X-Rtfd-Nonce: %s' -H 'X-Rtfd-Sign: %s'", ts, nonce, sign)
			if len(body) > 0 {
				curl += fmt.Sprintf(" -d '%s'", strings.ReplaceAll(string(body), "'", "'\\''"))
			}
			fmt.Println("\n# 等效 curl 命令：")
			fmt.Println(curl)
		}
	},
}

func init() {
	signCmd.Flags().SortFlags = false
	signCmd.Flags().StringVar(&signSecret, "secret", "", "签名密钥；缺省从 --config 指定的 rtfd.cfg 的 [api] secret 读取")
	signCmd.Flags().StringVar(&signMethod, "method", "GET", "HTTP 方法（GET/POST/PUT/DELETE）")
	signCmd.Flags().StringVar(&signPath, "path", "/rtfd/projects", "请求路径（不含查询串）")
	signCmd.Flags().StringVar(&signBody, "body", "", "请求体原文（或以 @ 开头表示文件）")
	signCmd.Flags().StringVar(&signBodyFile, "body-file", "", "请求体文件路径（等价于 --body @文件）")
	signCmd.Flags().StringVar(&signTs, "ts", "", "时间戳（Unix 秒），缺省取当前时间")
	signCmd.Flags().StringVar(&signNonce, "nonce", "", "随机串，缺省随机生成")
	signCmd.Flags().StringVar(&signBaseURL, "url", "", "服务根地址，提供后额外输出完整 curl 命令")

	rootCmd.AddCommand(signCmd)
}
