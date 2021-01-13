package build

import (
	"path/filepath"
	"time"

	"rtfd/pkg/conf"
	"rtfd/pkg/lib"
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

	baseDir := cfg.BaseDir()
	sh := filepath.Join(baseDir, "assets", "script", "builder.sh")
	return &Builder{path, sh, pm}, nil
}

/*

func (b *Builder) build(name, branch string, sender Sender) {
	if !b.has(name) {
		return "Did not find this project " + name
	}
	data := b.get(name)
	if branch == "latest" {
		branch = data["latest"]
	}

	// 响应信息
	status := "failing"
	usedtime := -1

	util.RunCmdStream("bash", b.sh, "-n", name, "-u", data["url"], "-b", branch, "-c", b.path)

		    for i in run_cmd_stream(*cmd):
		    if "Build Successfully" in i:
		        status = "passing"
		        try:
		            usedtime = int(i.split(" ")[2])
		        except (ValueError, TypeError):
		            pass
		    yield i
		#: 更新构建信息
		_build_info = {"_build_%s" % branch: dict(
		    btime=get_now(),
		    status=status,
		    sender=sender,
		    usedtime=usedtime,
		)}
		self._cpm.update(name, **_build_info)

}
*/
