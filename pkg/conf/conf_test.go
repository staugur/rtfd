package conf

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
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
	err := ioutil.WriteFile(f, data, 0644)
	if err != nil {
		t.Fatal("write test file error")
	}

	cfg, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	if changeDefaultSection("DEFaULt") != ini.DEFAULT_SECTION {
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
