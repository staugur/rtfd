package build

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"tcw.im/rtfd/pkg/conf"
	"tcw.im/rtfd/pkg/lib"

	"github.com/rakyll/statik/fs"
)

// Sender 发起构建来源类型
type Sender string

const (
	// APISender 从API接口发起构建
	APISender Sender = "api"
	// CLISender 从命令行发起构建
	CLISender Sender = "cli"
	// WebhookSender 从git webhook发起自动构建
	WebhookSender Sender = "webhook"
)

// Result 构建结果
type Result struct {
	// status 构建结果 passing表示true 其他表示false
	status   bool
	sender   string
	btime    time.Time
	usedtime uint
}

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
func (b *Builder) Build(name, branch string, sender Sender) error {
	if !b.pm.HasName(name) {
		return errors.New("not found name")
	}
	data, err := b.pm.GetName(name)
	if err != nil {
		return err
	}
	if branch == "" {
		branch = data.Latest
	}
	args := []string{b.sh, "-n", name, "-u", data.URL, "-b", branch, "-c", b.path}
	//status := "failing"
	//usedtime := -1
	if sender == "cli" {
		cmder := exec.Command("bash", args...)
		stdout, err := cmder.StdoutPipe()
		if err != nil {
			return err
		}
		cmder.Start()

		buf := bufio.NewReader(stdout) // not in a loop
		for {
			line, _, _ := buf.ReadLine()
			fmt.Println(string(line))
		}
	}

	return nil
}

func genBuilderScript(dir string) (sh string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}
	sh = filepath.Join(dir, ".rtfd-builder.sh")

	statikFS, err := fs.New()
	if err != nil {
		return
	}

	r, err := statikFS.Open("/builder.sh")
	if err != nil {
		return
	}
	defer r.Close()
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}

	err = ioutil.WriteFile(sh, content, 0644)
	if err != nil {
		return
	}

	return sh, nil
}
