/* 封装内嵌数据库 NutsDB

使用 桶 的概念表示项目，每个文档项目都是一个桶，
项目的数据放到 桶 下的各个不同数据类型的key中。
*/

package db

import (
	"log"
	"path/filepath"
	"rtfd/internal/conf"

	"github.com/xujiajun/nutsdb"
)

// Bucket 即 NutsDB 桶，表示一个文档项目名称
type Bucket = string

// DB 一个数据库连接结构
type DB struct {
	// subdir in rtfd.cfg base_dir
	DBDir string
	// instance of nutsdb
	obj *nutsdb.DB
}

// New 打开一个DB连接，path是rtfd配置文件
func New(path string) (db *DB, err error) {
	cfg, err := conf.New(path)
	if err != nil {
		return
	}
	DBDir := filepath.Join(cfg.BaseDir(), "db")
	opt := nutsdb.DefaultOptions
	opt.Dir = DBDir
	connect, err := nutsdb.Open(opt)
	if err != nil {
		return
	}
	return &DB{DBDir, connect}, nil
}

// Close 关闭连接
func (db *DB) Close() {
	err := db.obj.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// Set 添加数据
func (db *DB) Set(name Bucket, key, value []byte) error {
	if err := db.obj.Update(
		func(tx *nutsdb.Tx) error {
			if err := tx.Put(name, key, value, 0); err != nil {
				return err
			}
			return nil
		}); err != nil {
		return err
	}
	return nil
}

// Get 获取数据
func (db *DB) Get(name Bucket, key []byte) (value string, err error) {
	if err = db.obj.View(
		func(tx *nutsdb.Tx) error {
			e, err := tx.Get(name, key)
			if err != nil {
				return err
			}
			value = string(e.Value)
			return nil
		}); err != nil {
		return
	}
	return
}

// RPush 从指定bucket里面的指定队列key的右边入队一个或者多个元素value
func (db *DB) RPush(name Bucket, key, value []byte) error {
	if err := db.obj.Update(
		func(tx *nutsdb.Tx) error {
			return tx.RPush(name, key, value)
		}); err != nil {
		return err
	}
	return nil
}
