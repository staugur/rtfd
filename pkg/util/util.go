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
)

var (
	namePat = regexp.MustCompile(`^[a-zA-Z][0-9a-zA-Z\_\-]{1,100}$`)
	dnPat   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,62}(\.[a-zA-Z0-9][a-zA-Z0-9-]{0,62})*(\.[a-zA-Z][a-zA-Z0-9]{0,10}){1}$`)
	ipPat   = regexp.MustCompile(
		`^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$`,
	)
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
				if passwd, has := u.User.Password(); has == true {
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
	_, err = CheckGitURL(rawurl)
	if err != nil {
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
		return "GitHub", nil
	case strings.HasPrefix(git, "gitee.com"):
		return "Gitee", nil
	default:
		return "N/A", nil
	}
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
