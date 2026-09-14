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
	"net/http"

	"pkg.tcw.im/rtfd/v2/assets"
	_ "pkg.tcw.im/rtfd/v2/docs" // swag 生成的接口文档（注册到 swag 注册表，由 /rtfd/docs 提供）
	"pkg.tcw.im/rtfd/v2/pkg/build"
	"pkg.tcw.im/rtfd/v2/pkg/lib"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// swaggerIndex 提供自定义 Swagger UI 首页：注入 requestInterceptor 自动为请求签名
func swaggerIndex(c echo.Context) error {
	return c.HTML(http.StatusOK, string(assets.SwaggerIndexHTML))
}

var (
	pm      *lib.ProjectManager
	bld     *build.Builder
	cfgFile string
)

// New 初始化API服务：加载配置、创建项目管理器与构建器并注册路由
func New(cfg string) (*echo.Echo, error) {
	ipm, err := lib.New(cfg)
	if err != nil {
		return nil, err
	}
	pm = ipm
	cfgFile = cfg
	// 启动即生成 Caddyfile（暂无项目时也生成），避免 Caddy 因配置缺失反复启动失败
	ipm.InitCaddy()
	// 构建器复用同一个项目管理器，避免每次构建都新建数据库连接
	bld, err = build.NewFromPM(cfg, ipm)
	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = customHTTPErrorHandler

	g := e.Group("/rtfd", middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"X-Rtfd-Sign", "X-Rtfd-Ts", "X-Rtfd-Nonce", "X-RTFD-SIGN", echo.HeaderContentType},
		AllowMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPost,
			http.MethodPut, http.MethodDelete, http.MethodOptions,
		},
	}))
	registerRoutes(g)
	return e, nil
}

// Start 启动web服务
func Start(host string, port uint, cfg string) {
	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = 5000
	}
	e, err := New(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Rtfd API is listening on %s:%d\n", host, port)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))
}

// registerRoutes 注册路由，兼容两种路径风格（:name 在前或在后），
// 以便管理端与文档页面挂件使用同一套接口
func registerRoutes(g *echo.Group) {
	// 公开接口：文档页面挂件数据、状态徽章与静态资源
	g.GET("/:name/desc", apiDesc)
	g.GET("/desc/:name", apiDesc)
	g.GET("/:name/badge", apiBadge)
	g.GET("/badge/:name", apiBadge)
	g.Match([]string{"HEAD", "GET"}, "/assets/rtfd.js", func(c echo.Context) error {
		return c.Blob(200, "application/javascript; charset=utf-8", assets.RtfdJS)
	})
	g.POST("/github/app", ghApp)

	// 构建触发与git webhook
	g.POST("/:name/build", apiBuild)
	g.POST("/build/:name", apiBuild)
	g.POST("/:name/webhook", webhookBuild)
	g.POST("/webhook/:name", webhookBuild)

	// 项目管理：与CLI的project子命令对应，需密钥鉴权
	g.GET("/projects", apiProjectList)
	g.POST("/projects", apiProjectCreate)
	g.GET("/:name/info", apiProjectInfo)
	g.GET("/info/:name", apiProjectInfo)
	g.POST("/:name/update", apiProjectUpdate)
	g.POST("/update/:name", apiProjectUpdate)
	g.POST("/:name/remove", apiProjectRemove)
	g.POST("/remove/:name", apiProjectRemove)
	g.DELETE("/:name/remove", apiProjectRemove)
	g.DELETE("/remove/:name", apiProjectRemove)
	g.GET("/:name/export", apiProjectExport)
	g.GET("/export/:name", apiProjectExport)
	g.POST("/import", apiProjectImport)

	// 接口文档（Swagger UI）：/rtfd/docs/index.html，规范文件 /rtfd/docs/doc.json（或 doc.yaml）
	// 文档由 `make docs` 依据代码注解生成到 docs/，仅提供服务端已编译的接口元数据，无需鉴权
	g.GET("/docs", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/rtfd/docs/index.html")
	})
	// 自定义 index：注入 requestInterceptor，按 rtfd 动态签名规则自动为请求签名（便于调试）
	g.GET("/docs/index.html", swaggerIndex)
	g.GET("/docs/*", echoSwagger.WrapHandler)
}
