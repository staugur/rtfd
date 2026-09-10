# rtfd 架构文档

> 基于 v1.5.0（master 分支）源码分析整理。

## 1. 项目定位

rtfd 是一个**自托管的 Sphinx 文档构建与托管服务**（类 Read the Docs 精简实现）。
用户提供一个 git 仓库（GitHub/Gitee）地址，rtfd 拉取源码、按语言/分支（版本）用
Sphinx 构建 HTML 文档，交由 Nginx 静态托管，并提供 API / Webhook / 状态徽章等能力。

| 组成 | 技术 | 职责 |
|---|---|---|
| CLI | Go + cobra | 项目 CRUD、触发构建、查看配置、启动 API 服务 |
| API 服务 | Go + echo v4 | desc / badge / build / webhook / GitHub App 回调 |
| 构建器 | bash 脚本 `assets/builder.sh` | git clone → virtualenv → pip → sphinx-build |
| 元数据存储 | Redis（`gtc/redigo`） | 项目配置、自定义域名、构建结果 |
| 文档托管 | Nginx | 静态托管 `{base_dir}/docs/` 下生成的 HTML，并负责域名/SSL |
| 前端挂件 | `assets/rtfd.js`（jQuery+Tipped） | 注入生成文档，提供语言/版本/编辑链接浮动导航 |

**运行环境**：仅支持 Linux；构建期依赖 bash、git、python3（含 pip、virtualenv，支持配置多版本）、
nginx、外部 Redis 服务；GitHub App 功能需能访问 GitHub API。**不再支持 Python 2**。

### 技术栈

- 语言/工程：Go 1.26，module `pkg/tcw.im/rtfd`，单二进制
- CLI：`spf13/cobra v1.1`；Web：`labstack/echo/v4`
- 存储：`gomodule/redigo` + `pkg.tcw.im/gtc/redigo`（Redis 客户端封装，统一前缀）
- 配置：`gopkg.in/ini.v1`（INI 插值支持 `%(key)s`）
- 静态资源打包：`go:embed`（builder.sh / rtfd.js / rtfd.cfg / VERSION）
- 其它：`pkg.tcw.im/gtc`（通用工具库）、`golang-jwt/jwt/v5`（GitHub App 身份）

---

## 2. 整体架构

```
┌──────────── 使用方 ────────────┐       ┌────── Git 仓库 (GitHub / Gitee) ──────┐
│ 用户 / CI / Webhook 事件        │       │  用户源码 (docs/ + conf.py + .rtfd.ini)│
└──────────────┬─────────────────┘       └──────────────▲───────────────────────┘
               │ ① rtfd 命令 / HTTP API                    │ ④ git clone / webhook 自动构建
               ▼                                          │
┌──────────────────────────────────────────────────────────────────────────────┐
│                          rtfd（单一二进制 main → cmd）                        │
│                                                                              │
│  cmd/ (cobra)                    api/ (echo，前缀 /rtfd)                       │
│  ├── rtfd project create/get/…   ├── GET  /rtfd/:name/desc  …/desc/:name     │
│  ├── rtfd build <name>           ├── GET  /rtfd/:name/badge  …/badge/:name   │
│  ├── rtfd cfg                    ├── POST /rtfd/:name/build   (X-Rtfd-Sign)  │
│  └── rtfd api                    ├── POST /rtfd/:name/webhook                │
│        │                         ├── POST /rtfd/github/app   (GitHub App)    │
│        │                         └── GET  /rtfd/assets/rtfd.js               │
│        ▼                                                                     │
│  pkg/build.Builder ──执行──▶ assets/builder.sh (落盘为 {base}/.rtfd-builder.sh)│
│    git clone(branch) → virtualenv → pip install → conf.py 注入 →             │
│    sphinx-build ×(lang × branch) → latest 软链 → .rtfd.ini 回写              │
└───────┬──────────────────────────────────────────────────────────────────────┘
        │ 读写（Pipeline 批量事务）                │ 生成/重载配置 (nginx -t / reload)
        ▼                                        ▼
┌──────────────────────┐      ┌────────────────────────────────────────────────┐
│ Redis  (前缀 rtfd:)   │      │ Nginx                                        │
│  projects   (set)     │      │ base/nginx/{name}.conf     默认域名 {n}.{dn}  │
│  domains    (set)     │      │ base/nginx/ext/{name}.conf 自定义域名(可选SSL)│
│  project:{n} (string) │      │ root: base/docs/{name}/{lang}/{version}       │
│  builder:{n} (hash)   │      └────────────────────────────────────────────────┘
└──────────────────────┘
```

