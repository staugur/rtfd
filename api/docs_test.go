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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSwaggerDocs 校验接口文档（Swagger UI 与规范文件）可访问，且规范内容与实现一致
func TestSwaggerDocs(t *testing.T) {
	e := newTestEcho(t)

	// UI 入口重定向到 index.html
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rtfd/docs", nil))
	if rec.Code != http.StatusMovedPermanently ||
		rec.Header().Get("Location") != "/rtfd/docs/index.html" {
		t.Fatalf("docs redirect error: %d %s", rec.Code, rec.Header().Get("Location"))
	}

	// UI 静态资源
	for _, path := range []string{
		"/rtfd/docs/index.html", "/rtfd/docs/swagger-ui.css", "/rtfd/docs/swagger-ui-bundle.js",
	} {
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != 200 || rec.Body.Len() == 0 {
			t.Fatalf("swagger ui asset error: %s %d", path, rec.Code)
		}
	}
	// UI 必须指向本服务提供的规范文件
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rtfd/docs/index.html", nil))
	if !strings.Contains(rec.Body.String(), "doc.json") {
		t.Fatalf("swagger ui index should point to doc.json: %s", rec.Body.String())
	}

	// 规范文件：包含公开/管理接口、安全定义与响应结构
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rtfd/docs/doc.json", nil))
	if rec.Code != 200 {
		t.Fatalf("swagger spec error: %d", rec.Code)
	}
	var spec struct {
		Info struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
		Paths               map[string]any `json:"paths"`
		SecurityDefinitions map[string]any `json:"securityDefinitions"`
		Definitions         map[string]any `json:"definitions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid swagger spec: %v", err)
	}
	if !strings.Contains(spec.Info.Title, "Rtfd") || spec.Info.Version == "" {
		t.Fatalf("invalid swagger info: %+v", spec.Info)
	}
	for _, path := range []string{
		"/rtfd/{name}/desc", "/rtfd/{name}/badge", "/rtfd/{name}/build", "/rtfd/{name}/webhook",
		"/rtfd/projects", "/rtfd/{name}/info", "/rtfd/{name}/update", "/rtfd/{name}/remove",
		"/rtfd/{name}/export", "/rtfd/import", "/rtfd/github/app",
	} {
		if _, ok := spec.Paths[path]; !ok {
			t.Fatalf("swagger path missing: %s", path)
		}
	}
	for _, name := range []string{"RtfdTs", "RtfdNonce", "RtfdSign"} {
		if _, ok := spec.SecurityDefinitions[name]; !ok {
			t.Fatalf("swagger securityDefinitions missing: %s", name)
		}
	}
	// Options 未定义 json tag，字段名应与实际响应保持一致（首字母大写）
	if _, ok := spec.Definitions["lib.Options"]; !ok {
		t.Fatal("swagger definitions missing: lib.Options")
	}
}
