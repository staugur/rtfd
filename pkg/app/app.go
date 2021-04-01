// GitHub App 功能接口

package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"tcw.im/rtfd/pkg/lib"
	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"

	jwt "github.com/dgrijalva/jwt-go"
	"tcw.im/ufc"
)

var (
	WebhookID = "_WebhookID"
	InstallID = "_InstallationID"
)

type GHApp struct {
	AppId      uint64
	privateKey string

	baseURL        string // api服务监听的地址（scheme://hostname:port）
	installationId uint64

	jwtoken   string
	jwtokenAt int64 // 生成 jwtoken 时间戳

	accessToken   string
	accessTokenAt int64 // 生成 accessToken 时间戳

	pm *lib.ProjectManager
}

func New(cfgpath string, installationId uint64) (gh *GHApp, err error) {
	pm, err := lib.New(cfgpath)
	if err != nil {
		return
	}
	cfg := pm.CFG()
	sec := "ghapp"
	if ufc.IsFalse(cfg.GetKey(sec, "enable")) {
		err = errors.New("service is not enabled")
		return
	}
	id := cfg.GetKey(sec, "app_id")
	pkey, err := cfg.GetPath(sec, "private_key")
	if err != nil {
		return
	}
	if id == "" || pkey == "" {
		err = errors.New("invalid param")
		return
	}
	if !ufc.IsFile(pkey) {
		err = errors.New("not found private key file")
		return
	}
	appId, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return
	}
	baseURL := cfg.GetKey(sec, "base_url")
	gh = &GHApp{
		AppId: appId, privateKey: pkey, installationId: installationId, pm: pm,
		baseURL: baseURL,
	}
	return
}

func (gh *GHApp) BaseURL(url string) {
	gh.baseURL = url
}

func (gh *GHApp) generateJWT() error {
	content, err := ioutil.ReadFile(gh.privateKey)
	if err != nil {
		return err
	}
	rsakey, err := jwt.ParseRSAPrivateKeyFromPEM(content)
	if err != nil {
		return err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now(),
		"exp": now() + 600,
		"iss": gh.AppId,
	})
	jwtoken, err := token.SignedString(rsakey)
	if err != nil {
		return err
	}
	gh.jwtoken = jwtoken
	gh.jwtokenAt = now()
	return nil
}

func (gh *GHApp) setAllToken(AccessTokenURL string) error {
	if gh.jwtoken == "" || (gh.jwtokenAt+600 < now()) {
		err := gh.generateJWT()
		if err != nil {
			return err
		}
	}
	if gh.accessTokenAt == 0 || gh.accessTokenAt+3600 < now() {
		text, err := request("POST", AccessTokenURL, "Bearer "+gh.jwtoken)
		if err != nil {
			log.Println("get accesstoken response is ", string(text))
			return err
		}
		data := new(AccessToken)
		err = json.Unmarshal(text, data)
		if err != nil {
			return err
		}
		gh.accessToken = data.Token
		gh.accessTokenAt = now()
	}
	return nil
}

func (gh *GHApp) requestWithToken(method, uri string) (text []byte, err error) {
	if gh.accessToken == "" {
		err = errors.New("invalid access token")
		return
	}
	return request(method, "https://api.github.com"+uri, "token "+gh.accessToken)
}

func (gh *GHApp) requestWithTokenBody(method, uri string, config UserWebhookConfig) (text []byte, err error) {
	body := make(map[string]interface{})
	body["config"] = config
	body["events"] = []string{"push", "release"}
	bodyByte, err := json.Marshal(body)
	if err != nil {
		return
	}
	bodyReader := bytes.NewReader(bodyByte)
	return requestBase(
		method, "https://api.github.com"+uri, "token "+gh.accessToken, bodyReader,
	)
}

func (gh *GHApp) apiGenRoute(name string) []string {
	r1 := fmt.Sprintf("%s/rtfd/%s/webhook", gh.baseURL, name)
	r2 := fmt.Sprintf("%s/rtfd/webhook/%s", gh.baseURL, name)
	return []string{r1, r2}
}

func (gh *GHApp) apiSetUserWebhook(opt lib.Options, fullname string) {
	uri := fmt.Sprintf("/repos/%s/hooks", fullname)
	text, err := gh.requestWithToken("GET", uri)
	if err != nil {
		log.Println(err)
		return
	}
	var uws []UserWebhook
	err = json.Unmarshal(text, &uws)
	if err != nil {
		log.Println(err)
		log.Println("query user webhook response is", string(text))
		return
	}
	isAdd := true
	routes := gh.apiGenRoute(opt.Name)
	for _, uw := range uws {
		uwu := uw.Config.URL
		if ufc.StrInSlice(uwu, routes) {
			isAdd = false
			opt.UpdateMeta(WebhookID, fmt.Sprint(uw.ID))
			opt.Writeback(gh.pm)
		}
	}
	if isAdd {
		// add repo webhook
		config := UserWebhookConfig{routes[1], "json", opt.Secret}
		text, err = gh.requestWithTokenBody("POST", uri, config)
		if err != nil {
			log.Println(err)
			log.Println("the create web hook response is", string(text))
			return
		}
		var resp UserWebhook
		err = json.Unmarshal(text, &resp)
		if err != nil {
			log.Println(err)
			return
		}
		opt.UpdateMeta(WebhookID, fmt.Sprint(resp.ID))
		err = opt.Writeback(gh.pm)
		if err != nil {
			log.Println(err)
		}
	}
}

func (gh *GHApp) apiUpdateOption(i Installation, repos []Repository, isRemove bool) error {
	mems, err := gh.pm.ListFullProject()
	if err != nil {
		return err
	}

	err = gh.setAllToken(i.AccessTokenURL)
	if err != nil {
		return err
	}

	willSetup := make([]string, 0, len(repos))
	for _, r := range repos {
		willSetup = append(willSetup, strings.ToLower(r.FullName))
	}
	for _, opt := range mems {
		if opt.GSP != vars.GSPGitHub {
			continue
		}
		fullname, err := util.GitUserRepo(opt.URL)
		if err != nil {
			continue
		}
		if !ufc.StrInSlice(fullname, willSetup) {
			continue
		}
		if isRemove {
			(&opt).UpdateMeta(InstallID, vars.ResetEmpty)
			(&opt).UpdateMeta(WebhookID, vars.ResetEmpty)
		} else {
			(&opt).UpdateMeta(InstallID, fmt.Sprint(i.ID))
			// cannot pass pointer
			go gh.apiSetUserWebhook(opt, fullname)
		}
		err = (&opt).Writeback(gh.pm)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func (gh *GHApp) Dispatch(w Webhook) (err error) {
	switch w.Action {
	case "created":
		err = gh.apiUpdateOption(w.Installation, w.Repositories, false)
	case "deleted":
		err = gh.apiUpdateOption(w.Installation, w.Repositories, true)
	case "added":
		err = gh.apiUpdateOption(w.Installation, w.Repositories_added, false)
	case "removed":
		err = gh.apiUpdateOption(w.Installation, w.Repositories_removed, true)
	}
	if err != nil {
		log.Println(err)
	}
	return
}