核心设计点：

- **构建与托管分离**：Go 只负责状态编排与参数化，真正的构建在 bash 脚本内完成；
  `builder.sh` 在构建期反查 `rtfd` CLI（`rtfd project get <name>:<key>`）获取配置，
  形成「Go → bash → Go」的回环协作。
- **项目配置收敛于 Redis JSON**，本地文件系统只放构建产物与 nginx conf，
  便于迁移（`rtfd project transfer` 可 base64 导入/导出）。
- **webhook 自动构建**：GitHub/Gitee 推送事件打到 API，校验签名后异步触发构建。

---

## 3. 数据模型

### 3.1 Redis Key（客户端统一前缀 `rtfd:`）

| Key | 类型 | 说明 |
|---|---|---|
| `projects` | set | 全部文档项目名（小写） |
| `domains` | set | 全部已占用自定义域名 |
| `project:{name}` | string | 项目配置，内容为 Options 的 JSON |
| `builder:{name}` | hash | 构建结果，field=分支/标签，value=Result 的 JSON |

Key 构造统一走 `pkg/lib` 中 `GBPK / GBDK / BCK(name) / BRK(name)` 常量与函数，
项目名一律先 `strings.ToLower`。

### 3.2 Options —— 项目配置（`pkg/lib/lib.go`）

| 字段 | JSON 语义 | 说明 |
|---|---|---|
| Name | 项目名 | 唯一标识，小写，正则 `^[a-zA-Z][0-9a-zA-Z_\-]{1,100}$` |
| URL | git 地址 | 仅 http(s)，私有仓在协议后携带编码的 `username:password` |
| Latest | 分支 | latest 软链指向的分支（系统默认 `default_branch`） |
| Version | Python 版本 | 字符串，如 3、3.10、3.12，需在配置 `[py]` 分区中定义 |
| Single | 是否单版本 | true 时 nginx 用单版本模板 |
| SourceDir | 文档目录 | 相对仓库根，如 `docs`、`.` |
| Lang | 语言列表 | 逗号分隔，如 `en,zh_CN` |
| Requirement | 依赖文件 | 逗号分隔多个，相对仓库根 |
| Install | 是否 `pip install .` | bool |
| Index | pip 源 | 默认官方源 |
| ShowNav / HideGit | 导航开关 | 决定 rtfd.js 是否展示及 git 链接 |
| Secret | 密钥 | build/webhook 校验用 |
| DefaultDomain | 默认域名 | `{name}.{nginx.dn}`（只读，自动生成） |
| CustomDomain / SSL / SSLPublic / SSLPrivate | 自定义域名 | 支持 HTTPS |
| Builder | 构建器 | `html` / `dirhtml` / `singlehtml` |
| GSP / IsPublic | git 服务商 | `GitHub` / `Gitee` / `N/A` 及公私有 |
| BeforeHook / AfterHook | 钩子命令 | 构建前后执行 |
| Meta | 扩展 KV | map，以下划线开头为系统保留 |

**Python 版本**：`Version` 为字符串（`lib.PyVer`），取值必须是 `[py]` 分区中已定义的版本号；
旧数据以数字存储（2 或 3），反序列化时自动归一为 `3`（Python 2 已移除），
保证存量项目平滑升级。

**Meta 系统保留字段**（定义于 `vars`）：`_webhook_id`、`_installation_id`、
`_update_file_md5`（`.rtfd.ini` 文件 MD5，未变更时跳过回写）。
用户常用 meta：`excluded_branch`（webhook 排除的分支，`_sep` 指定分隔符，默认 `|`）。
Meta key 需匹配 `^[a-z_][0-9a-z_]{1,63}$`，取值 `-`（`vars.ResetEmpty`）表示清空。

