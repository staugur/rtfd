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

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"pkg.tcw.im/gtc"
)

const testAPISecret = "testsecret"

// newTestEcho 使用临时配置（sqlite存储）创建API服务实例
func newTestEcho(t *testing.T) *echo.Echo {
	t.Helper()
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "rtfd.cfg")
	data := fmt.Sprintf(`
base_dir = %s

[api]
secret = %s

[database]
type = sqlite
dsn = %s/rtfd.db

[nginx]
dn = example.com
exec = true

[py]
3 = python3
`, dir, testAPISecret, dir)
	if err := os.WriteFile(cfgFile, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	e, err := New(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// doReq 发起请求并返回状态码与JSON响应体
func doReq(t *testing.T, e *echo.Echo, method, target string, form url.Values, sign string) (int, map[string]any) {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, target, body)
	if form != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	}
	if sign != "" {
		req.Header.Set("X-Rtfd-Sign", sign)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var data map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &data); err != nil {
		t.Fatalf("invalid response body: %s", rec.Body.String())
	}
	return rec.Code, data
}

func TestManageAPI(t *testing.T) {
	e := newTestEcho(t)
	sign := gtc.MD5(testAPISecret)

	// 未携带密钥与错误密钥均应被拒绝
	if _, data := doReq(t, e, http.MethodGet, "/rtfd/projects", nil, ""); data["success"] == true {
		t.Fatalf("should require api secret: %v", data)
	}
	if _, data := doReq(t, e, http.MethodGet, "/rtfd/projects", nil, "bad"); data["success"] == true {
		t.Fatalf("invalid sign should be rejected: %v", data)
	}

	// 创建项目
	form := url.Values{
		"name": {"Demo"}, "url": {"https://github.com/staugur/rtfd"},
		"lang": {"zh_CN"}, "single": {"on"},
	}
	code, data := doReq(t, e, http.MethodPost, "/rtfd/projects", form, sign)
	if code != 201 || data["success"] != true {
		t.Fatalf("create project failed: %v", data)
	}
	// 重复创建应失败
	if _, data = doReq(t, e, http.MethodPost, "/rtfd/projects", form, sign); data["success"] == true {
		t.Fatal("duplicated project should be rejected")
	}

	// 列表
	_, data = doReq(t, e, http.MethodGet, "/rtfd/projects", nil, sign)
	members, ok := data["data"].([]any)
	if !ok || len(members) != 1 || members[0] != "demo" {
		t.Fatalf("list project error: %v", data)
	}
	_, data = doReq(t, e, http.MethodGet, "/rtfd/projects?verbose=1", nil, sign)
	if len(data["data"].([]any)) != 1 {
		t.Fatalf("list verbose project error: %v", data)
	}

	// 详情、字段与构建集（详情返回 Options 结构体，字段名为结构体字段名）
	_, data = doReq(t, e, http.MethodGet, "/rtfd/demo/info", nil, sign)
	info, ok := data["data"].(map[string]any)
	if !ok || info["Lang"] != "zh_CN" || info["Single"] != true {
		t.Fatalf("project info error: %v", data)
	}
	_, data = doReq(t, e, http.MethodGet, "/rtfd/info/demo?key=lang", nil, sign)
	if data["data"].(map[string]any)["value"] != "zh_CN" {
		t.Fatalf("project option error: %v", data)
	}
	_, data = doReq(t, e, http.MethodGet, "/rtfd/demo/info?build=1", nil, sign)
	if _, ok = data["data"].(map[string]any)["Buildset"]; !ok {
		t.Fatalf("project buildset error: %v", data)
	}
	if _, data = doReq(t, e, http.MethodGet, "/rtfd/notexist/info", nil, sign); data["success"] == true {
		t.Fatal("unknown project should be not found")
	}

	// 更新（text方式）
	_, data = doReq(t, e, http.MethodPost, "/rtfd/demo/update",
		url.Values{"text": {"lang:en,builder:dirhtml"}}, sign)
	if data["success"] != true || len(data["updated"].([]any)) != 2 {
		t.Fatalf("update project error: %v", data)
	}
	// 更新（直接字段方式）
	_, data = doReq(t, e, http.MethodPost, "/rtfd/update/demo",
		url.Values{"install": {"on"}}, sign)
	if data["success"] != true {
		t.Fatalf("update project by field error: %v", data)
	}
	_, data = doReq(t, e, http.MethodGet, "/rtfd/demo/info?key=install", nil, sign)
	if data["data"].(map[string]any)["value"] != "true" {
		t.Fatalf("project not updated: %v", data)
	}
	// 更新（非法字段会出现在failed中）
	_, data = doReq(t, e, http.MethodPost, "/rtfd/demo/update",
		url.Values{"text": {"notexist:1"}}, sign)
	if len(data["failed"].([]any)) != 1 {
		t.Fatalf("invalid field should be in failed: %v", data)
	}

	// 项目密钥也可访问项目级接口
	if _, data = doReq(t, e, http.MethodPost, "/rtfd/demo/update",
		url.Values{"secret": {"ownsecret"}}, sign); data["success"] != true {
		t.Fatalf("set project secret error: %v", data)
	}
	if _, data = doReq(t, e, http.MethodGet, "/rtfd/demo/info", nil, gtc.MD5("ownsecret")); data["success"] != true {
		t.Fatalf("project secret should be accepted: %v", data)
	}

	// 导出与导入
	_, data = doReq(t, e, http.MethodGet, "/rtfd/demo/export", nil, sign)
	encoded, ok := data["data"].(map[string]any)["export"].(string)
	if !ok || encoded == "" {
		t.Fatalf("export project error: %v", data)
	}
	code, data = doReq(t, e, http.MethodPost, "/rtfd/projects",
		url.Values{"name": {"copy"}, "url": {"https://gitee.com/staugur/rtfd"}}, sign)
	if code != 201 {
		t.Fatalf("create project for import error: %v", data)
	}
	code, data = doReq(t, e, http.MethodPost, "/rtfd/import",
		url.Values{"export": {encoded}, "name": {"demo2"}}, sign)
	if code != 201 || data["success"] != true {
		t.Fatalf("import project error: %v", data)
	}
	if _, data = doReq(t, e, http.MethodGet, "/rtfd/demo2/info", nil, sign); data["success"] != true {
		t.Fatalf("imported project not found: %v", data)
	}

	// 删除（POST与DELETE两种方式）
	if _, data = doReq(t, e, http.MethodPost, "/rtfd/remove/demo", nil, sign); data["success"] != true {
		t.Fatalf("remove project error: %v", data)
	}
	if _, data = doReq(t, e, http.MethodDelete, "/rtfd/demo2/remove", nil, sign); data["success"] != true {
		t.Fatalf("delete project error: %v", data)
	}
	_, data = doReq(t, e, http.MethodGet, "/rtfd/projects", nil, sign)
	if len(data["data"].([]any)) != 1 {
		t.Fatalf("project list error: %v", data)
	}
}
