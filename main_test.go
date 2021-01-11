package main

import (
	"fmt"
	"testing"

	"rtfd/internal/db"

	"gopkg.in/ini.v1"
)

func TestDefaultConf(t *testing.T) {
	cfg, err := ini.Load("assets/rtfd.cfg")
	if err != nil {
		t.Fatalf("Fail to read file: %v", err)
	}
	cfg.BlockMode = false

	// 默认分区
	dftSec := cfg.Section(ini.DEFAULT_SECTION)
	if !dftSec.HasKey("base_dir") {
		t.Fatal("no base_dir")
	}

	ngxSec := cfg.Section("nginx")
	if !ngxSec.HasKey("dn") {
		t.Fatal("no nginx.dn")
	}
	if !ngxSec.HasKey("exec") {
		t.Fatal("no nginx.exec")
	}
	ssl := ngxSec.Key("ssl").MustBool()
	if ssl != false {
		t.Fatal("the nginx.ssl should be off")
	}

	pySec := cfg.Section("py")
	if !pySec.HasKey("py2") {
		t.Fatal("no py2")
	}
	if !pySec.HasKey("py3") {
		t.Fatal("no py3")
	}

	apiSec := cfg.Section("api")
	if !apiSec.HasKey("host") {
		t.Fatal("no api.host")
	}
	if !apiSec.HasKey("port") {
		t.Fatal("no api.port")
	}
	if !apiSec.HasKey("server_url") {
		t.Fatal("no api.server_url")
	}
	_, err = apiSec.Key("port").Int()
	if err != nil {
		t.Fatal("invalid api.port")
	}
}

func TestDB(t *testing.T) {
	db, err := db.New("~/.rtfd.cfg")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(db.DBDir)

	name := "test"
	k1 := []byte("hash")
	v1 := []byte("v1")
	err = db.Set(name, k1, v1)
	if err != nil {
		t.Fatal(err)
	}
	_v1, err := db.Get(name, k1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(_v1)
	if string(v1) != _v1 {
		t.Fatal("value not equal")
	}

	defer db.Close()

}
