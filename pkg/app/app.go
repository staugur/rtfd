// GitHub App 功能接口

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"tcw.im/rtfd/pkg/conf"

	jwt "github.com/dgrijalva/jwt-go"
	"tcw.im/ufc"
)

type GHApp struct {
	AppId      uint64
	privateKey string

	installationId uint64

	jwtoken     string
	accessToken string
}

func New(cfgpath string, installationId uint64) (gh *GHApp, err error) {
	cfg, err := conf.New(cfgpath)
	if err != nil {
		return
	}
	sec := "ghapp"
	if ufc.IsFalse(cfg.GetKey(sec, "enable")) {
		err = errors.New("service is not enabled")
		return
	}
	name := cfg.GetKey(sec, "app_name")
	id := cfg.GetKey(sec, "app_id")
	pkey := cfg.GetKey(sec, "private_key")
	if name == "" || id == "" || pkey == "" {
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
	gh = &GHApp{AppId: appId, privateKey: pkey, installationId: installationId}
	return
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
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 10).Unix(),
		"iss": gh.AppId,
	})
	jwtoken, err := token.SignedString(rsakey)
	if err != nil {
		return err
	}
	gh.jwtoken = jwtoken
	return nil
}

func (gh *GHApp) setAccessToken() error {
	uri := fmt.Sprintf("/app/installations/%d/access_tokens", gh.installationId)
	if gh.jwtoken == "" {
		err := gh.generateJWT()
		if err != nil {
			return err
		}
	}
	text, err := request("POST", uri, "Bearer "+gh.jwtoken)
	if err != nil {
		return err
	}
	log.Println(text)
	data := new(AccessToken)
	json.Unmarshal(text, data)
	log.Println(data)
	return nil
}

func (gh *GHApp) Dispatch(action string) error {
	return nil
}