### 3.3 Result —— 构建结果（hash 的 value）

| 字段 | 说明 |
|---|---|
| Branch | 构建的分支或标签 |
| Status | true=passing，其它为失败 |
| Sender | 发起来源：`cli` / `api` / `webhook` |
| Btime | 结束时间 `YYYY-MM-DD HH:MM:SS` |
| Usedtime | 耗时秒数（解析脚本 `Build Successfully` 行） |

### 3.4 .rtfd.ini —— 仓库内构建规则（可选）

文档仓库根放 `.rtfd.ini`（样例见 `assets/rtfd.ini`），构建时**覆盖**系统存储的
构建参数，仅限以下分区/字段：

- `[project] latest`；`[sphinx] sourcedir / lang / builder`
- `[python] version / requirement / install / index`（`version` 取值须在 `[py]` 分区中已定义）

---

## 4. 项目生命周期（`pkg/lib`）

`ProjectManager` 是核心对象：持有配置路径 + `conf.Config` + Redis `db.DB`。

### 4.1 创建 create

```
cli create → GenerateOption(name, url) 生成默认 Options
  ├─ 校验 name / git URL（仅 github.com、gitee.com，判定 public/private）
  ├─ 识别 GSP，去 .git 后缀，拼默认域名 {name}.{nginx.dn}
  → SetOption 逐项覆盖 CLI 入参（reflect 赋值）
  → Create:
      ├─ 排除保留名 www 与 default.unallowed_name
      ├─ 校验必填项；校验 python 版本已在 [py] 分区定义；自定义域名校验且须未占用；SSL 证书文件须存在
      ├─ renderNginx：渲染 {base}/nginx/{name}.conf（默认域）与 ext 目录（自定义域）
      │    → nginx -t 通过后 -s reload
      └─ Redis Pipeline 批量事务：SAdd projects / Set project:{name}=JSON / SAdd domains
         （GitHub 项目再异步创建仓库 webhook，失败仅告警不阻断）
```

### 4.2 查询

- `GetSourceName`：原始 JSON；`GetName`：反序列化为 Options
- `GetNameOption(name, key)`：反射读单字段；`Meta@key` 语法读 meta；`OptionKeyMap` 兼容字段名大小写
- `ListBuildset / GetBuildset / GetNameWithBuildset`：构建集与配置合并输出

### 4.3 更新 update（两种模式）

1. `-t "Field:Value,..."`：`updateHook.handle()` 按字段分发到处理函数，逐字段
   校验/落 Options，设置 `render` 标志；收集 ok/fail 列表逐个打印；
   完成后整体 JSON 写回 Redis，`render=true` 字段（lang/single/domain/ssl…）重渲染 nginx。
2. `-f .rtfd.ini`：构建期自动回写，白名单抽取 `latest` 等字段，比对 meta
   `_update_file_md5` 未变化则跳过（输出 `not updated`）。

安全细节：`latest`、`sourcedir`、`requirement` 禁止以 `/`、`..` 开头（防目录穿越）。

### 4.4 删除 remove

按 name 清理：`docs/{name}` 目录、默认/扩展 nginx conf（删除后 reload）、
GitHub webhook（若 GSP=GitHub）、最后 Redis Pipeline `SRem/Del` 四键原子完成。

### 4.5 转储 transfer

导出：项目 Options 序列化 JSON → base64 输出（默认剔除 `_` 开头系统 meta，
`--export-sys-meta` 可含）。导入：base64 → 反序列化 → 可选别名覆盖 → 重建默认域名
→ 走 `Create`，即「复制项目」。

---

## 5. 文档构建流程（核心）

### 5.1 触发入口与形态

| 入口 | Sender | 说明 |
|---|---|---|
| `rtfd build <name>` | cli | 可 `--branch`，`--debug`=bash -x，`--log`=行日志 |
| `POST /rtfd/:name/build` | api | 异步 `go BuildWithLog` |
| `POST /rtfd/:name/webhook` | webhook | GitHub push/release、Gitee push，排除分支不构建 |

`pkg/build.Builder.build()` 流程：

