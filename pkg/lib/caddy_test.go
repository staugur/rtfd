package lib

import (
	"strings"
	"testing"
)

func TestRenderCaddy(t *testing.T) {
	opt := &caddyOptions{
		Sites: []caddySite{
			{Address: "a.example.com, docs.custom.com", Root: "/rtfd/docs/a", Redirect: "/en/latest/"},
		},
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rst, "#:") {
		t.Fatal("render fail")
	}
	if !strings.Contains(rst, "a.example.com, docs.custom.com {") {
		t.Fatal("render site block error")
	}
	if !strings.Contains(rst, "root * /rtfd/docs/a") {
		t.Fatal("render root error")
	}
	if !strings.Contains(rst, "redir / /en/latest/ 302") {
		t.Fatal("render latest redirect error")
	}
	if !strings.Contains(rst, "file_server") {
		t.Fatal("render file_server error")
	}
	// 未配置邮箱时不应出现全局选项
	if strings.Contains(rst, "email") {
		t.Fatal("unexpected global options")
	}
}

func TestRenderCaddyEmail(t *testing.T) {
	opt := &caddyOptions{
		Email: "admin@example.com",
		Sites: []caddySite{{Address: "a.example.com", Root: "/rtfd/docs/a"}},
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rst, "email admin@example.com") {
		t.Fatal("render email error")
	}
}

func TestRenderCaddyCache(t *testing.T) {
	opt := &caddyOptions{
		Sites:         []caddySite{{Address: "a.example.com", Root: "/rtfd/docs/a"}},
		StaticExpires: "3600", HTMLNoCache: true,
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rst, "max-age=3600") {
		t.Fatal("render static expires error")
	}
	if !strings.Contains(rst, `Cache-Control "no-cache"`) {
		t.Fatal("render html cache-control error")
	}

	// 关闭缓存：不应出现任何缓存指令
	opt.StaticExpires = ""
	opt.HTMLNoCache = false
	rst, err = opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rst, "max-age") || strings.Contains(rst, "no-cache") {
		t.Fatal("should not render cache directives")
	}
}

func TestCaddyAddress(t *testing.T) {
	if got := caddyAddress("a.example.com", true); got != "a.example.com" {
		t.Fatalf("https address error: %s", got)
	}
	if got := caddyAddress("a.example.com", false); got != "http://a.example.com" {
		t.Fatalf("http address error: %s", got)
	}
}

func TestCaddySiteAddrs(t *testing.T) {
	// 模式一致：合并为一个站点块（逗号分隔地址）
	if got := caddySiteAddrs("a.rtfd.vip", "docs.example.com", true, true); len(got) != 1 ||
		got[0] != "a.rtfd.vip, docs.example.com" {
		t.Fatalf("merge error: %v", got)
	}
	// 模式一致（均关闭）：都加 http:// 前缀，仍合并
	if got := caddySiteAddrs("a.rtfd.vip", "docs.example.com", false, false); len(got) != 1 ||
		got[0] != "http://a.rtfd.vip, http://docs.example.com" {
		t.Fatalf("merge http error: %v", got)
	}
	// 模式不同：拆成两个站点块
	if got := caddySiteAddrs("a.rtfd.vip", "docs.example.com", false, true); len(got) != 2 ||
		got[0] != "http://a.rtfd.vip" || got[1] != "docs.example.com" {
		t.Fatalf("split error: %v", got)
	}
	// 无自定义域名：仅默认域名
	if got := caddySiteAddrs("a.rtfd.vip", "", false, true); len(got) != 1 ||
		got[0] != "http://a.rtfd.vip" {
		t.Fatalf("no custom domain error: %v", got)
	}
}
