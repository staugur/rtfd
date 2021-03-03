package lib

import (
	"strings"
	"testing"
)

func TestRenderNginx(t *testing.T) {
	opt := &nginxOptions{Name: "test", Lang: "zh-CN", Domain: "x.y.z"}
	_, err := opt.render()
	if err == nil {
		t.Fatal("should raise error")
	}

	opt.DocsDir = "/rtfd/docs"
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rst, "#:") {
		t.Fatal("render fail")
	}
	if strings.Count(rst, "location") != 1 {
		t.Fatal("render multi conf error")
	}

	opt.Single = true
	rst, _ = opt.render()
	if strings.Count(rst, "location") != 0 {
		t.Fatal("render single conf error")
	}

	opt.SSLCrt = "nginx.go"
	opt.SSLKey = "nginx.go"
	rst, _ = opt.render()
	if strings.Count(rst, "listen 443") != 1 {
		t.Fatal("render ssl conf error")
	}
}
