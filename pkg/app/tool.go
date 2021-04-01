package app

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func requestBase(method, url, auth string, body io.Reader) (text []byte, err error) {
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

func request(method, url, auth string) (text []byte, err error) {
	return requestBase(method, url, auth, nil)
}

func now() int64 {
	return time.Now().Unix()
}
