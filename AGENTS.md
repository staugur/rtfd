# rtfd — 项目上下文

rtfd 是一个自托管的 **Sphinx 文档构建与托管服务**（单二进制 Go 程序）：
用户提供 git 仓库（GitHub/Gitee），rtfd 按语言/分支用 Sphinx 构建 HTML，
写回数据库元数据并交由 static-web-server 静态托管；支持 CLI、HTTP API、Git 仓库 Webhook、
GitHub App 自动注册 webhook 与文档状态徽章。

技术栈：Go 1.26，spf13/cobra（CLI）、labstack/echo v4（API）、GORM（存储，支持 sqlite/mysql/pgsql）、
gopkg.in/ini.v1（配置）、go:embed（assets 打包）、
golang-jwt/jwt/v5（GitHub App 身份）。
运行时依赖（仅 Linux）：bash、git、python3（含 pip、virtualenv，支持配置多版本，已移除 python2）、static-web-server；
元数据存储使用 sqlite / mysql / pgsql（默认 sqlite，纯 Go 驱动，无需外部服务）。
核心构建逻辑在 bash 脚本（`assets/builder.sh`）；生成文档页注入 `assets/rtfd.js` 浮动导航挂件。
详细架构见 `ARCH.md`。

## 目录结构

```
rtfd/
├── main.go             入口（空白导入 assets 激活 embed + cmd.Execute）
├── main_test.go        默认配置结构断言（新增配置项需同步）
├── api/                echo API 层：api.go(路由/Start) view.go(处理) tool.go(公共)
├── assets/             go:embed 资源：rtfd.cfg / builder.sh / rtfd.js / VERSION / swagger.html / rtfd.ini(样例)
├── cmd/                cobra 命令：root/api/cfg/build/project(+create/get/list/remove/transfer/update)
├── pkg/
│   ├── build/          Builder：bash 调用、输出流解析、结果入库
│   ├── conf/           ini 配置封装
│   ├── lib/            核心业务：lib.go(ProjectManager/CRUD) sws.go(SWS 模板) update.go(更新钩子) app.go(GitHub App)
│   ├── store/          ★ 存储层：GORM 模型(Project/BuildResult) 与多方言连接、自动迁移
│   └── util/           纯工具（命令执行/校验/git url/hmac）
├── vars/               跨包常量与全局类型
├── scripts/            部署脚本（supervisord/systemd）
└── Makefile / Dockerfile / .github/workflows/
```

## Go 编码规范

### 版本与废弃用法

- go.mod 的 go 指令与 CI/Dockerfile 保持 **Go 1.26**；新增代码可用 1.18+ 语法
- 空接口统一写 `any`，不写 `interface{}`
- **禁止使用已废弃/停维护的包与函数**（见下表），替代方案已在代码中使用：

| 废弃项 | 替代 |
|---|---|
| `strings.Title` | `util.TitleCase` |
| `github.com/mitchellh/go-homedir`（已归档） | `os.UserHomeDir` / `util.ExpandPath` |
| `github.com/dgrijalva/jwt-go`（停维护） | `github.com/golang-jwt/jwt/v5` |
| `io/ioutil` | `os` / `io` |
| `fmt.Printf(非恒定串)`、`log.Printf(非恒定串)` | `fmt.Print` / `log.Print` |

### 基本风格

- 采用官方 `go fmt` 格式；**每个 `.go` 文件顶部必须保留 Apache-2.0 License 头**（见现有文件）
- 缩进 Tab；文件末尾换行；单行不过度堆叠
- 包名小写单词；main 与各 package 均为单一职责目录

### 导入顺序

按三组排列，组间空行（此为仓库主导风格，个别旧文件有偏差，新代码遵循）：

1. 标准库
2. 项目内部：`pkg.tcw.im/rtfd/v2/...`
3. 外部依赖：`github.com/*`、`pkg.tcw.im/*`

```go
import (
	"encoding/json"
	"fmt"
	"os"

	"pkg.tcw.im/rtfd/v2/pkg/conf"
	"pkg.tcw.im/rtfd/v2/pkg/lib"
	"pkg.tcw.im/rtfd/v2/vars"

	"github.com/labstack/echo/v4"
	"pkg.tcw.im/gtc"
)
```

### 命名规范

| 类型 | 风格 | 示例 |
|---|---|---|
| 包/文件 | 小写单词（多词不强制） | `pkg/build`、`sws.go` |
| 导出类型/函数 | PascalCase | `ProjectManager`、`OptionKeyMap` |
| 非导出函数/变量 | camelCase | `genBuilderScript`、`updateHook` |
| 方法接收者 | 单字母缩写 | `pm *ProjectManager`、`u *updateHook` |
| 全局常量 | UPPER_SNAKE_CASE | `ResetEmpty`、`PY3`、`SQLite` |
| 跨包常量 | 统一放 `vars` 包 | `APISender`、`GSPGitHub`、`PUFMD5` |
| 类型别名定义 | `type` 分组块 | `type PyVer uint8` |