```
校验项目存在 → 取 Options → branch 缺省=Latest
→ 把 assets.BuiderSH 落盘 {base_dir}/.rtfd-builder.sh
→ 执行: bash [-x] .rtfd-builder.sh -n <name> -b <branch> -c <cfg>
→ RunCmdStream 逐行回调:
    ├─ cli 直接打印，api/webhook 走 log 或忽略
    └─ 匹配 "Build Successfully, N seconds passed." → status=true, usedtime=N
→ BuildRecord：Result JSON 写入 builder:{name} hash(branch)
```

### 5.2 builder.sh 流水线

```
main
 ├─ 校验参数/配置 → base_dir 必须以 / 开头且长度≥2
 ├─ 建 {base}/docs、{base}/runtimes，mktemp 出临时运行目录
 ├─ _codeManager: rm -rf 旧克隆
 │    git clone --branch <branch> --single-branch --depth=1 --recursive <URL> <name>
 └─ _envManager:
      ├─ 读取仓库根 .rtfd.ini（存在则其值为最高优先级，缺省回落项目 Options/系统配置）
      ├─ 按项目 Version 解析 python 程序（Go 侧经 -p 传入；脚本独立运行时按版本号从
      │    `rtfd cfg py <version>` 反查，缺省回落到 py.default）建虚拟环境 venv-<ver>
      │    → source activate
      ├─ 逐 requirements 文件: <venv_py> -m pip install -i <index> -r <req>
      ├─ install=true 时额外 <venv_py> -m pip install .
      ├─ conf.py 追加注入（自动生成标记 + 仅当未定义）:
      │    html_js_files += "<server_static_url>rtfd.js?v=<ver>&name=..&branch=..&rtfd_api=.."
      │    html_favicon = favicon_url
      ├─ 执行 before_hook
      ├─ 对每种 lang: sphinx-build -E -T -D language=<lang> -b <builder>
      │        <SourceDir>  {docs}/{name}/{lang}/{branch}
      │    并 ln -nsf ../.. 使 {lang}/latest → {lang}/{Latest}
      ├─ 执行 after_hook（成败仅 debug 输出，不阻断）
      ├─ deactivate
      └─ 若存在 .rtfd.ini → rtfd project update -f .rtfd.ini <name>（回写 DB）
主流程结束打印 "Build Successfully, N seconds passed." 并删除临时目录
trap SIGINT/SIGTERM 兜底清理退出
```

### 5.3 参数优先级

`.rtfd.ini（仓库内）` > `项目 Options（Redis）` > `系统 rtfd.cfg 默认值`。
builder.sh 通过 `_getDocsConf`（`rtfd project get`）与 `_getRtfdConf`（`rtfd cfg`）反查配置。

---

## 6. 托管与 URL 布局（Nginx）

生成的文档目录布局（`{base_dir}/docs/{name}/`）：

```
docs/{name}/
└── {lang}/
    ├── latest/          → 符号链接，指向当前 Latest 分支目录
    ├── master/          ← 每次构建按 branch 全量覆盖
    └── v1.0/ …          ← tag/release 构建的版本目录
```

每个项目渲染两套 server 配置（text/template，`pkg/lib/nginx.go`）：

| 配置 | 位置 | 域名 | SSL |
|---|---|---|---|
| 默认 | `{nginx.conf_dir}/{name}.conf` | `{name}.{nginx.dn}` | 系统级 `ssl_crt/ssl_key` |
| 自定义 | `{nginx.conf_ext_dir}/{name}.conf` | 项目 CustomDomain | 项目级 SSLPublic/SSLPrivate |

模板特性：

- **多版本**（Single=false）：`root {docs}/{name}/`，`set $home /{lang}/latest;`
  对未带版本前缀的 URL，若在 latest 路径下存在则 `302 → $home$document_uri`。
- **单版本**（Single=true）：`root {docs}/{name}/{lang}/latest/`。
- **SSL**：`listen 443 ssl http2` + http→https 301 + 完整 TLS 配置（TLS1.0~1.3、HSTS）。
- 渲染后统一执行 `nginx -t && nginx -s reload`（是否 sudo 由 `nginx.sudo` 控制）。

