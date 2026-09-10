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
	"encoding/json"
	"testing"
)

func TestPyVerUnmarshalJSON(t *testing.T) {
	cases := []struct {
		data string
		want PyVer
	}{
		// 历史数据以数字存储
		{`{"Version":3}`, PY3},
		{`{"Version":2}`, PY3},
		{`{"Version":"2"}`, PY3},
		{`{"Version":"2.7"}`, PY3},
		{`{"Version":null}`, PY3},
		{`{"Version":""}`, PY3},
		// 多版本
		{`{"Version":"3.10"}`, PyVer("3.10")},
		{`{"Version":"3.12"}`, PyVer("3.12")},
	}
	for _, c := range cases {
		var opt Options
		if err := json.Unmarshal([]byte(c.data), &opt); err != nil {
			t.Fatalf("unmarshal %s error: %v", c.data, err)
		}
		if opt.Version != c.want {
			t.Fatalf("unmarshal %s got %s, want %s", c.data, opt.Version, c.want)
		}
	}
}

func TestOptionKeyMap(t *testing.T) {
	if OptionKeyMap("version") != "Version" {
		t.Fatal("option key map version error")
	}
	if OptionKeyMap("sourcedir") != "SourceDir" {
		t.Fatal("option key map sourcedir error")
	}
}
