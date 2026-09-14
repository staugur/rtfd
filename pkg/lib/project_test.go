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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// newTestPM 使用sqlite创建用于测试的项目管理器
func newTestPM(t *testing.T) (*ProjectManager, string) {
	t.Helper()
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "rtfd.cfg")
	data := fmt.Sprintf(`
base_dir = %s

[database]
type = sqlite
dsn = %s/rtfd.db

[caddy]
dn = example.com
exec = true

[py]
3 = python3
3.10 = python3
`, dir, dir)
	if err := os.WriteFile(cfgFile, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	pm, err := New(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pm.Close() })
	return pm, dir
}

func TestProjectCRUD(t *testing.T) {
	pm, _ := newTestPM(t)

	if pm.HasName("demo") {
		t.Fatal("project should not exist")
	}
	opt, err := pm.GenerateOption("Demo", "https://github.com/staugur/rtfd")
	if err != nil {
		t.Fatal(err)
	}
	if err = pm.Create("demo", opt); err != nil {
		t.Fatal(err)
	}
	if !pm.HasName("demo") {
		t.Fatal("project should exist after create")
	}
	// 重复创建应失败
	if err = pm.Create("demo", opt); err == nil {
		t.Fatal("duplicated project should raise error")
	}

	got, err := pm.GetName("demo")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "demo" || got.URL != "https://github.com/staugur/rtfd" {
		t.Fatalf("project data error: %+v", got)
	}
	if got.Version != PY3 {
		t.Fatalf("default version error: %s", got.Version)
	}

	// 更新字段
	ok, fail, err := pm.Update(&got, map[string]any{"lang": "zh_CN", "single": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ok) != 2 || len(fail) != 0 {
		t.Fatalf("update result error: ok=%v fail=%v", ok, fail)
	}
	got2, err := pm.GetName("demo")
	if err != nil {
		t.Fatal(err)
	}
	if got2.Lang != "zh_CN" || !got2.Single {
		t.Fatalf("project not updated: %+v", got2)
	}

	// meta 与写回
	if err = got2.UpdateMeta("foo", "bar"); err != nil {
		t.Fatal(err)
	}
	if err = got2.Writeback(pm); err != nil {
		t.Fatal(err)
	}
	got3, _ := pm.GetName("demo")
	if got3.GetMeta("foo") != "bar" {
		t.Fatal("meta not persisted")
	}

	// 构建结果
	if err = pm.BuildRecord("demo", "master", Result{
		Branch: "master", Status: true, Sender: "cli", Btime: "2026-09-10 00:00:00", Usedtime: 8,
	}); err != nil {
		t.Fatal(err)
	}
	rst, err := pm.GetBuildset("demo", "master")
	if err != nil {
		t.Fatal(err)
	}
	if !rst.Status || rst.Usedtime != 8 {
		t.Fatalf("buildset error: %+v", rst)
	}
	// 同分支重复记录应更新而非新增
	if err = pm.BuildRecord("demo", "master", Result{
		Branch: "master", Status: false, Sender: "api", Btime: "2026-09-10 00:01:00", Usedtime: 9,
	}); err != nil {
		t.Fatal(err)
	}
	builders, err := pm.ListBuildset("demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(builders) != 1 || builders[0].Usedtime != 9 {
		t.Fatalf("buildset should be upsert: %+v", builders)
	}
	if _, err = pm.GetBuildset("demo", "v1.0"); err == nil {
		t.Fatal("unknown branch should raise error")
	}

	// 列表与详情
	list, err := pm.ListProject()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != "demo" {
		t.Fatalf("list project error: %v", list)
	}
	full, err := pm.ListFullProject()
	if err != nil {
		t.Fatal(err)
	}
	if len(full) != 1 {
		t.Fatal("list full project error")
	}
	if _, err = pm.GetNameWithBuildset("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err = pm.GetSourceName("demo"); err != nil {
		t.Fatal(err)
	}

	// 删除
	if err = pm.Remove("demo"); err != nil {
		t.Fatal(err)
	}
	if pm.HasName("demo") {
		t.Fatal("project should be removed")
	}
	list, _ = pm.ListProject()
	if len(list) != 0 {
		t.Fatalf("project list should be empty: %v", list)
	}
	if err = pm.Remove("demo"); err == nil {
		t.Fatal("remove unknown project should raise error")
	}
}

func TestProjectCustomDomain(t *testing.T) {
	pm, _ := newTestPM(t)

	opt, err := pm.GenerateOption("site1", "https://github.com/staugur/rtfd")
	if err != nil {
		t.Fatal(err)
	}
	pm.SetOption(&opt, "CustomDomain", "docs.example.com")
	if err = pm.Create("site1", opt); err != nil {
		t.Fatal(err)
	}
	if !pm.HasCustomDomain("docs.example.com") {
		t.Fatal("custom domain should be occupied")
	}

	opt2, err := pm.GenerateOption("site2", "https://github.com/staugur/rtfd")
	if err != nil {
		t.Fatal(err)
	}
	pm.SetOption(&opt2, "CustomDomain", "docs.example.com")
	// 域名已被占用，创建应失败
	if err = pm.Create("site2", opt2); err == nil {
		t.Fatal("duplicated custom domain should raise error")
	}
	// 取消自定义域名后应释放
	opt3, err := pm.GetName("site1")
	if err != nil {
		t.Fatal(err)
	}
	ok, _, err := pm.Update(&opt3, map[string]any{"domain": "false"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ok) != 1 {
		t.Fatalf("clear domain error: %v", ok)
	}
	if pm.HasCustomDomain("docs.example.com") {
		t.Fatal("custom domain should be released")
	}
}

func TestInitCaddy(t *testing.T) {
	pm, dir := newTestPM(t)
	// 即使没有任何项目，启动初始化也应生成 Caddyfile
	pm.InitCaddy()
	if _, err := os.Stat(filepath.Join(dir, "caddy", "Caddyfile")); err != nil {
		t.Fatal("caddyfile should be created on init:", err)
	}
}
