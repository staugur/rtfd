package api

import (
	"errors"
	"strings"

	"tcw.im/rtfd/pkg/lib"
	"tcw.im/rtfd/pkg/util"

	"github.com/labstack/echo/v4"
	"tcw.im/gtc"
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
