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
		return ini.DEFAULT_SECTION
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

// BaseDir 获取base_dir（专项方法）
func (c Config) BaseDir() string {
	dir := c.GetKey(vars.DFT, "base_dir")
	if dir == "" {
		panic("base_dir is empty")
	}
	if strings.HasPrefix(dir, "~") {
		dir, err := homedir.Expand(dir)
		if err != nil {
			panic(err)
		}
		return dir
	}
	return dir
}
