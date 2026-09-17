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

package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateProject(t *testing.T) {
	pm, _ := newTestPM(t)

	rule := map[string]any{
		"url":       "https://github.com/staugur/rtfd",
		"latest":    "main",
		"lang":      "zh_CN,en",
		"single":    "on",
		"sourcedir": "doc",
		"install":   true,
		"domain":    "docs.example.com",
	}
	opt, err := pm.CreateProject("Demo", rule)
	if err != nil {
		t.Fatal(err)
	}
	if opt.Name != "demo" || opt.Latest != "main" || opt.Lang != "zh_CN,en" {
		t.Fatalf("create option error: %+v", opt)
	}
	if !opt.Single || !opt.Install || opt.SourceDir != "doc" {
		t.Fatalf("create rule not applied: %+v", opt)
	}
	if opt.CustomDomain != "docs.example.com" || !pm.HasCustomDomain("docs.example.com") {
		t.Fatalf("custom domain not applied: %+v", opt)
	}

	// 重复创建与缺少url应失败
	if _, err = pm.CreateProject("demo", rule); err == nil {
		t.Fatal("duplicated project should raise error")
	}
	if _, err = pm.CreateProject("nourl", map[string]any{"lang": "en"}); err == nil {
		t.Fatal("empty url should raise error")
	}
	// 非法字段应失败
	if _, err = pm.CreateProject("badfield", map[string]any{
		"url": "https://github.com/staugur/rtfd", "notexist": "1",
	}); err == nil {
		t.Fatal("invalid field should raise error")
	}
	// 空值不覆盖默认配置
	opt2, err := pm.CreateProject("demo2", map[string]any{
		"url": "https://github.com/staugur/rtfd", "lang": "", "latest": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opt2.Lang != "en" || opt2.Latest != "master" {
		t.Fatalf("empty value should keep default: %+v", opt2)
	}
}

func TestParseUpdateRule(t *testing.T) {
	rule, err := ParseUpdateRule("lang:zh_CN,single:on,requirement:-", ":")
	if err != nil {
		t.Fatal(err)
	}
	if rule["lang"] != "zh_CN" || rule["single"] != "on" {
		t.Fatalf("parse rule error: %v", rule)
	}
	if rule["requirement"] != "" {
		t.Fatalf("reset empty field error: %v", rule)
	}

	// sslcrt与sslpri合并为ssl
	rule, err = ParseUpdateRule("sslcrt:/tmp/a.pem,sslpri:/tmp/b.pem", ":")
	if err != nil {
		t.Fatal(err)
	}
	if rule["ssl"] != "/tmp/a.pem,/tmp/b.pem" {
		t.Fatalf("merge ssl error: %v", rule)
	}
	// ssl仅支持取消
	if _, err = ParseUpdateRule("ssl:on", ":"); err == nil {
		t.Fatal("invalid ssl should raise error")
	}
	rule, err = ParseUpdateRule("ssl:off", ":")
	if err != nil {
		t.Fatal(err)
	}
	if rule["ssl"] != "off" {
		t.Fatalf("clear ssl error: %v", rule)
	}
	// 自定义分隔符与非法格式
	rule, err = ParseUpdateRule("lang=zh_CN", "=")
	if err != nil {
		t.Fatal(err)
	}
	if rule["lang"] != "zh_CN" {
		t.Fatalf("custom sep error: %v", rule)
	}
	if _, err = ParseUpdateRule("lang", ":"); err == nil {
		t.Fatal("invalid format should raise error")
	}
	// 空规则
	if _, err = ParseUpdateRule("lang:", ":"); err == nil {
		t.Fatal("empty rule should raise error")
	}
}

func TestParseUpdateFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".rtfd.ini")
	data := `[project]
latest = main
unused = 1

[sphinx]
lang = zh_CN
builder = dirhtml

[python]
version = 3.10
install = true

[other]
foo = bar
`
	if err := os.WriteFile(file, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	rule, md5, err := ParseUpdateFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if md5 == "" {
		t.Fatal("empty file md5")
	}
	for k, want := range map[string]any{
		"latest": "main", "lang": "zh_CN", "builder": "dirhtml",
		"version": "3.10", "install": "true",
	} {
		if rule[k] != want {
			t.Fatalf("parse file rule error: %v", rule)
		}
	}
	if _, ok := rule["unused"]; ok {
		t.Fatalf("field outside whitelist should be ignored: %v", rule)
	}
	if _, ok := rule["foo"]; ok {
		t.Fatalf("field outside whitelist should be ignored: %v", rule)
	}
	if _, _, err = ParseUpdateFile(filepath.Join(dir, "notfound.ini")); err == nil {
		t.Fatal("not found file should raise error")
	}
}

func TestTransfer(t *testing.T) {
	pm, _ := newTestPM(t)

	opt, err := pm.GenerateOption("demo", "https://github.com/staugur/rtfd")
	if err != nil {
		t.Fatal(err)
	}
	if err = opt.UpdateMeta("_webhook_id", "123"); err != nil {
		t.Fatal(err)
	}
	if err = opt.UpdateMeta("foo", "bar"); err != nil {
		t.Fatal(err)
	}
	if err = pm.Create("demo", opt); err != nil {
		t.Fatal(err)
	}

	// 默认导出剔除系统meta
	encode, err := pm.Export("demo", false)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pm.DecodeExport(encode)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "demo" || decoded.GetMeta("foo") != "bar" {
		t.Fatalf("export data error: %+v", decoded)
	}
	if decoded.GetMeta("_webhook_id") != "" {
		t.Fatal("system meta should be stripped")
	}

	// 导入（别名）后默认域名按当前系统配置重建
	name, err := pm.Import(decoded, "demo2")
	if err != nil {
		t.Fatal(err)
	}
	got, err := pm.GetName(name)
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultDomain != "demo2.example.com" || got.GetMeta("foo") != "bar" {
		t.Fatalf("import data error: %+v", got)
	}
	if _, err = pm.Import(decoded, "demo2"); err == nil {
		t.Fatal("import duplicated name should raise error")
	}

	// 含系统meta导出
	encode, err = pm.Export("demo", true)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = pm.DecodeExport(encode)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.GetMeta("_webhook_id") != "123" {
		t.Fatalf("system meta should be kept: %+v", decoded.Meta)
	}

	// 非法编码与不存在的项目
	if _, err = pm.DecodeExport("###"); err == nil {
		t.Fatal("invalid base64 should raise error")
	}
	if _, err = pm.Export("notexist", false); err == nil {
		t.Fatal("unknown project should raise error")
	}
}

func TestExcludedBranches(t *testing.T) {
	// 未设置：返回 [""]，不会命中任何非空分支
	opt := Options{Meta: map[string]string{}}
	if got := opt.ExcludedBranches(); len(got) != 1 || got[0] != "" {
		t.Fatalf("empty excluded_branch error: %v", got)
	}
	// 单个分支
	opt = Options{Meta: map[string]string{"excluded_branch": "master"}}
	if got := opt.ExcludedBranches(); len(got) != 1 || got[0] != "master" {
		t.Fatalf("single excluded_branch error: %v", got)
	}
	// 默认分隔符 |
	opt = Options{Meta: map[string]string{"excluded_branch": "master|dev"}}
	if got := opt.ExcludedBranches(); len(got) != 2 || got[0] != "master" || got[1] != "dev" {
		t.Fatalf("default sep error: %v", got)
	}
	// 自定义分隔符
	opt = Options{Meta: map[string]string{"excluded_branch": "master,dev", "excluded_sep": ","}}
	if got := opt.ExcludedBranches(); len(got) != 2 || got[0] != "master" || got[1] != "dev" {
		t.Fatalf("custom sep error: %v", got)
	}
}
