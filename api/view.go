package api

import (
	"io/ioutil"

	"github.com/labstack/echo/v4"
	"github.com/rakyll/statik/fs"
)

func index(c echo.Context) error {
	return c.JSONBlob(200, []byte(`"hello world"`))
}

func js(c echo.Context) error {
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	r, err := statikFS.Open("/rtfd.js")
	if err != nil {
		return err
	}
	defer r.Close()
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return c.Blob(200, "application/javascript", contents)
}
