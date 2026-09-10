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
	// 多版本：静态资源location + 根location
	if strings.Count(rst, "location") != 2 {
		t.Fatal("render multi conf error")
	}

	opt.Single = true
	rst, _ = opt.render()
	// 单版本：仅静态资源location
	if strings.Count(rst, "location") != 1 {
		t.Fatal("render single conf error")
	}

	opt.SSLCrt = "nginx.go"
	opt.SSLKey = "nginx.go"
	rst, _ = opt.render()
	if strings.Count(rst, "listen 443") != 1 {
		t.Fatal("render ssl conf error")
	}
}

func TestRenderNginxCache(t *testing.T) {
	// 开启缓存：单版本与多版本模板均应渲染缓存指令
	for _, single := range []bool{false, true} {
		opt := &nginxOptions{
			Name: "test", Lang: "en", Domain: "x.y.z", DocsDir: "/rtfd/docs",
			Single: single, StaticExpires: "3600", HTMLNoCache: true, OpenFileCache: true,
		}
		rst, err := opt.render()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(rst, "expires 3600s") {
			t.Fatal("render static expires error")
		}
		if !strings.Contains(rst, `add_header Cache-Control "public"`) {
			t.Fatal("render static cache-control error")
		}
		if !strings.Contains(rst, "open_file_cache max=5000") {
			t.Fatal("render open_file_cache error")
		}
		if !strings.Contains(rst, `add_header Cache-Control "no-cache"`) {
			t.Fatal("render html cache-control error")
		}
	}

	// 关闭缓存：不应出现任何缓存指令
	opt := &nginxOptions{
		Name: "test", Lang: "en", Domain: "x.y.z", DocsDir: "/rtfd/docs",
		StaticExpires: "", HTMLNoCache: false, OpenFileCache: false,
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rst, "expires") ||
		strings.Contains(rst, "Cache-Control") ||
		strings.Contains(rst, "open_file_cache") {
		t.Fatal("should not render cache directives")
	}
}
