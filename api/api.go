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
	"fmt"

	"tcw.im/rtfd/assets"
	"tcw.im/rtfd/pkg/lib"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	pm      *lib.ProjectManager
	cfgFile string
)

// Start 启动web服务
func Start(host string, port uint, cfg string) {
	ipm, err := lib.New(cfg)
	if err != nil {
		panic(err)
	}
	pm = ipm
	cfgFile = cfg

	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = 5000
	}

	e := echo.New()
	e.HTTPErrorHandler = customHTTPErrorHandler
	g := e.Group("/rtfd", middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"X-Rtfd-Sign", "X-RTFD-SIGN"},
		AllowMethods: []string{"OPTIONS", "POST", "GET", "HEAD"},
	}))

	g.GET("/:name/desc", apiDesc)
	g.GET("/desc/:name", apiDesc)

	g.GET("/:name/badge", apiBadge)
	g.GET("/badge/:name", apiBadge)

	g.POST("/:name/build", apiBuild)
	g.POST("/build/:name", apiBuild)

	g.POST("/:name/webhook", webhookBuild)
	g.POST("/webhook/:name", webhookBuild)

	g.Match([]string{"HEAD", "GET"}, "/assets/rtfd.js", func(c echo.Context) error {
		return c.Blob(200, "application/javascript", assets.RtfdJS)
	})

	g.POST("/github/app", ghApp)

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))
}
