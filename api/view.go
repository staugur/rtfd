package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"tcw.im/rtfd/pkg/build"
	"tcw.im/rtfd/vars"
	"tcw.im/ufc"

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
type resp struct {
	res
	Ping string `json:"ping"`
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
	var gst, event string

	H := c.Request().Header
	agent := H.Get("User-Agent")
	if strings.HasPrefix(agent, "GitHub-Hookshot") {
		gst = "github"
		event = H.Get("X-GitHub-Event")
	} else if agent == "git-oschina-hook" {
		gst = "gitee"
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
		return errors.New("unsupported webhook event")
	}
	if !ufc.StrInSlice(event, []string{"ping", "push", "release"}) {
		return errors.New("invalid event type")
	}
	if event == "ping" {
		return c.JSON(200, resp{res{Success: true}, "pong"})
	}

	name := c.Param("name")
	branch := ""
	opt, err := pm.GetName(name)
	if err != nil {
		return err
	}
	fmt.Println(name, gst, event)

	var body map[string]interface{}
	RawBody, err := ioutil.ReadAll(c.Request().Body)
	if err := json.Unmarshal(RawBody, &body); err != nil {
		return err
	}
	b, err := build.New(cfgFile)
	if err != nil {
		return err
	}

	if gst == "github" {
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

	} else if gst == "gitee" {
		if err := checkGiteeWebhook(c, opt); err != nil {
			return err
		}
		ref := strings.Split(body["ref"].(string), "/")
		branch = ref[len(ref)-1]
	} else {
		return errors.New("unsupported git service provider")
	}

	go b.Build(name, branch, vars.WebhookSender)
	return c.JSON(201, resb{res{Success: true}, branch})
}
