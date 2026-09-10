package conf

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/ini.v1"
)

func TestConf(t *testing.T) {
	data := []byte(`
    gn = global
    [project]
    latest = master
    `)
	f := filepath.Join(os.TempDir(), "_rtfd_conf_test.ini")
	err := os.WriteFile(f, data, 0644)
	if err != nil {
		t.Fatal("write test file error")
	}

	cfg, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	if changeDefaultSection("DEFaULt") != ini.DefaultSection {
		t.Fatal("changeDefaultSection fail")
	}
	ds := make(map[string]string)
	ds["gn"] = "global"

	if reflect.DeepEqual(ds, cfg.SecHash("default")) != true {
		t.Fatal("SecHash default error")
	}

	hash := make(map[string]map[string]string)
	hash["DEFAULT"] = ds
	project := make(map[string]string)
	project["latest"] = "master"
	hash["project"] = project
	if reflect.DeepEqual(hash, cfg.AllHash()) != true {
		t.Fatal("all hash error")
	}

	if cfg.GetKey("project", "latest") != "master" {
		t.Fatal("get key error")
	}

	if cfg.MustKey("project", "non", "dft") != "dft" {
		t.Fatal("must get key error")
	}

}

func writeConf(t *testing.T, name, data string) (*Config, string) {
	t.Helper()
	f := filepath.Join(os.TempDir(), name)
	if err := os.WriteFile(f, []byte(data), 0644); err != nil {
		t.Fatal("write test file error")
	}
	cfg, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, f
}

func TestPy(t *testing.T) {
	cfg, f := writeConf(t, "_rtfd_py_test.ini", `
[py]
3 = python3
3.10 = /usr/bin/python3.10
default = 3.10
index = https://pypi.org/simple
`)
	defer os.Remove(f)

	versions := cfg.PyVersions()
	if strings.Join(versions, ",") != "3,3.10" {
		t.Fatalf("py versions error: %v", versions)
	}
	if cfg.HasPyVersion("default") || cfg.HasPyVersion("index") {
		t.Fatal("default or index should not be a python version")
	}
	if cfg.HasPyVersion("2") {
		t.Fatal("python2 should not be supported")
	}
	if cfg.PyCommand("3.10") != "/usr/bin/python3.10" {
		t.Fatal("py command error")
	}
	if cfg.PyCommand("3.12") != "" {
		t.Fatal("unconfigured version should be empty")
	}
	if cfg.DefaultPyVersion() != "3.10" {
		t.Fatalf("default py version error: %s", cfg.DefaultPyVersion())
	}
}

func TestPyCompat(t *testing.T) {
	// 旧配置：仅 py3，未配置 default 时取内置默认版本
	cfg, f := writeConf(t, "_rtfd_py_compat_test.ini", `
[py]
py3 = /usr/local/bin/python3
`)
	defer os.Remove(f)

	if len(cfg.PyVersions()) != 0 {
		t.Fatal("legacy py3 should not be a python version")
	}
	if cfg.PyCommand("3") != "/usr/local/bin/python3" {
		t.Fatal("legacy py3 command error")
	}
	if cfg.DefaultPyVersion() != "3" {
		t.Fatalf("default py version error: %s", cfg.DefaultPyVersion())
	}
}