### 存储层约定（`pkg/store`）

- 一律通过 `pkg/store` 的 GORM 模型访问数据库：`Project`（项目配置）、`BuildResult`（构建结果），
  由 `store.Open(type, dsn, debug)` 打开连接并 `AutoMigrate`
- 新增字段：改 `store.Project/BuildResult` 模型即可（自动迁移），**同时**更新 `lib` 中的
  `projectFromOptions / optionsFromProject` 转换，否则字段会静默丢失
- 不要再引入 Redis 等 KV 存储；项目名统一小写后查询（`strings.ToLower`）
- 多库兼容：SQL 只用 GORM 表达，不写方言专属语法；sqlite 驱动为纯 Go（`glebarez/sqlite`），
  以便 `CGO_ENABLED=0` 交叉编译
- 数据库配置从 `[database]` 分区读取（`conf.DatabaseType/DatabaseDSN`）；是否打印 SQL 由 `conf.DatabaseDebug()`
  决定，而它不再读 `[database] debug`，改为读取顶层（default）`log_level`，值为 `debug` 时开启

### 结构体与类型

- 业务结构（`Options`、`Result`）集中定义于所属 package 顶部 `type` 块；
  持久化结构（`store.Project`、`store.BuildResult`）定义于 `pkg/store`
- Options/Result 与模型之间通过 `projectFromOptions/optionsFromProject`、`resultFromBuildResult`
  转换；**新增/改名 Options 字段必须同步模型与转换函数**
- 查询一律把项目名 `strings.ToLower` 后匹配（`name` 列存小写）

### 注释与文档

- 导出标识符加注释且**以标识符名开头**；项目内注释与提示文案以**中文**为主（与现有代码一致）
- 模块职责用文件顶部包注释或文件注释说明（如 `pkg/lib/lib.go` 顶部"对项目管理的封装"）
- 复杂逻辑（如 Options→SWS 配置渲染、builder 参数回查）写清"为什么"，不逐行翻译代码
- 敏感数据（secret、私有仓库密码）禁止写入注释与日志

### 错误处理

- 不用 panic 表达业务错误；函数返回 `error`，cmd 层统一 `fmt.Println(err)` + `os.Exit(code)`
- 退出码约定：127（配置/项目不可用）、128（已存在）、129（参数非法）、130（操作失败），
  新命令保持相似语义即可（个别处 1）
- API 层返回 echo `error`，由 `customHTTPErrorHandler` 统一转
  `{"success":false,"message":"..."}`；空 secret 视为免鉴权通道，勿改变
- 对外部命令/网络调用（git、static-web-server、GitHub API）须检查 error 与 exit code

### 配置读取

- 一律通过 `conf.New(path)` 后使用 `GetKey/SecHash/GetPath/BaseDir/MustPath` 等方法，
  禁止直接读 ini 文件；新增系统配置分区/键时同步维护 `assets/rtfd.cfg`
  与 `main_test.go` 断言
- SWS 配置模板（`pkg/lib/sws.go`）的缓存开关来自 `[sws]` 分区的
  `static_expires / html_nocache`，由 `renderSWS` 读取；
  **改动模板需同步 `pkg/lib/sws_test.go` 中的表名与缓存指令断言**

### Python 版本约定

- 版本以字符串标识（`lib.PyVer`，如 `3`、`3.10`、`3.12`），**必须在配置 `[py]` 分区中定义**
  （键为版本号，值为对应的 python 程序，要求带 pip、virtualenv）
- `[py]` 分区中 `default` 为新建项目默认版本（缺省取第一个可用版本）、`index` 为 pip 源；
  不符合版本号格式的键（如历史的 `py3`）不作为可用版本，`PyCommand` 对默认版本兼容回退读取 `py3`
- **已移除 Python 2**：历史数据中的 `2`（数字或字符串）在反序列化时归一为默认版本，
  新增代码不得再引用 python2
- 构建时由 `pkg/build` 解析版本对应的解释器并以 `-p` 传入 builder.sh；
  builder.sh 独立运行时按项目 Version 从 `rtfd cfg py <version>` 反查，缺省回落到 `py.default`
- 读取版本列表/解释器/默认版本统一走 `conf` 的 `PyVersions/PyCommand/DefaultPyVersion`，
  不要在业务代码里拼 key
- 官方镜像基于 `ubuntu:24.04` + deadsnakes PPA 固定预装 `3.10` 与 `3.12`（系统自带 3.12，deadsnakes 补 3.10），
  版本列表写死在 `assets/rtfd.cfg` 的 `[py]` 分区与 Dockerfile 的 apt 安装中（两处须同步）；
  自定义镜像时须保证登记的每个版本都能 `pythonX.Y -m virtualenv`

## CLI / API 扩展约定

