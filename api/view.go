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
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"tcw.im/rtfd/pkg/build"
	"tcw.im/rtfd/pkg/lib"
	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"

	"github.com/labstack/echo/v4"
	"tcw.im/ufc"
)

type res struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type resb struct {
	res
	Branch string `json:"branch"`
}
type resp struct {
	res
	Ping string `json:"ping"`
}
type resd struct {
	res
	Data map[string]interface{} `json:"data"`
}

func customHTTPErrorHandler(err error, c echo.Context) {
	code := 200
	msg := err.Error()
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = he.Message.(string)
	}
	c.JSON(code, res{false, msg})
}

func apiDesc(c echo.Context) error {
	name := c.Param("name")
	if !pm.HasName(name) {
		return c.JSON(200, res{Message: "Not Found"})
	}
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}

	data := make(map[string]interface{})
	if opt.IsPublic {
		data["url"] = opt.URL
	} else {
		urlpub, _ := util.PublicGitURL(opt.URL)
		data["url"] = urlpub
	}
	data["lang"] = strings.Split(opt.Lang, ",")
	data["latest"] = opt.Latest
	if util.IsDomain(opt.CustomDomain) {
		data["dn"] = opt.CustomDomain
	} else {
		data["dn"] = false
	}
	data["sourceDir"] = opt.SourceDir
	data["single"] = opt.Single
	data["builder"] = opt.Builder
	data["showNav"] = opt.ShowNav
	if opt.Builder != "html" {
		data["hideGit"] = true
	} else {
		data["hideGit"] = opt.HideGit
	}
	data["icon"] = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAAlUlEQVQ4T92S0Q0CMQxDnydBtwEbABvcRjAKK7DBscGNwCZGRbSKDigB/uhv4lc7svjxqeptj8AeWL9hTpJ2dScCLsAqY0hS00WA7+ITcJA0p2AhQgUMwBHYdAAtxoODYs92hb1k1BhdQMy6hKYAvRukANHB8lYpwB84+DTCVMrzdQ/ib7ZvsI6Ds6RtmbciZXr/bOcKjCNuESAd+XoAAAAASUVORK5CYII="
	data["public"] = opt.IsPublic
	data["gsp"] = opt.GSP
	basedir := pm.CFG().BaseDir()
	if basedir == "" || !ufc.IsDir(basedir) {
		return c.JSON(200, res{Message: "invalid data directory"})
	}
	versions := make(map[string][]string)
	for _, lang := range strings.Split(opt.Lang, ",") {
		langDir := filepath.Join(basedir, "docs", name, lang)
		if ufc.IsDir(langDir) {
			ifs, err := ioutil.ReadDir(langDir)
			if err != nil {
				continue
			}
			vs := []string{"latest"}
			for _, f := range ifs {
				name := f.Name()
				if f.IsDir() && name != "" && name != "." && name != ".." {
					vs = append(vs, name)
				}
			}
			if vs != nil {
				if len(vs) == 1 && vs[0] == "latest" {
					continue
				}
				versions[lang] = vs
			}
		}
	}
	data["versions"] = versions
	return c.JSON(200, resd{res{Success: true}, data})
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
	var gst, event string

	H := c.Request().Header
	agent := H.Get("User-Agent")
	if strings.HasPrefix(agent, "GitHub-Hookshot") {
		gst = vars.GSPGitHub
		event = H.Get("X-GitHub-Event")
	} else if agent == "git-oschina-hook" {
		gst = vars.GSPGitee
		evt := H.Get("X-Gitee-Event")
		ping := H.Get("X-Gitee-Ping")
		if ufc.IsTrue(ping) {
			event = "ping"
		} else if evt == "Push Hook" {
			event = "push"
		} else if evt == "Tag Push Hook" {
			event = "release"
		}
	} else {
		return errors.New("unsupported provider")
	}
	if event == "" {
		return errors.New("invalid event type")
	}
	if event == "ping" {
		return c.JSON(200, resp{res{Success: true}, "pong"})
	}
	if !ufc.StrInSlice(event, []string{"ping", "push", "release"}) {
		return errors.New("unsupported webhook event")
	}

	name := c.Param("name")
	branch := ""
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}

	var body map[string]interface{}
	RawBody, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(RawBody, &body); err != nil {
		return err
	}
	b, err := build.New(cfgFile)
	if err != nil {
		return err
	}

	if gst == vars.GSPGitHub {
		if err := checkGitHubWebhook(c, opt, RawBody); err != nil {
			return err
		}
		if event == "push" {
			ref := strings.Split(body["ref"].(string), "/")
			branch = ref[len(ref)-1]
		} else {
			action := body["action"].(string)
			if action == "released" {
				release := body["release"].(map[string]interface{})
				branch = release["tag_name"].(string)
			} else {
				return errors.New("the action is ignored in the release event")
			}
		}
	} else if gst == vars.GSPGitee {
		if err := checkGiteeWebhook(c, opt); err != nil {
			return err
		}
		ref := strings.Split(body["ref"].(string), "/")
		branch = ref[len(ref)-1]
	} else {
		return errors.New("unsupported git service provider")
	}

	sep := opt.GetMeta("excluded_sep")
	if sep == "" {
		sep = opt.MustMeta("_sep", "|")
	}
	if ufc.StrInSlice(branch, strings.Split(opt.GetMeta("excluded_branch"), sep)) {
		return c.JSON(200, resb{res{false, "excluded branch"}, branch})
	}

	go b.Build(name, branch, vars.WebhookSender)
	return c.JSON(201, resb{res{Success: true}, branch})
}