### 6.1 性能与缓存

文档站的特征是「海量静态小文件 + 内容随重建变化 URL 不变」，因此采用**分级缓存**：

| 资源 | 策略 | 原因 |
|---|---|---|
| `_static/`、`_images/`、css/js/图片/字体 | `expires <static_expires>s` + `Cache-Control: public`，并关闭 access_log | 内容稳定，可强缓存 |
| HTML | `Cache-Control: no-cache`（仍可 304 协商，依赖 ETag/Last-Modified） | 重建后同 URL 内容变化，不能复用过期副本 |
| 所有文件 | `open_file_cache`（缓存 fd 与 stat 结果，`valid 30s`） | 降低大量小文件的 open/stat 系统调用 |

**主配置（`scripts/nginx.conf`，http 级，Docker 镜像使用）** 已包含：
`sendfile + tcp_nopush/nodelay`、`keepalive_requests 1000`、`gzip`（含 `gzip_types/js/css/svg`、`gzip_min_length`、
`gzip_static`）、`open_file_cache`、`etag on`、`if_modified_since before`、access_log 缓冲
（`buffer=64k flush=5s`）、多项目域名所需的 `server_names_hash_*`，并 `include /rtfd/nginx/*.conf`
（否则生成的配置不会被加载）。

**项目模板（`pkg/lib/nginx.go`）** 按上述分级缓存渲染，开关来自 `[nginx]` 分区：

| 配置键 | 默认 | 说明 |
|---|---|---|
| `static_expires` | 3600 | 静态资源缓存秒数，0 表示不下发缓存头 |
| `html_nocache` | on | HTML 使用 `no-cache` 协商缓存 |
| `open_file_cache` | on | 是否启用文件元数据缓存 |

注意：

- 修改缓存配置后，已存在项目需触发一次重渲染才会生效（如 `rtfd p u -t single:<原值> <name>`）
- `open_file_cache` 生效期间，新建/变更的文件最长 30s 才可见（构建新版本后立即访问可能短暂 404）
- 静态资源强缓存期内若重建了同名静态文件，浏览器可能仍用旧副本（Sphinx 资源名不带 hash，
  建议取值不宜过长，或用版本号目录隔离）

访问页面的语言/版本切换由注入页面的 `rtfd.js` 挂件完成：构建时 URL query 携带
`name/branch/rtfd_api`，运行时向 `GET /rtfd/:name/desc` 拉取元数据渲染浮动面板。

---

## 7. 对外 API（`api` 包，均挂 `/rtfd` 前缀）

兼容两种路径风格（`:name` 在前后），新增接口建议同时注册两种。

| 路由 | 方法 | 功能 | 鉴权/说明 |
|---|---|---|---|
| `/:name/desc`、`/desc/:name` | GET | 项目描述 | 返回 URL、langs、latest、versions、builder、showNav 等，供 rtfd.js 渲染 |
| `/:name/badge`、`/badge/:name` | GET | 文档状态徽章 SVG | `?branch=`，默认 Latest；passing/failing/unknown |
| `/:name/build`、`/build/:name` | POST | 触发构建 | Header `X-Rtfd-Sign=MD5(secret)`（空 secret 免鉴权）；参数 branch；异步执行返回 201 |
| `/:name/webhook`、`/webhook/:name` | POST | git webhook | 校验 GitHub `X-Hub-Signature`(sha1=HMAC) / Gitee `X-Gitee-Token`；识别 UA 分派；ping→pong；排除 `excluded_branch` |
| `/assets/rtfd.js` | GET/HEAD | 静态挂件脚本 | 内嵌 assets.RtfdJS |
| `/github/app` | POST | GitHub App 事件 | `installation`/`installation_repositories`，校验 App ID 后 `Dispatch` |

统一响应：`{"success": bool, "message": string}`；错误由 `customHTTPErrorHandler`
输出（默认 HTTP 200，echo.HTTPError 除外）。

---

## 8. CLI 命令一览

