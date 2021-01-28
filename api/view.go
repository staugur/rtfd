package api

import (
	"tcw.im/rtfd/pkg/build"
	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"

	"github.com/labstack/echo/v4"
)

type res struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type resb struct {
	res
	Branch string `json:"branch"`
}

type resd struct {
	res
	Data string `json:"data"`
}

func getArg(c echo.Context, key string) string {
	val := c.FormValue(key)
	if val == "" {
		val = c.QueryParam(key)
	}
	return val
}

func checkSecret(c echo.Context) (bool, error) {
	name := c.Param("name")
	opt, err := pm.GetName(name)
	if err != nil {
		return false, err
	}
	if opt.Secret == "" {
		return true, nil
	}
	sign := c.Request().Header.Get("X-Rtfd-Sign")
	return sign == util.MD5(opt.Secret), nil
}

func apiDesc(c echo.Context) error {
	name := c.Param("name")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	return c.JSON(200, res{Success: true})
}

func apiBuild(c echo.Context) error {
	if ok, err := checkSecret(c); !ok {
		return err
	}
	name := c.Param("name")
	branch := getArg(c, "branch")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	b, err := build.New(cfgFile)
	if err != nil {
		return err
	}
	go b.Build(name, branch, vars.APISender)
	return c.JSON(201, resb{res{Success: true}, branch})
}

func webhookBuild(c echo.Context) error {
	if ok, err := checkSecret(c); !ok {
		return err
	}
	name := c.Param("name")
	branch := getArg(c, "branch")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	b, err := build.New(cfgFile)
	if err != nil {
		return err
	}
	go b.Build(name, branch, vars.WebhookSender)
	return c.JSON(201, resb{res{Success: true}, branch})
}
