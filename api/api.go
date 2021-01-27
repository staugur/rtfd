package api

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

// Start 启动web服务
func Start(host string, port uint) {
	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = 5000
	}

	e := echo.New()
	e.GET("/", index)
	e.GET("/assets/rtfd.js", js)

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))
}
