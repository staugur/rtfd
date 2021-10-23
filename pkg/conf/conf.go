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
	"strings"

	"tcw.im/rtfd/vars"

	homedir "github.com/mitchellh/go-homedir"
	"gopkg.in/ini.v1"
)

// Config 封装程序操作ini配置文件的方法
type Config struct {
	path string
	obj  *ini.File
}

// New 初始化Config结构体
func New(configPath string) (cfg *Config, err error) {
	if strings.HasPrefix(configPath, "~") {
		configPath, err = homedir.Expand(configPath)
		if err != nil {
			return
		}
	}
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
	v := c.GetKey(section, key)
	if strings.HasPrefix(v, "~") {
		return homedir.Expand(v)
	}
	return v, nil
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