func apiBadge(c echo.Context) error {
	passing := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="86" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="86" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#4c1" d="M35 0h51v20H35z"/><path fill="url(#b)" d="M0 0h86v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="595" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="410">passing</text><text x="595" y="140" transform="scale(.1)" textLength="410">passing</text></g> </svg>`
	failing := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="78" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="78" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#e05d44" d="M35 0h43v20H35z"/><path fill="url(#b)" d="M0 0h78v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="555" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="330">failing</text><text x="555" y="140" transform="scale(.1)" textLength="330">failing</text></g> </svg>`
	unknown := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="96" height="20"><linearGradient id="b" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient><clipPath id="a"><rect width="96" height="20" rx="3" fill="#fff"/></clipPath><g clip-path="url(#a)"><path fill="#555" d="M0 0h35v20H0z"/><path fill="#dfb317" d="M35 0h61v20H35z"/><path fill="url(#b)" d="M0 0h96v20H0z"/></g><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110"><text x="185" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="250">docs</text><text x="185" y="140" transform="scale(.1)" textLength="250">docs</text><text x="645" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="510">unknown</text><text x="645" y="140" transform="scale(.1)" textLength="510">unknown</text></g> </svg>`

	name := c.Param("name")
	branch := strings.ToLower(getArg(c, "branch"))
	status := ""

	if !pm.HasName(name) {
		return badgeRes(c, unknown)
	}

	if branch == "" || branch == "latest" {
		opt, err := pm.GetName(name)
		if err != nil {
			return err
		}
		branch = opt.Latest
	}

	builder, err := pm.GetBuilder(name, branch)
	if err != nil {
		if strings.HasPrefix(err.Error(), "not found branch") {
			return badgeRes(c, unknown)
		}
		return err
	}
	if status == "" {
		if builder.Status {
			status = passing
		} else {
			status = failing
		}
	}
	return badgeRes(c, status)
}

func ghApp(c echo.Context) error {
	H := c.Request().Header
	evt := H.Get("X-GitHub-Event")
	hiti := H.Get("X-GitHub-Hook-Installation-Target-ID")
	hitt := H.Get("X-GitHub-Hook-Installation-Target-Type")
	if !ufc.StrInSlice(evt, []string{"installation", "installation_repositories"}) {
		return errors.New("unsupported event")
	}
	if hitt != "integration" {
		return errors.New("unsupported type")
	}
	hitiU, err := strconv.ParseUint(hiti, 10, 64)
	if err != nil {
		return err
	}

	gh, err := lib.NewGHApp(pm)
	if err != nil {
		return err
	}
	gh.BaseURL(getBaseURL(c))

	var data lib.AppWebhook
	if err := c.Bind(&data); err != nil {
		return err
	}
	if hitiU != data.Installation.AppID {
		return errors.New("not match installation app")
	}

	err = gh.Dispatch(data)
	if err != nil {
		return err
	}
	return c.JSON(200, res{Success: true})
}
