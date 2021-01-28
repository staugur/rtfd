package api

import (
	"fmt"
	"io/ioutil"

	"tcw.im/rtfd/pkg/lib"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rakyll/statik/fs"
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

	js, err := getRtfdJS()
	if err != nil {
		panic(err)
	}

	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = 5000
	}

	e := echo.New()
	g := e.Group("/rtfd")
	g.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"X-Rtfd-Sign"},
		AllowMethods: []string{"OPTIONS", "POST", "GET"},
	}))

	g.GET("/:name/desc", apiDesc)
	g.POST("/:name/build", apiBuild)
	g.POST("/:name/webhook", webhookBuild)
	g.Match([]string{"HEAD", "GET"}, "/assets/rtfd.js", func(c echo.Context) error {
		return c.Blob(200, "application/javascript", js)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))
}

func getRtfdJS() (contents []byte, err error) {
	statikFS, err := fs.New()
	if err != nil {
		return
	}
	r, err := statikFS.Open("/rtfd.js")
	if err != nil {
		return
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}
