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

// 项目配置的转储（导出、导入），CLI 与 API 共用

package lib

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// Export 导出项目配置，返回base64编码的字符串；
// sysMeta为false时剔除内置meta（以下划线开头），便于跨环境导入
func (pm *ProjectManager) Export(name string, sysMeta bool) (string, error) {
	opt, err := pm.GetName(name)
	if err != nil {
		return "", err
	}
	if !sysMeta {
		meta := make(map[string]string)
		for k, v := range opt.Meta {
			if !strings.HasPrefix(k, "_") {
				meta[k] = v
			}
		}
		opt.Meta = meta
	}
	val, err := json.Marshal(opt)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(val), nil
}

// DecodeExport 解析（导出的）base64字符串为项目配置
func (pm *ProjectManager) DecodeExport(encoded string) (opt Options, err error) {
	val, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return opt, errors.New("import decode fail")
	}
	if err = json.Unmarshal(val, &opt); err != nil {
		return
	}
	if opt.Name == "" {
		err = errors.New("empty project name")
	}
	return
}

// Import 导入项目配置，name为空表示使用配置中的名称，
// 返回实际使用的项目名；注意域名不可跨环境复用，此处按当前系统配置重建默认域名
func (pm *ProjectManager) Import(opt Options, name string) (string, error) {
	if name == "" {
		name = opt.Name
	}
	name = strings.ToLower(name)
	dn := pm.cfg.GetKey("sws", "dn")
	if dn == "" {
		return name, errors.New("invalid sws dn")
	}
	if pm.HasName(name) {
		return name, errors.New("the name already exists")
	}
	opt.Name = name
	opt.DefaultDomain = name + "." + dn

	if err := pm.Create(name, opt); err != nil {
		return name, err
	}
	return name, nil
}
