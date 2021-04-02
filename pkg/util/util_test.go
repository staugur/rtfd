package util

import (
	"strings"
	"testing"
)

func TestUtil(t *testing.T) {
	okNames := []string{"abc", "a1", "V-8", "is_true", "Flask-PluginKit",
		"Bxxa-", "c_no"}
	errNames := []string{"a", "1b", "-abc", "_d0", "/abc", "v-8@"}
	for _, ok := range okNames {
		if IsProjectName(ok) != true {
			t.Fatalf("%s should be ok\n", ok)
		}
	}
	for _, fail := range errNames {
		if IsProjectName(fail) != false {
			t.Fatalf("%s should not be ok\n", fail)
		}
	}

	falseNames := []string{"a", "_", "0a", "Ab", "@_a", "a-bc", "a#", "123"}
	trueNames := []string{"a0", "_b", "a_c", "hello"}
	for _, f := range falseNames {
		if LLPat.MatchString(f) != false {
			t.Fatalf("LLPat: %s should be false\n", f)
		}
	}
	for _, v := range trueNames {
		if LLPat.MatchString(v) != true {
			t.Fatalf("LLPat: %s should be true\n", v)
		}
	}

	_, _, err := RunCmd("ls")
	if err != nil {
		t.Fatal("run cmd error")
	}

	hs1k := "hello world!"
	hs1v := "9bf0f4bf184c31eea044ee583ef35aa9532337e6"
	if HMACSha1("abc", hs1k) != hs1v {
		t.Fatal("hmac sha1 fail")
	}
	if HMACSha1Byte([]byte("abc"), []byte(hs1k)) != hs1v {
		t.Fatal("hmac-byte sha1 fail")
	}
}

func TestGitURL(t *testing.T) {
	giturls := []struct {
		url    string
		status string
		hasErr bool
	}{
		{"http://github.com/staugur/rtfd", "public", false},
		{"http://gitee.com/staugur/rtfd", "public", false},
		{"https://github.com/staugur/rtfd", "public", false},
		{"https://gitee.com/staugur/rtfd", "public", false},
		{"https://user:passwd@github.com/staugur/rtfd", "private", false},
		{"https://user:passwd@gitee.com/staugur/rtfd", "private", false},
		{"https://:passwd@github.com/staugur/rtfd", "public", false},
		{"https://user:@github.com/staugur/rtfd", "", true},
		{"https://gitlab.com/staugur/rtfd", "", true},
		{"https://user:passwd@gitlab.com/staugur/rtfd", "", true},
		{"", "", true},
		{"xyz", "", true},
	}

	for _, g := range giturls {
		status, err := CheckGitURL(g.url)
		if err != nil {
			// 发生错误，表明url不合法或不支持
			if g.hasErr != true {
				t.Fatalf("not support or invalid: %s\n", g.url)
			}
		}
		// 无错误
		if status != g.status {
			t.Fatalf("no error, but status fail, %s\n", g.url)
		}
	}

	for _, g := range giturls {
		if g.hasErr == false {
			pub, err := PublicGitURL(g.url)
			if err != nil {
				t.Fatal("public git url should no error")
			}
			u1 := strings.Replace(g.url, "user:passwd@", "", 1)
			u2 := strings.Replace(u1, "user:@", "", 1)
			u3 := strings.Replace(u2, ":passwd@", "", 1)
			if pub != u3 {
				t.Fatalf("the last u3 should be equal to url: %s %s\n", pub, g.url)
			}

			gsp, _ := GitServiceProvider(g.url)
			if strings.HasPrefix(pub, "https://github.com") || strings.HasPrefix(pub, "http://github.com") {
				if gsp != "GitHub" {
					t.Fatalf("%s should be github\n", g.url)
				}
			} else if strings.HasPrefix(pub, "https://gitee.com") || strings.HasPrefix(pub, "http://gitee.com") {
				if gsp != "Gitee" {
					t.Fatalf("%s should be gitee\n", g.url)
				}
			}
		}
	}
}

func TestDNS(t *testing.T) {
	var dns = []struct {
		param    string
		expected bool
	}{
		{"localhost", false},
		{"http://localhost:5000", false},
		{"https://abc.com", false},
		{"https://abc.com:8443", false},
		{"ftp://192.168.1.2", false},
		{"rsync://192.168.1.2", false},
		{"192.168.1.2", false},
		{"://127.0.0.1/hello-world", false},
		{"x_y_z.com", false},
		{"_x-y-z.com", false},
		{"false", false},
		{"test.test.example.com", true},
		{"x-y-z.com", true},
		{"a.bc", true},
		{"a.b.", false},
		{"a.b..", false},
		{"localhost.local", true},
		{"localhost.localdomain.intern", true},
		{"l.local.intern", true},
		{"ru.link.n.svpncloud.com", true},
		{"-localhost", false},
		{"localhost.-localdomain", false},
		{"localhost.localdomain.-int", false},
		{"_localhost", false},
		{"localhost._localdomain", false},
		{"localhost.localdomain._int", false},
		{"lÖcalhost", false},
		{"localhost.lÖcaldomain", false},
		{"localhost.localdomain.üntern", false},
		{"__", false},
		{"localhost/", false},
		{"127.0.0.1", false},
		{"http://127.0.0.1", false},
		{"[::1]", false},
		{"50.50.50.50", false},
		{"localhost.localdomain.intern:65535", false},
		{"漢字汉字", false},
		{"www.jubfvq1v3p38i51622y0dvmdk1mymowjyeu26gbtw9andgynj1gg8z3msb1kl5z6906k846pj3sulm4kiyk82ln5teqj9nsht59opr0cs5ssltx78lfyvml19lfq1wp4usbl0o36cmiykch1vywbttcus1p9yu0669h8fj4ll7a6bmop505908s1m83q2ec2qr9nbvql2589adma3xsq2o38os2z3dmfh2tth4is4ixyfasasasefqwe4t2ub2fz1rme.de", false},
		{"http://127.0.0.1", false},
		{"http://localhost:5000", false},
		{"https://abc.com", false},
		{"https://abc.com:8443", false},
		{"ftp://192.168.1.2", false},
		{"rsync://192.168.1.2", false},
		{"192.168.1.2", false},
		{"1.1.1.1", false},
		{"localhost", false},
		{"127.0.0.1:8000", false},
		{"://127.0.0.1/hello-world", false},
		{"abc.com", true},
		{"localhost.localdomain", true},
	}

	for _, test := range dns {
		actual := IsDomain(test.param)
		if actual != test.expected {
			t.Errorf("Expected IsDomain(%q) to be %v, got %v", test.param, test.expected, actual)
		}
	}
}
