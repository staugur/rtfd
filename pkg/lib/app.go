// GitHub App 功能接口

package lib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"

	jwt "github.com/dgrijalva/jwt-go"
	"tcw.im/ufc"
)

// GitHub App Post Webhook data
type AppWebhook struct {
	// Install / Uninstall (nonexistent app)
	// Suspend / Unsuspend (existing app)
	Action       string       `json:"action"`
	Installation Installation `json:"installation"`
	Repositories []Repository `json:"repositories"`
	// Add / Remove repo in an existing app
	Repositories_removed []Repository `json:"repositories_removed"`
	Repositories_added   []Repository `json:"repositories_added"`
}

// Data structure after the user installs the github app
type Installation struct {
	ID             uint64 `json:"id"`
	AppID          uint64 `json:"app_id"`
	AppName        string `json:"app_slug"`
	AccessTokenURL string `json:"access_tokens_url"`
}

// Repo name & id
type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	ID       uint64 `json:"id"`
}

type AccessToken struct {
	Token string `json:"token"`
}

type UserWebhook struct {
	Type   string            `json:"type"`
	ID     uint64            `json:"id"`
	Name   string            `json:"name"`
	Active bool              `json:"active"`
	Events []string          `json:"events"`
	Config UserWebhookConfig `json:"config"`
}

type UserWebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret"`
}

type GHApp struct {
	AppId      uint64
	privateKey string

	baseURL string // api服务监听的地址（scheme://hostname:port）

	jwtoken   string
	jwtokenAt int64 // 生成 jwtoken 时间戳

	accessToken   string
	accessTokenAt int64 // 生成 accessToken 时间戳

	pm *ProjectManager
}

