package lib

import (
	"strings"
	"testing"
)

func TestRenderSWS(t *testing.T) {
	opt := &swsOptions{
		Host: "0.0.0.0", Port: 80, Root: "/rtfd/docs",
		VHosts: []swsVHost{{Host: "a.example.com", Root: "/rtfd/docs/a"}},
		Redirects: []swsRedirect{
			{Host: "a.example.com", Source: "/", Destination: "/en/latest/", Kind: 302},
		},
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rst, "#:") {
		t.Fatal("render fail")
	}
	if !strings.Contains(rst, "[general]") {
		t.Fatal("no general section")
	}
	if !strings.Contains(rst, "[[advanced.virtual-hosts]]") {
		t.Fatal("no virtual-hosts table")
	}
	if !strings.Contains(rst, `host = "a.example.com"`) {
		t.Fatal("render virtual host error")
	}
	if !strings.Contains(rst, `destination = "/en/latest/"`) || !strings.Contains(rst, "kind = 302") {
		t.Fatal("render latest redirect error")
	}
	// 未配置 TLS 时不应出现 https 相关指令
	if strings.Contains(rst, "https-redirect") || strings.Contains(rst, "http2-tls-cert") {
		t.Fatal("unexpected tls directives")
	}
}

func TestRenderSWSTLS(t *testing.T) {
	opt := &swsOptions{
		Root:   "/rtfd/docs",
		VHosts: []swsVHost{{Host: "a.example.com", Root: "/rtfd/docs/a"}},
		TLSCert: "/rtfd/cert.pem", TLSKey: "/rtfd/key.pem",
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rst, `http2-tls-cert = "/rtfd/cert.pem"`) {
		t.Fatal("render tls cert error")
	}
	if !strings.Contains(rst, "https-redirect = true") {
		t.Fatal("render https redirect error")
	}
}

func TestRenderSWSCache(t *testing.T) {
	opt := &swsOptions{
		Root:   "/rtfd/docs",
		VHosts: []swsVHost{{Host: "a.example.com", Root: "/rtfd/docs/a"}},
		StaticExpires: "3600", HTMLNoCache: true,
	}
	rst, err := opt.render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rst, "max-age=3600") {
		t.Fatal("render static expires error")
	}
	if !strings.Contains(rst, `Cache-Control = "no-cache"`) {
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
