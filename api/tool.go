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
	"errors"
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