func request(method, url, auth string, body io.Reader) (text []byte, err error) {
	var client = &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(
		strings.ToUpper(method), url, body,
	)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")
	if auth != "" {
		// Bearer <jwt>, or token <Token>
		req.Header.Add("Authorization", auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func now() int64 {
	return time.Now().Unix()
}

func NewGHApp(pm *ProjectManager) (gh *GHApp, err error) {
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
	baseURL := cfg.GetKey("api", "server_url")
	gh = &GHApp{
		AppId: appId, privateKey: pkey, pm: pm, baseURL: baseURL,
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
		text, err := gh.requestWithJWT("POST", AccessTokenURL)
		if err != nil {
			log.Println("get accesstoken response is", string(text))
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

func (gh *GHApp) ghurl(uri string) string {
	if !strings.HasPrefix(uri, vars.GitHubApi) {
		if strings.HasPrefix(uri, "/") {
			uri = strings.TrimPrefix(uri, "/")
		}
		uri = fmt.Sprintf("%s/%s", vars.GitHubApi, uri)
	}
	return uri
}

func (gh *GHApp) requestWithJWT(method, uri string) (text []byte, err error) {
	if gh.jwtoken == "" {
		err = errors.New("invalid jwt")
		return
	}
	return request(method, gh.ghurl(uri), "Bearer "+gh.jwtoken, nil)
}

func (gh *GHApp) requestWithToken(method, uri string) (text []byte, err error) {
	if gh.accessToken == "" {
		err = errors.New("invalid access token")
		return
	}
	return request(method, gh.ghurl(uri), "token "+gh.accessToken, nil)
}

func (gh *GHApp) requestWithTokenBody(method, uri string, config UserWebhookConfig) (text []byte, err error) {
	if gh.accessToken == "" {
		err = errors.New("invalid access token")
		return
	}
	body := make(map[string]interface{})
	body["config"] = config
	body["events"] = []string{"push", "release"}
	bodyByte, err := json.Marshal(body)
	if err != nil {
		return
	}
	bodyReader := bytes.NewReader(bodyByte)
	return request(method, gh.ghurl(uri), "token "+gh.accessToken, bodyReader)
}

func (gh *GHApp) genRoute(name string) []string {
	u := gh.baseURL
	if strings.HasSuffix(u, "/") {
		u = strings.TrimSuffix(u, "/")
	}
	r1 := fmt.Sprintf("%s/rtfd/%s/webhook", u, name)
	r2 := fmt.Sprintf("%s/rtfd/webhook/%s", u, name)
	return []string{r1, r2}
}

func (gh *GHApp) setUserWebhook(opt *Options, fullname string) (err error) {
	uri := fmt.Sprintf("/repos/%s/hooks", fullname)
	text, err := gh.requestWithToken("GET", uri)
	if err != nil {
		return
	}
	var uws []UserWebhook
	err = json.Unmarshal(text, &uws)
	if err != nil {
		log.Println("query user webhook response is", string(text))
		return
	}
	isAdd := true
	routes := gh.genRoute(opt.Name)
	for _, uw := range uws {
		uwu := uw.Config.URL
		if ufc.StrInSlice(uwu, routes) {
			isAdd = false
			opt.UpdateMeta(vars.WebhookID, fmt.Sprint(uw.ID))
			err = opt.Writeback(gh.pm)
			if err != nil {
				log.Printf("found webhook, but record id failed: %s\n", err)
			}
		}
	}
	if isAdd {
		// add repo webhook
		config := UserWebhookConfig{routes[1], "json", opt.Secret}
		text, err = gh.requestWithTokenBody("POST", uri, config)
		if err != nil {
			log.Println("the create web hook response is", string(text))
			return
		}
		var resp UserWebhook
		err = json.Unmarshal(text, &resp)
		if err != nil {
			return
		}
		opt.UpdateMeta(vars.WebhookID, fmt.Sprint(resp.ID))
		err = opt.Writeback(gh.pm)
		if err != nil {
			return err
		}
	}
	return err
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
	for _, mo := range mems {
		opt := mo
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
			(&opt).UpdateMeta(vars.InstallID, vars.ResetEmpty)
			(&opt).UpdateMeta(vars.WebhookID, vars.ResetEmpty)
		} else {
			(&opt).UpdateMeta(vars.InstallID, fmt.Sprint(i.ID))
			go gh.setUserWebhook(&opt, fullname)
		}
		err = (&opt).Writeback(gh.pm)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func (gh *GHApp) Dispatch(w AppWebhook) (err error) {
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

func (gh *GHApp) cliSetWebhook(opt *Options) error {
	if gh.baseURL == "" {
		return errors.New("invalid base_url")
	}
	fullname, err := util.GitUserRepo(opt.URL)
	if err != nil {
		return err
	}
	err = gh.generateJWT()
	if err != nil {
		return err
	}
	uri := fmt.Sprintf("/repos/%s/installation", fullname)
	text, err := gh.requestWithJWT("GET", uri)
	if err != nil {
		return err
	}
	var i Installation
	err = json.Unmarshal(text, &i)
	if err != nil {
		return err
	}
	if i.ID == 0 {
		return errors.New("not installed or authorized")
	}
	err = gh.setAllToken(i.AccessTokenURL)
	if err != nil {
		return err
	}
	err = gh.setUserWebhook(opt, fullname)
	if err != nil {
		return err
	}
	opt.UpdateMeta(vars.InstallID, fmt.Sprint(i.ID))
	return opt.Writeback(gh.pm)
}

func (gh *GHApp) cliRemoveWebhook(opt Options) error {
	wid := opt.GetMeta(vars.WebhookID)
	iid := opt.GetMeta(vars.InstallID)
	if wid == "" || iid == "" {
		// not install, not error
		return nil
	}

	fullname, err := util.GitUserRepo(opt.URL)
	if err != nil {
		return err
	}
	err = gh.generateJWT()
	if err != nil {
		return err
	}
	err = gh.setAllToken(
		gh.ghurl(fmt.Sprintf("/app/installations/%s/access_tokens", iid)),
	)
	if err != nil {
		return err
	}
	_, err = gh.requestWithToken(
		"DELETE", fmt.Sprintf("/repos/%s/hooks/%s", fullname, wid),
	)
	return err
}
