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
	"encoding/json"
	"errors"
	"io"
	"strings"

	"pkg/tcw.im/rtfd/pkg/lib"
	"pkg/tcw.im/rtfd/pkg/util"

	"github.com/labstack/echo/v4"
	"pkg.tcw.im/gtc"
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
	sign := c.Request().Header.Get("X-Rtfd-Sign")
	return sign == gtc.MD5(opt.Secret), nil
}

// checkAPISecret 校验管理接口的全局密钥（配置文件 [api] secret），
// 未配置时管理接口不可用，避免Web管理能力被意外暴露
func checkAPISecret(c echo.Context) error {
	secret := pm.CFG().APISecret()
	if secret == "" {
		return errors.New("api secret is not configured, please set secret in [api] section")
	}
	if c.Request().Header.Get("X-Rtfd-Sign") != gtc.MD5(secret) {
		return errors.New("invalid sign")
	}
	return nil
}

// checkProjectSecret 校验项目级管理接口：全局密钥或项目密钥均可
func checkProjectSecret(c echo.Context) error {
	opt, err := pm.GetName(c.Param("name"))
	if err != nil {
		return err
	}
	sign := c.Request().Header.Get("X-Rtfd-Sign")
	secret := pm.CFG().APISecret()
	if secret != "" && sign == gtc.MD5(secret) {
		return nil
	}
	if opt.Secret != "" && sign == gtc.MD5(opt.Secret) {
		return nil
	}
	if secret == "" && opt.Secret == "" {
		return errors.New("secret is not configured, please set secret in [api] section or project")
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
