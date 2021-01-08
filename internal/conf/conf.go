// 查询配置

package conf

import (
	"gopkg.in/ini.v1"
)

// Config 封装程序操作ini配置文件的方法
type Config struct {
	path string
	obj  *ini.File
}

// New 初始化Config结构体
func New(cfg string) (c *Config, err error) {
	obj, err := ini.Load(cfg)
	if err != nil {
		return
	}
	c = &Config{cfg, obj}
	return
}

// SecHash 获取ini文件某个分区下所有经过解析的键值对
func (c Config) SecHash(section string) (data map[string]string) {
	data = make(map[string]string)
	for _, k := range c.obj.Section(section).KeyStrings() {
		data[k] = c.obj.Section(section).Key(k).String()
	}
	return
}

// GetKey 获取分区下某个键的值
func (c Config) GetKey(section, key string) string {
	return c.obj.Section(section).Key(key).String()
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
