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

// 内部公用工具

package util

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"tcw.im/rtfd/vars"
)

var (
	namePat = regexp.MustCompile(`^[a-zA-Z][0-9a-zA-Z\_\-]{1,100}$`)
	dnPat   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,62}(\.[a-zA-Z0-9][a-zA-Z0-9-]{0,62})*(\.[a-zA-Z][a-zA-Z0-9]{0,10}){1}$`)
	LLPat   = regexp.MustCompile(`^[a-z\_][0-9a-z\_]{1,63}$`)
)

// IsProjectName 判断name是否为合法名称
func IsProjectName(name string) bool {
	if name != "" && namePat.MatchString(name) {
		return true
	}
	return false
}

// RunCmd 封装命令执行方法
func RunCmd(name string, args ...string) (exitCode int, out string, err error) {
	cmd := exec.Command(name, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	return cmd.ProcessState.ExitCode(), string(data), nil
}

// RunCmdStream 在控制台实时输出命令返回信息
func RunCmdStream(name string, args []string, f func(line string)) error {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	//从管道中实时循环读取输出流中的一行内容
	reader := bufio.NewReader(stdout)
	for {
		line, e := reader.ReadString('\n')
		if e != nil || io.EOF == e {
			break
		}
		if f != nil {
			f(line)
		}
	}

	return cmd.Wait()
}

// IsIP 检测IPv4、IPv6
func IsIP(str string) bool {
	return net.ParseIP(str) != nil
}

// IsDomain 判断是否为合法DNS域名
func IsDomain(v string) bool {
	if v == "" || len(strings.Replace(v, ".", "", -1)) > 255 {
		return false
	}
	dots := strings.Count(v, ".")
	if dots < 1 {
		return false
	}
	if !IsIP(v) && dnPat.MatchString(v) {
		return true
	}
	return false
}

// GetNow 获取当前年月日时分秒
func GetNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// CheckGitURL 检查url是否为支持的git地址。
// 当无error时，返回public或private表示公共、私有仓库；
// 当有error时，返回错误提示。
func CheckGitURL(rawurl string) (string, error) {
	if rawurl != "" && (strings.HasPrefix(rawurl, "http://") || strings.HasPrefix(rawurl, "https://")) {
		u, err := url.Parse(rawurl)
		if err != nil {
			return "", err
		}
		if u.Host == "github.com" || u.Host == "gitee.com" {
			if u.User.Username() != "" {
				if passwd, has := u.User.Password(); has {
					if passwd == "" {
						return "", errors.New("empty password")
					}
					return "private", nil
				}
				return "", errors.New("the warehouse has set up users but no password")
			}
			return "public", nil
		}
		return "", errors.New("unsupported git service provider")
	}
	return "", errors.New("invalid url")
}

// PublicGitURL 获取可公开的git地址（如果是私有仓库则会去掉用户名密码）
func PublicGitURL(rawurl string) (puburl string, err error) {
	if _, err = CheckGitURL(rawurl); err != nil {
		return "", err
	}
	u, _ := url.Parse(rawurl)
	return strings.Replace(u.String(), u.User.String()+"@", "", 1), nil
}

// GitServiceProvider 获取git服务商
func GitServiceProvider(rawurl string) (gsp string, err error) {
	puburl, err := PublicGitURL(rawurl)
	if err != nil {
		return "", err
	}
	git := strings.ToLower(strings.Split(puburl, "//")[1])
	switch {
	case strings.HasPrefix(git, "github.com"):
		return vars.GSPGitHub, nil
	case strings.HasPrefix(git, "gitee.com"):
		return vars.GSPGitee, nil
	default:
		return vars.GSPNA, nil
	}
}

// GitUserRepo 提取git地址中username、repo
func GitUserRepo(rawurl string) (fullname string, err error) {
	puburl, err := PublicGitURL(rawurl)
	if err != nil {
		return
	}
	u, err := url.Parse(puburl)
	if err != nil {
		return
	}
	fullname = strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), "/")
	return strings.ToLower(fullname), nil
}

// HMACSha1 以hmac加盐方式检测字符串sha1值
func HMACSha1(key, text string) string {
	return HMACSha1Byte([]byte(key), []byte(text))
}

// HMACSha1Byte 同 HMACSha1
func HMACSha1Byte(key, text []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(text)
	return hex.EncodeToString(mac.Sum(nil))
}
