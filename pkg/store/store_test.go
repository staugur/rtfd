/*
   Copyright 2021 Hiroshi.tao

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package store

import (
	"path/filepath"
	"testing"
)

func TestParseDBType(t *testing.T) {
	cases := map[string]DBType{
		"sqlite":     SQLite,
		"SQLite3":    SQLite,
		"mysql":      MySQL,
		"mariadb":    MySQL,
		"pgsql":      PostgreSQL,
		"postgres":   PostgreSQL,
		"POSTGRESQL": PostgreSQL,
	}
	for in, want := range cases {
		got, err := ParseDBType(in)
		if err != nil {
			t.Fatalf("ParseDBType(%q) error: %v", in, err)
		}
		if got != want {
			t.Fatalf("ParseDBType(%q) = %s, want %s", in, got, want)
		}
	}
	if _, err := ParseDBType("oracle"); err == nil {
		t.Fatal("unsupported type should raise error")
	}
}

func TestOpenSQLite(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "sub", "rtfd.db")
	db, err := Open("sqlite", dsn, false)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(db)

	if !db.Migrator().HasTable(&Project{}) {
		t.Fatal("projects table not created")
	}
	if !db.Migrator().HasTable(&BuildResult{}) {
		t.Fatal("build_results table not created")
	}

	p := &Project{
		Name: "demo", URL: "https://github.com/staugur/rtfd", Latest: "master",
		Version: "3.10", SourceDir: "docs", Lang: "en", Builder: "html",
		DefaultDomain: "demo.example.com", Meta: map[string]string{"a": "1"},
	}
	if err = db.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	var got Project
	if err = db.Where("name = ?", "demo").First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Version != "3.10" || got.Lang != "en" {
		t.Fatalf("project data error: %+v", got)
	}
	if got.Meta["a"] != "1" {
		t.Fatalf("project meta error: %+v", got.Meta)
	}

	// 同名项目应唯一
	if err = db.Create(&Project{Name: "demo", URL: "x"}).Error; err == nil {
		t.Fatal("duplicated project name should raise error")
	}

	// 构建结果
	if err = db.Create(&BuildResult{
		Project: "demo", Branch: "master", Status: true, Sender: "cli",
		Btime: "2026-09-10 00:00:00", Usedtime: 10,
	}).Error; err != nil {
		t.Fatal(err)
	}
	var rst BuildResult
	if err = db.Where("project = ? AND branch = ?", "demo", "master").First(&rst).Error; err != nil {
		t.Fatal(err)
	}
	if !rst.Status || rst.Usedtime != 10 {
		t.Fatalf("build result error: %+v", rst)
	}
}

func TestOpenInvalid(t *testing.T) {
	if _, err := Open("oracle", "x", false); err == nil {
		t.Fatal("unsupported type should raise error")
	}
	if _, err := Open("sqlite", "", false); err == nil {
		t.Fatal("empty dsn should raise error")
	}
}
