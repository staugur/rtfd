package main

import (
	"testing"

	"pkg.tcw.im/rtfd/v2/pkg/conf"

	"gopkg.in/ini.v1"
)

func TestDefaultConf(t *testing.T) {
	cfg, err := ini.Load("assets/rtfd.cfg")
	if err != nil {
		t.Fatalf("Fail to read file: %v", err)
	}
	cfg.BlockMode = false

	// 默认分区
	dftSec := cfg.Section(ini.DefaultSection)
	if !dftSec.HasKey("base_dir") {
		t.Fatal("no base_dir")
	}
	if dftSec.HasKey("redis") {
		t.Fatal("redis is not supported")
	}

	dbSec := cfg.Section("database")
	if !dbSec.HasKey("type") {
		t.Fatal("no database.type")
	}
	if !dbSec.HasKey("dsn") {
		t.Fatal("no database.dsn")
	}

	caddySec := cfg.Section("caddy")
	if !caddySec.HasKey("dn") {
		t.Fatal("no caddy.dn")
	}
	if !caddySec.HasKey("exec") {
		t.Fatal("no caddy.exec")
	}

	pySec := cfg.Section("py")
	if pySec.HasKey("py2") {
		t.Fatal("python2 is not supported")
	}
	if !pySec.HasKey("3") {
		t.Fatal("no python3")
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
	if !apiSec.HasKey("secret") {
		t.Fatal("no api.secret")
	}
	_, err = apiSec.Key("port").Int()
	if err != nil {
		t.Fatal("invalid api.port")
	}

	pcfg, err := conf.New("assets/rtfd.cfg")
	if err != nil {
		t.Fatalf("Fail to read file: %v", err)
	}
	if len(pcfg.PyVersions()) == 0 {
		t.Fatal("no available python version")
	}
	if !pcfg.HasPyVersion(pcfg.DefaultPyVersion()) {
		t.Fatal("invalid default python version")
	}
	if pcfg.DatabaseType() == "" || pcfg.DatabaseDSN() == "" {
		t.Fatal("invalid database config")
	}
}
