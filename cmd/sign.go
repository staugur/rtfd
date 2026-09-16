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
	"time"

	"pkg.tcw.im/rtfd/v2/pkg/conf"
	"pkg.tcw.im/rtfd/v2/pkg/util"

	"github.com/spf13/cobra"
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "生成 rtfd 管理接口的 HMAC-SHA256 签名头（便于调试 / curl 调用）",
	Long: `rtfd 管理/项目接口采用动态签名鉴权，需随请求携带 X-Rtfd-Ts、X-Rtfd-Nonce、X-Rtfd-Sign 三个头。
签名仅由 secret、时间戳、nonce 三者决定（HMAC-SHA256(secret, ts+"\n"+nonce)），与 method/path/body 无关。
secret 取自 rtfd.cfg 的 [api] secret（与程序本身一致，可用 --config 指定配置文件），无需额外传入。
本命令仅生成这三个请求头，供调试或在 curl 中自行拼接；签名不再依赖请求方法、路径或请求体。
示例：
  rtfd sign                        # 读取 rtfd.cfg 的 [api] secret，ts/nonce 自动生成
  rtfd sign --config /path/rtfd.cfg`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := conf.New(util.ExpandPath(cfgFile))
		if err != nil {
			fmt.Printf("无法读取配置 %s：%v\n请通过 --config 指定 rtfd.cfg 路径\n", cfgFile, err)
			os.Exit(1)
		}
		secret := cfg.APISecret()
		if secret == "" {
			fmt.Println("secret 为空：请在 rtfd.cfg 的 [api] secret 中配置")
			os.Exit(1)
		}

		ts := fmt.Sprintf("%d", time.Now().Unix())
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			fmt.Printf("生成 nonce 失败：%v\n", err)
			os.Exit(1)
		}
		nonce := hex.EncodeToString(buf)

		sign := util.SignAPIRequest(secret, ts, nonce)
		fmt.Println("X-Rtfd-Ts: " + ts)
		fmt.Println("X-Rtfd-Nonce: " + nonce)
		fmt.Println("X-Rtfd-Sign: " + sign)
	},
}

func init() {
	rootCmd.AddCommand(signCmd)
}
