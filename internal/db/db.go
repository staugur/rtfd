// 封装内嵌数据库

package db

import (
	"log"
	"rtfd/internal/conf"

	"github.com/xujiajun/nutsdb"
)

// DB 一个数据库连接结构
type DB struct {
	// rtfd.cfg default base_dir
	baseDir string
	// instance of nutsdb
	obj *nutsdb.DB
}

// New 打开一个DB连接，path是rtfd配置文件
func New(path string) (db *DB, err error) {
	cfg, err := conf.New(path)
	if err != nil {
		return
	}
	baseDir := cfg.GetKey("default", "base_dir")
	opt := nutsdb.DefaultOptions
	opt.Dir = baseDir
	connect, err := nutsdb.Open(opt)
	if err != nil {
		return
	}
	return &DB{baseDir, connect}, nil
}

// Close 关闭连接
func (db *DB) Close() {
	err := db.obj.Close()
	if err != nil {
		log.Fatal(err)
	}
}
