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

// 数据存储层：基于GORM的关系型数据库存储，支持sqlite、mysql、pgsql

package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type (
	// DBType 数据库类型
	DBType string
)

const (
	// SQLite sqlite数据库
	SQLite DBType = "sqlite"
	// MySQL mysql或mariadb数据库
	MySQL DBType = "mysql"
	// PostgreSQL pgsql数据库
	PostgreSQL DBType = "pgsql"
)

// Project 文档项目，是 lib.Options 的关系化存储
type Project struct {
	ID     uint   `gorm:"primaryKey;autoIncrement"`
	Name   string `gorm:"uniqueIndex:idx_project_name;size:100;not null"`
	URL    string `gorm:"size:512;not null"`
	Latest string `gorm:"size:128;not null"`
	// Version 构建所用的python版本，如3、3.10
	Version   string `gorm:"size:16;not null"`
	Single    bool
	SourceDir string `gorm:"size:255;not null"`
	Lang      string `gorm:"size:255;not null"`
	// Requirement 依赖文件，逗号分隔
	Requirement string `gorm:"size:512"`
	Install     bool
	Index       string `gorm:"size:512"`
	ShowNav     bool
	HideGit     bool
	Secret      string `gorm:"size:128"`
	// DefaultDomain 默认域名，{name}.{nginx.dn}
	DefaultDomain string `gorm:"size:255;not null"`
	// CustomDomain 自定义域名，空表示未设置
	CustomDomain string `gorm:"index:idx_custom_domain;size:255"`
	SSL          bool
	SSLPublic    string `gorm:"size:512"`
	SSLPrivate   string `gorm:"size:512"`
	Builder      string `gorm:"size:32;not null"`
	GSP          string `gorm:"size:32"`
	IsPublic     bool
	Meta         map[string]string `gorm:"serializer:json;type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName 项目表名
func (Project) TableName() string {
	return "projects"
}

// BuildResult 文档项目的构建结果（按项目+分支唯一）
type BuildResult struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	Project  string `gorm:"uniqueIndex:idx_build_project_branch;size:100;not null"`
	Branch   string `gorm:"uniqueIndex:idx_build_project_branch;size:255;not null"`
	Status   bool
	Sender   string `gorm:"size:16"`
	Btime    string `gorm:"size:32"`
	Usedtime int
	// CreatedAt 首次构建时间
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName 构建结果表名
func (BuildResult) TableName() string {
	return "build_results"
}

// ParseDBType 解析并标准化数据库类型
func ParseDBType(typ string) (DBType, error) {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "sqlite", "sqlite3":
		return SQLite, nil
	case "mysql", "mariadb":
		return MySQL, nil
	case "pgsql", "postgres", "postgresql":
		return PostgreSQL, nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", typ)
	}
}

// Open 打开数据库连接，并自动迁移表结构
// - typ: sqlite / mysql / pgsql
// - dsn: sqlite为文件路径，mysql与pgsql为对应驱动的连接串
// - debug: 是否打印SQL
func Open(typ, dsn string, debug bool) (*gorm.DB, error) {
	t, err := ParseDBType(typ)
	if err != nil {
		return nil, err
	}
	if dsn == "" {
		return nil, errors.New("empty database dsn")
	}

	var dialector gorm.Dialector
	switch t {
	case SQLite:
		if !isMemorySQLite(dsn) {
			dir := filepath.Dir(dsn)
			if err = os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
		}
		dialector = sqlite.Open(dsn)
	case MySQL:
		dialector = mysql.Open(dsn)
	case PostgreSQL:
		dialector = postgres.Open(dsn)
	}

	logLevel := logger.Warn
	if debug {
		logLevel = logger.Info
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}
	if t == SQLite && !isMemorySQLite(dsn) {
		// 提升并发读写表现
		db.Exec("PRAGMA busy_timeout = 5000")
	}

	if err = db.AutoMigrate(&Project{}, &BuildResult{}); err != nil {
		return nil, err
	}
	return db, nil
}

// Close 关闭数据库连接
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func isMemorySQLite(dsn string) bool {
	return strings.Contains(dsn, ":memory:") || strings.HasPrefix(dsn, "file:") && strings.Contains(dsn, "mode=memory")
}