```
rtfd  [-c/--config 文件]  [-v 版本]  [-i 构建信息]  [--init 生成默认配置]
├── api                 [--host] [--port]             启动 API 服务
├── cfg  [section] [key] [-j/--json]                  查看配置
├── build <name>        [-b/--branch] [--debug] [--log]  构建文档
└── project (p)
    ├── create (c) <name> -u/--url …                  创建项目
    ├── get (g) <name[:field|:Meta@k]> [-b/--build]   查询项目/字段/构建集
    ├── list (l)         [-v/--verbose]               列出项目
    ├── remove (r) <name>                             删除项目
    ├── transfer (t)     -e <name> | -i <base64> [新名] 导入导出
    └── update (u) <name> [-s sep] (-t Field:Value,… | -f .rtfd.ini)  更新配置
```

根命令 `initConfig`：除 `-h/-v/-i/--init` 外，所有子命令要求配置文件存在，否则打印提示并 `os.Exit(127)`。
错误处理约定：cmd 层 `fmt.Println(err)` + `os.Exit`，退出码大致分为
127（配置/项目不可用）、128（项目/域名已存在）、129（参数非法）、130（操作失败），
含义有一定重叠，新命令保持一致风格即可。

---

## 9. GitHub App 集成（`pkg/lib/app.go`）

目的：创建 GitHub 项目时**自动注册仓库 webhook**，删除时自动清理，无需用户手动配置。

- 配置段 `[ghapp]`：`enable/app_id/private_key`（PEM）+ `api.server_url`。
- 身份：私钥签 RS256 JWT（`iss=app_id`，10 分钟），JWT 换 installation access token（缓存 1 小时）。
- 事件流：
  - `/github/app` 收到 `installation.created/deleted`、`installation_repositories.added/removed`
    → `Dispatch` 对比全部 GitHub 项目 URL，更新 meta `_installation_id/_webhook_id`，
    并增删仓库级 webhook（URL 为 `{server_url}/rtfd/webhook/{name}`，push+release 事件）。
  - CLI 侧 `Create`/`Remove` 同步调 `cliSetWebhook/cliRemoveWebhook`。

---

## 10. 配置（`rtfd.cfg`，默认 `~/.rtfd.cfg` 或 `$RTFD_CFG`）

| 分区 | 关键项 | 说明 |
|---|---|---|
| （default） | base_dir* / redis* / default_branch / unallowed_name / favicon_url / log_level | 数据根目录（初始化后勿改，否则丢数据）；Redis 连接串 `redis://[user:pwd@]host:port/db` |
| [nginx] | dn* / exec / sudo / ssl_crt / ssl_key / conf_dir / conf_ext_dir | 托管域名后缀；nginx 路径与是否 sudo；SSL 与配置目录 |
| [nginx] | static_expires / html_nocache / open_file_cache | 静态资源缓存秒数（默认3600）、HTML 协商缓存（默认on）、文件元数据缓存（默认on），详见 6.1 |
| [py] | `<版本号>`* / default / index | 可用 Python 版本映射（至少一项，键为版本号如 3、3.10、3.12，值为程序路径，要求带 pip+virtualenv）、默认版本（缺省取第一个可用版本）、pip 源 |
| [api] | host / port / server_url* / server_static_url | 监听与对外服务地址（webhook 回跳用） |
| [ghapp] | enable / app_id / private_key | GitHub Apps 开关与凭据 |

> * = 必需。conf_dir 默认 `%(base_dir)s/nginx`，conf_ext_dir 默认 `%(conf_dir)s/ext`。
> 配置默认值同步维护于 `assets/rtfd.cfg`（`rtfd --init` 写入），变更需同步 `main_test.go` 断言。

---

## 11. 目录结构与包职责

