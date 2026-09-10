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

// 查询配置

package conf

import (
	"regexp"
	"strings"

	"pkg/tcw.im/rtfd/pkg/util"
	"pkg/tcw.im/rtfd/vars"

	"gopkg.in/ini.v1"
)

// pyVersionPat 匹配py分区中的Python版本号（如3、3.10、3.12）
var pyVersionPat = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

// Config 封装程序操作ini配置文件的方法
type Config struct {
	path string
	obj  *ini.File
}

// New 初始化Config结构体
func New(configPath string) (cfg *Config, err error) {
	configPath = util.ExpandPath(configPath)
	obj, err := ini.Load(configPath)
	if err != nil {
		return
	}
	return &Config{configPath, obj}, nil
}

func changeDefaultSection(section string) string {
	if strings.ToLower(section) == vars.DFT {
		return ini.DefaultSection
	}
	return section
}

// SecHash 获取ini文件某个分区下所有经过解析的键值对
func (c Config) SecHash(section string) (data map[string]string) {
	section = changeDefaultSection(section)
	data = make(map[string]string)
	for _, k := range c.obj.Section(section).KeyStrings() {
		data[k] = c.obj.Section(section).Key(k).String()
	}
	return
}

// GetKey 获取分区下某个键的值
func (c Config) GetKey(section, key string) string {
	section = changeDefaultSection(section)
	return c.obj.Section(section).Key(key).String()
}

// MustKey 获取分区下某个键的值，可设置默认值
func (c Config) MustKey(section, key, defaults string) string {
	v := c.GetKey(section, key)
	if v == "" {
		v = defaults
	}
	return v
}

// AllHash 获取ini文件所有分区的经过解析的键值对
func (c Config) AllHash() (data map[string]map[string]string) {
	data = make(map[string]map[string]string)
	for _, s := range c.obj.SectionStrings() {
		hash := make(map[string]string)
		for _, k := range c.obj.Section(s).KeyStrings() {
			hash[k] = c.obj.Section(s).Key(k).String()
		}
		data[s] = hash
	}
	return
}

// GetPath 封装 GetKey 结果，如果值以 ~ 开头，替换为家目录
func (c Config) GetPath(section, key string) (string, error) {
	return util.ExpandPath(c.GetKey(section, key)), nil
}

// MustPath 可设置默认值的 GetPath
func (c Config) MustPath(section, key, defaults string) string {
	v, _ := c.GetPath(section, key)
	if v == "" {
		v = defaults
	}
	return v
}

// BaseDir 获取base_dir（专项方法）
func (c Config) BaseDir() string {
	dir, err := c.GetPath(vars.DFT, "base_dir")
	if err != nil {
		panic(err)
	}
	if dir == "" {
		panic("base_dir is empty")
	}
	return dir
}

func (c Config) DefaultBranch() string {
	return c.MustKey(vars.DFT, "default_branch", "master")
}

// PyVersions 获取py分区中已配置的Python版本（键是版本号，值是对应的python程序）
func (c Config) PyVersions() (versions []string) {
	for _, k := range c.obj.Section("py").KeyStrings() {
		if pyVersionPat.MatchString(k) {
			versions = append(versions, k)
		}
	}
	return
}

// HasPyVersion 判断是否配置了某个Python版本
func (c Config) HasPyVersion(version string) bool {
	for _, v := range c.PyVersions() {
		if v == version {
			return true
		}
	}
	return false
}

// PyCommand 获取某个Python版本对应的程序（命令或绝对路径），未配置时返回空字符串
func (c Config) PyCommand(version string) string {
	if version == "" {
		return ""
	}
	if v := c.GetKey("py", version); v != "" {
		return v
	}
	// 兼容旧版本配置中的 py3 键
	if version == vars.DefaultPy {
		return c.GetKey("py", "py3")
	}
	return ""
}

// DefaultPyVersion 获取默认的Python版本：
// py分区default指定的版本，未配置或无效时取第一个可用版本，都不存在时返回 vars.DefaultPy
func (c Config) DefaultPyVersion() string {
	if v := c.GetKey("py", "default"); v != "" && c.HasPyVersion(v) {
		return v
	}
	if versions := c.PyVersions(); len(versions) > 0 {
		return versions[0]
	}
	return vars.DefaultPy
}
