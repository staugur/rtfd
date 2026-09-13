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

package api

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"pkg.tcw.im/rtfd/v2/pkg/lib"
	"pkg.tcw.im/rtfd/v2/pkg/util"

	"github.com/labstack/echo/v4"
)

func getArg(c echo.Context, key string) string {
	val := c.FormValue(key)
	if val == "" {
		val = c.QueryParam(key)
	}
	return val
}

// getFormParams 获取请求参数，支持JSON（body）、表单与query串，键统一转为小写
func getFormParams(c echo.Context) (map[string]any, error) {
	params := make(map[string]any)
	ctype := c.Request().Header.Get(echo.HeaderContentType)
	if strings.HasPrefix(ctype, echo.MIMEApplicationJSON) {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if err := json.Unmarshal(body, &params); err != nil {
				return nil, err
			}
		}
	} else {
		form, err := c.FormParams()
		if err != nil {
			return nil, err
		}
		for k, v := range form {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}
	lower := make(map[string]any, len(params))
	for k, v := range params {
		lower[strings.ToLower(k)] = v
	}
	return lower, nil
}

func getBaseURL(c echo.Context) string {
	p := c.Scheme()
	r := c.Request()
	h := r.Host
	erase := ":80"
	if p == "https" {
		erase = ":443"
	}
	h = strings.ReplaceAll(h, erase, "")
	return p + "://" + h
}

// 签名相关常量与防重放缓存
const signTSWindow = 300 // 签名时间戳允许偏差（秒）

var signNonceCache = &nonceCache{m: make(map[string]int64)}

// nonceCache 记录已使用的时间戳窗口内的 nonce，防止同一签名被重放
type nonceCache struct {
	mu sync.Mutex
	m  map[string]int64 // nonce -> 过期时间戳（Unix 秒）
}

func (n *nonceCache) seen(nonce string, ts int64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now().Unix()
	for k, exp := range n.m {
		if exp < now {
			delete(n.m, k)
		}
	}
	if exp, ok := n.m[nonce]; ok && exp >= now {
		return true
	}
	n.m[nonce] = ts + signTSWindow
	return false
}

// parseSign 解析并预校验签名头：必填项、时间戳在 ±5 分钟内、nonce 未重放（仅校验一次，便于多密钥尝试复用）
func parseSign(c echo.Context) (ts, nonce, sign string, err error) {
	ts = c.Request().Header.Get("X-Rtfd-Ts")
	nonce = c.Request().Header.Get("X-Rtfd-Nonce")
	sign = c.Request().Header.Get("X-Rtfd-Sign")
	if ts == "" || nonce == "" || sign == "" {
		return "", "", "", errors.New("missing sign headers (X-Rtfd-Ts/X-Rtfd-Nonce/X-Rtfd-Sign)")
	}
	tsVal, e := strconv.ParseInt(ts, 10, 64)
	if e != nil {
		return "", "", "", errors.New("invalid X-Rtfd-Ts")
	}
	now := time.Now().Unix()
	diff := now - tsVal
	if diff > signTSWindow || diff < -signTSWindow {
		return "", "", "", errors.New("expired sign (X-Rtfd-Ts out of range)")
	}
	if signNonceCache.seen(nonce, tsVal) {
		return "", "", "", errors.New("replay sign (X-Rtfd-Nonce reused)")
	}
	return ts, nonce, sign, nil
}

// matchSign 用给定密钥计算签名并与请求签名做恒定时间比对
func matchSign(secret, ts, nonce, sign string) bool {
	expect := util.SignAPIRequest(secret, ts, nonce)
	return subtle.ConstantTimeCompare([]byte(sign), []byte(expect)) == 1
}

// verifyAPISign 校验 rtfd 管理/项目接口的动态签名（单一密钥）：
// 请求需携带 X-Rtfd-Ts（Unix 秒）、X-Rtfd-Nonce（随机串）、X-Rtfd-Sign（HMAC-SHA256）。
// 签名串 = ts + "\n" + nonce（以 secret 为 HMAC 密钥），与 pkg/util.SignAPIRequest 保持一致。
func verifyAPISign(secret string, c echo.Context) error {
	ts, nonce, sign, err := parseSign(c)
	if err != nil {
		return err
	}
	if !matchSign(secret, ts, nonce, sign) {
		return errors.New("invalid sign")
	}
	return nil
}

// checkSecret 校验项目级密钥（构建接口等）：项目未设 secret 时免鉴权；
// 设置后按动态签名校验
func checkSecret(c echo.Context) (bool, error) {
	name := c.Param("name")
	opt, err := pm.GetName(name)
	if err != nil {
		return false, err
	}
	if opt.Secret == "" {
		// 无密码表示直接成功
		return true, nil
	}
	ts, nonce, sign, err := parseSign(c)
	if err != nil {
		return false, err
	}
	if !matchSign(opt.Secret, ts, nonce, sign) {
		return false, errors.New("invalid sign")
	}
	return true, nil
}

// checkAPISecret 校验管理接口的全局密钥（配置文件 [api] secret），
// 未配置时管理接口不可用，避免Web管理能力被意外暴露
func checkAPISecret(c echo.Context) error {
	secret := pm.CFG().APISecret()
	if secret == "" {
		return errors.New("api secret is not configured, please set secret in [api] section")
	}
	ts, nonce, sign, err := parseSign(c)
	if err != nil {
		return err
	}
	if !matchSign(secret, ts, nonce, sign) {
		return errors.New("invalid sign")
	}
	return nil
}

// checkProjectSecret 校验项目级管理接口：全局密钥或项目密钥均可（nonce 仅校验一次）
func checkProjectSecret(c echo.Context) error {
	opt, err := pm.GetName(c.Param("name"))
	if err != nil {
		return err
	}
	secret := pm.CFG().APISecret()
	if secret == "" && opt.Secret == "" {
		return errors.New("secret is not configured, please set secret in [api] section or project")
	}
	ts, nonce, sign, err := parseSign(c)
	if err != nil {
		return err
	}
	if secret != "" && matchSign(secret, ts, nonce, sign) {
		return nil
	}
	if opt.Secret != "" && matchSign(opt.Secret, ts, nonce, sign) {
		return nil
	}
	return errors.New("invalid sign")
}

func checkGitHubWebhook(c echo.Context, opt lib.Options, Body []byte) error {
	if opt.Secret == "" {
		// 无密码表示直接成功
		return nil
	}

	GHSignV := c.Request().Header.Get("X-Hub-Signature")
	if GHSignV == "" {
		return errors.New("empty signature")
	}
	GHSignS := strings.Split(GHSignV, "=")
	if GHSignS[0] != "sha1" {
		return errors.New("invalid signature method")
	}
	if util.HMACSha1Byte([]byte(opt.Secret), Body) == GHSignS[1] {
		return nil
	}
	return errors.New("verify signature failed")
}

func checkGiteeWebhook(c echo.Context, opt lib.Options) error {
	if opt.Secret == "" {
		// 无密码表示直接成功
		return nil
	}

	Token := c.Request().Header.Get("X-Gitee-Token")
	if Token == "" {
		return errors.New("empty token")
	}
	if opt.Secret == Token {
		return nil
	}
	return errors.New("verify signature failed")
}

func badgeRes(c echo.Context, status string) error {
	c.Response().Header().Set(echo.HeaderContentType, "image/svg+xml; charset=utf-8")
	return c.String(200, status)
}
