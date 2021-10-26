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

package build

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tcw.im/rtfd/assets"
	"tcw.im/rtfd/pkg/conf"
	"tcw.im/rtfd/pkg/lib"
	"tcw.im/rtfd/pkg/util"
	"tcw.im/rtfd/vars"
)

// Builder 构建器
type Builder struct {
	// 配置文件路径
	path string
	// sh 构建脚本路径
	sh string
	// 项目管理器
	pm *lib.ProjectManager
}

// New 新建构建器实例
func New(path string) (b *Builder, err error) {
	cfg, err := conf.New(path)
	if err != nil {
		return
	}
	pm, err := lib.New(path)
	if err != nil {
		return
	}

	sh, err := genBuilderScript(cfg.BaseDir())
	if err != nil {
		return
	}
	return &Builder{path, sh, pm}, nil
}

// Build 默认方式构建文档
func (b *Builder) Build(name, branch string, sender vars.Sender) error {
	return b.build(name, branch, sender, false, false)
}

// BuildWithDebug 调试方式构建文档
func (b *Builder) BuildWithDebug(name, branch string, sender vars.Sender) error {
	return b.build(name, branch, sender, true, false)
}

// BuildWithLog 构建文档时记录日志（cli方式除外）
func (b *Builder) BuildWithLog(name, branch string, sender vars.Sender) error {
	return b.build(name, branch, sender, false, true)
}

// BuildWithAll 以调试模式构建文档并记录日志
func (b *Builder) BuildWithAll(name, branch string, sender vars.Sender) error {
	return b.build(name, branch, sender, true, true)
}

// build 构建文档
// - isDebug 则以 `bash -x` 模式调试运行
// - isLog 则对脚本每行输出记录日志
func (b *Builder) build(name, branch string, sender vars.Sender, isDebug bool, isLog bool) error {
	if !b.pm.HasName(name) {
		return errors.New("not found project")
	}
	data, err := b.pm.GetName(name)
	if err != nil {
		return err
	}
	if branch == "" {
		branch = data.Latest
	}
	var args []string
	if isDebug {
		args = []string{"-x", b.sh, "-n", name, "-b", branch, "-c", b.path}
	} else {
		args = []string{b.sh, "-n", name, "-b", branch, "-c", b.path}
	}

	status := false
	usedtime := -1
	util.RunCmdStream("bash", args, func(line string) {
		if sender == vars.CLISender {
			fmt.Printf(line)
		} else if isLog {
			log.Printf(line)
		}
		if strings.HasPrefix(line, "Build Successfully") {
			status = true
			stime := strings.Split(line, " ")[2]
			itime, _ := strconv.Atoi(stime)
			if itime > 0 {
				usedtime = itime
			}
		}
	})
	rst := lib.Result{
		Status: status, Sender: sender, Usedtime: usedtime,
		Btime: util.GetNow(), Branch: branch,
	}
	err = b.pm.BuildRecord(name, branch, rst)
	if err != nil {
		return err
	}
	return nil
}

func genBuilderScript(dir string) (sh string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}
	sh = filepath.Join(dir, ".rtfd-builder.sh")

	err = ioutil.WriteFile(sh, assets.BuiderSH, 0644)
	if err != nil {
		return
	}

	return sh, nil
}
