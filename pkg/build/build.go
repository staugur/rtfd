package build

import (
	"errors"
	"fmt"
	"io/ioutil"
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

// Build 构建文档
func (b *Builder) Build(name, branch string, sender vars.Sender) error {
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
	args := []string{b.sh, "-n", name, "-b", branch, "-c", b.path}
	status := false
	usedtime := -1
	util.RunCmdStream("bash", args, func(line string) {
		if sender == vars.CLISender {
			fmt.Printf(line)
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