```
rtfd/
├── main.go           程序入口，仅空白导入 assets（embed 生效）+ 执行 cmd.Execute
├── main_test.go      默认配置结构断言（base_dir/nginx/py/api 分区完整性）
├── api/              API 层：api.go(路由注册/Start) · view.go(处理器) · tool.go(公共函数)
├── assets/           静态资源(go:embed)：rtfd.cfg · builder.sh · rtfd.js · VERSION · rtfd.ini(样例)
├── cmd/              cobra 命令层：root/api/cfg/build/project(+create/get/list/remove/transfer/update)
├── pkg/
│   ├── build/        构建编排：Builder + builder.sh 落盘与输出流解析
│   ├── conf/         ini 配置封装（SecHash/GetKey/BaseDir/DefaultBranch…）
│   ├── lib/          ★ 核心业务：
│   │   ├── lib.go     ProjectManager 项目 CRUD、Redis key、nginx 渲染调度
│   │   ├── nginx.go   nginx server 配置模板渲染（multi/single/ssl）
│   │   ├── update.go  更新字段分派（updateHook）与安全校验
│   │   └── app.go     GitHub App（JWT/installation token/webhook 同步）
│   └── util/         纯工具：命令执行、正则校验、git URL/域名解析、HMAC-SHA1
├── vars/             跨包常量（Sender、Redis Key 段、GitHub/Gitee 常量、ResetEmpty…）
├── scripts/          部署：nginx.conf / supervisord.conf / rtfd.service / start.sh
├── Makefile          构建/测试/发布目标
├── Dockerfile        两阶段构建：golang:1.26-alpine 编译 → python:3.11-slim 运行镜像(+ nginx + supervisor)，
│                     并用 uv 预装 `python_versions` 指定的额外 Python 版本后登记到 /rtfd.cfg 的 [py] 分区
└── .github/workflows/ gotest.yml(测试) · publish.yml(镜像 master→latest、dev→dev、release) · goreleaser.yml(tag→多平台二进制)
```

代码风格要点：每个 `.go` 文件带 Apache-2.0 License 头；导入分组
标准库 / 项目内部（`pkg/tcw.im/rtfd/*`）/ 外部依赖（`github.com`、`pkg.tcw.im`）；
注释与用户提示以中文为主；全局常量集中在 `vars`。

---

## 12. 构建、发布与 CI

- 本地：`make dev`（go run ./ api）、`make test`（`go test -count=1 ./...`）、
  `make build-arm/amd`（linux 交叉编译，`-ldflags -X cmd.commitID/-X cmd.built`）、
  `make release`（编译并打 tar.gz）。
- 版本：版本号存 `assets/VERSION`（当前 1.5.0），`-v` 输出；
  tag `v*` 推送触发 GoReleaser（linux amd64/386/arm64 + checksum）。
- 镜像：`publish.yml` 在 master→`latest`、dev→`dev`、release published 时构建，
  运行时镜像内含 nginx + python3.11 + supervisor（supervisord 拉起 `rtfd api` 与 nginx）。
- 镜像内多版本 Python：基础镜像 `python:3.11-slim` 提供版本号 `3`（`python3`），
  构建时用 uv 预装 `ARG python_versions`（默认 `3.10 3.12`）的预编译 CPython，
  为每个版本装上 virtualenv 并以 `版本号 = /usr/local/bin/pythonX.Y` 写入 `/rtfd.cfg` 的 `[py]` 分区，
  同时固定 `default = 3`；不需要多版本时可 `--build-arg python_versions=""`。
- 测试：纯单元测试（conf/util/nginx 渲染/main 默认配置），不依赖网络与 Redis 数据。

---

## 13. 安全要点

| 风险面 | 防护 |
|---|---|
| 私有仓库 | URL 内嵌凭据仅 http(s) 的 github.com/gitee.com；对外接口经 `PublicGitURL` 剥敏 |
| 构建/Webhook 伪造 | build API：`X-Rtfd-Sign=MD5(secret)`；GitHub：`X-Hub-Signature` HMAC-SHA1；Gitee：`X-Gitee-Token` 比对；`ping` 仅回 pong |
| 路径穿越 | `latest/sourcedir/requirement` 禁以 `/`、`..` 开头；meta key 白名单正则 |
| 名称/域名注入 | name 正则、域名 `IsDomain` 校验、自定义域名全局唯一（domains set） |
| GitHub App | 事件类型/目标类型白名单、installation App ID 与 header 强校验；webhook 仅 push/release |
| 保留名 | `www` 及 `unallowed_name` 列表内名称禁止创建 |
| 存储敏感信息 | secret 入 Options JSON（Redis），日志不输出明文；导出时默认剔除系统 meta |