- 新子命令在 `cmd/` 新建文件，`init()` 中 `rootCmd.AddCommand(...)` 注册；
  `project` 子命令注册到 `projectCmd` 并视情况加短别名（c/g/l/r/u/t）
- `cmd` 层只做参数解析与打印，业务下沉 `pkg/lib` 或 `pkg/build`
- API 新路由注册到 `api/api.go` 的 `registerRoutes`（`/rtfd` Group，`api.New` 同时供启动与测试使用），
  **兼容两种路径风格**（`/:name/xxx` 与 `/xxx/:name` 各注册一条）；响应体保持
  `{"success":bool,"message":string,"data":any}` 风格
- 管理类接口（项目增删改查、配置查询、导入导出）必须调用 `checkAPISecret`（全局，
  配置 `[api] secret`）或 `checkProjectSecret`（全局或项目密钥），未配置 `[api] secret` 时拒绝；
  鉴权为 HMAC-SHA256 动态签名（`verifyAPISign`：X-Rtfd-Ts/X-Rtfd-Nonce/X-Rtfd-Sign，含时间戳窗口与 nonce 防重放）；
  项目相关入参用 `getFormParams` 解析（表单/query/JSON 通吃），值统一经 `util.ParamString/ParamBool` 转换
- 接口文档由代码注解生成（swaggo）：新增/修改接口需在 handler 上方补
  `@Summary/@Tags/@Param/@Success/@Failure/@Security/@Router` 注解，业务错误统一写
  `@Failure default {object} res`（同一状态码只允许一个响应），`data` 用 `resd{data=<类型>}` 展开，
  然后执行 `make docs` 并提交 `docs/` 生成物（CI 会校验是否最新）；
  `docs/` 下的文件为生成物，不要手工修改（也无需补 License 头）
- 触发构建统一经 `pkg/build.Builder`，Sender 用 `vars` 中 `CLISender/APISender/WebhookSender`
- 构建输出识别以 `"Build Successfully"` 前缀行为准（解析第 3 段为耗时），改动 builder.sh 时勿破坏该约定

## 构建脚本与 assets

- `assets/builder.sh` 是唯一真实构建逻辑（git clone → venv → pip → sphinx-build）；
  Go 侧仅落盘脚本并逐行消费输出
- 构建期参数化：builder.sh 通过调用 `rtfd project get <name>:<Key>` / `rtfd cfg` 反查配置，
  覆盖优先级：仓库 `.rtfd.ini` > 项目 Options > 系统 rtfd.cfg
- 对 `.rtfd.ini` 的读取/回写白名单固定为 `project.latest / sphinx.{sourcedir,lang,builder} /
  python.{version,requirement,install,index}`，回写经 `rtfd project update -f`
- 涉及路径/命令的字段（latest/sourcedir/requirement）新增校验时，
  保持防 `/`、`..` 前缀穿越与白名单策略

## 前端（rtfd.js）

- `assets/rtfd.js`：注入生成页的右下角浮动挂件，**零依赖**纯原生 JS（IIFE + 原生 DOM/`fetch`），
  禁止再引入 jQuery / Tipped 之类第三方库与 CDN 资源；经构建 conf.py 追加
  `html_js_files` 引入，query 携带 `name/branch/rtfd_api`
- 数据源为 `GET {rtfd_api}/rtfd/{name}/desc`；新增/修改返回字段需同步 rtfd.js 消费逻辑
- 交互：默认只显示当前语言与版本，点击展开语言/版本/仓库链接；多版本从 URL `/{lang}/{branch}/`
  推导当前状态，单版本固定 `latest`，`showNav=false` 时整体不渲染
- 样式全部以 `#rtfd-widget` 作用域前缀注入，避免污染文档主题自带样式
- 遵循现有 JS 风格：IIFE 包裹、`const/let`（**禁用 `var`**）、`===`、文本一律经
  `textContent` 写入（防 XSS，禁止 innerHTML 拼接外部数据）

## 测试与持续集成

- `make test` = `go test -count=1 ./...`；测试为纯单元/本地（不依赖网络与外部数据库，
  存储相关测试使用 sqlite 临时库）
- 测试文件置于被测包内 `*_test.go`（`pkg/conf`、`pkg/lib`、`pkg/util`、根 `main_test.go`）
- CI（`.github/workflows/gotest.yml`）跑 push/PR 的 `go test`，无需外部服务（存储测试使用 sqlite）
- 修改默认配置模板（`assets/rtfd.cfg`）必须同步 `main_test.go`，否则 CI 失败

## 变更与发布

- 版本号存 `assets/VERSION`（勿在代码硬编码）；发布流程 `make release`（linux amd64/arm64 tar.gz）
- git 提交信息沿用现有简洁英文风格（参考 git log），tag `v*` 推送触发 GoReleaser
  与 Docker 镜像发布（master→`latest`、dev→`dev`）
- 破坏性变更（数据表结构、Options 字段、rtfd.cfg 必需项、builder.sh 输出约定）
  需在提交信息与版本号中体现
