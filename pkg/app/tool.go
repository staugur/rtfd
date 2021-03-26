package app

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func request(method string, uri string, auth string) (text []byte, err error) {
	var client = &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(
		strings.ToUpper(method), "https://api.github.com"+uri, nil,
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
