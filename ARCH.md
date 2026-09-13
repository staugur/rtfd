# rtfd 架构文档

> 基于 v1.5.0（master 分支）源码分析整理。

## 1. 项目定位

rtfd 是一个**自托管的 Sphinx 文档构建与托管服务**（类 Read the Docs 精简实现）。
用户提供一个 git 仓库（GitHub/Gitee）地址，rtfd 拉取源码、按语言/分支（版本）用
Sphinx 构建 HTML 文档，交由 Caddy 静态托管，并提供 API / Webhook / 状态徽章等能力。

| 组成 | 技术 | 职责 |
|---|---|---|
| CLI | Go + cobra | 项目 CRUD、触发构建、查看配置、启动 API 服务 |
| API 服务 | Go + echo v4 | desc / badge / build / webhook / GitHub App 回调 |
| 构建器 | bash 脚本 `assets/builder.sh` | git clone → virtualenv → pip → sphinx-build |
| 元数据存储 | 关系型数据库 + GORM（sqlite / mysql / pgsql） | 项目配置、自定义域名、构建结果 |
| 文档托管 | Caddy | 单实例多站点块静态托管 `{base_dir}/docs/` 下生成的 HTML，自动申请域名证书（HTTPS） |
| 前端挂件 | `assets/rtfd.js`（原生 JS，零依赖） | 注入生成文档，右下角浮动面板提供语言/版本/编辑链接切换 |

**运行环境**：仅支持 Linux；构建期依赖 bash、git、python3（含 pip、virtualenv，支持配置多版本）、
Caddy；元数据存储使用 sqlite / mysql / pgsql（默认 sqlite，无需外部服务）；
GitHub App 功能需能访问 GitHub API。**不再支持 Python 2**。

### 技术栈

- 语言/工程：Go 1.26，module `pkg.tcw.im/rtfd/v2`，单二进制
- CLI：`spf13/cobra v1.10`；Web：`labstack/echo/v4`
- 存储：`gorm.io/gorm` + 驱动 `glebarez/sqlite`（纯Go，保持 `CGO_ENABLED=0` 交叉编译）、
  `gorm.io/driver/mysql`、`gorm.io/driver/postgres`
- 配置：`gopkg.in/ini.v1`（INI 插值支持 `%(key)s`）
- 静态资源打包：`go:embed`（builder.sh / rtfd.js / rtfd.cfg / VERSION / swagger.html）
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
        │ 读写（GORM）                              │ 生成/重载配置 (renderCaddy + reload)
        ▼                                        ▼
┌────────────────────────────┐  ┌─────────────────────────────────────────────┐
│ 数据库 (sqlite/mysql/pgsql) │  │ Caddy                                       │
│  projects      项目配置     │  │ base/caddy/Caddyfile       默认域名 {n}.{dn} │
│  build_results 构建结果     │  │ （每域名自动申请证书，站点块独立配置）        │
└────────────────────────────┘  │ root: base/docs/{name}/{lang}/{version}      │
                                └─────────────────────────────────────────────┘
```

核心设计点：

- **构建与托管分离**：Go 只负责状态编排与参数化，真正的构建在 bash 脚本内完成；
  `builder.sh` 在构建期反查 `rtfd` CLI（`rtfd project get <name>:<key>`）获取配置，
  形成「Go → bash → Go」的回环协作。
- **项目配置收敛于数据库**，本地文件系统只放构建产物与 Caddy 配置，
  便于迁移（`rtfd project transfer` 可 base64 导入/导出，也是跨存储迁移的手段）。
- **webhook 自动构建**：GitHub/Gitee 推送事件打到 API，校验签名后异步触发构建。

---

## 3. 数据模型

### 3.1 表结构（`pkg/store`，GORM 自动迁移）

**projects** —— 文档项目配置（`store.Project`，对应 `lib.Options` 的关系化存储）：

| 列 | 类型 | 说明 |
|---|---|---|
| id | 主键 | 自增 |
| name | varchar(100) | 项目名，唯一索引，统一小写 |
| url / latest / version | varchar | git地址、默认分支、python版本（如 3.10） |
| single / install / show_nav / hide_git / ssl / is_public | bool | 各类开关 |
| source_dir / lang / requirement / index | varchar | 文档目录、语言、依赖文件、pip源 |
| secret | varchar | build/webhook 校验密钥 |
| default_domain | varchar | `{name}.{caddy.dn}` |
| custom_domain | varchar(255) | 自定义域名，**普通索引**（唯一性由 `HasCustomDomain` 校验） |
| ssl_public / ssl_private | varchar | 自定义域名证书 |
| builder / gsp | varchar | 构建器、git服务商 |
| meta | text | `map[string]string` 的 JSON（`serializer:json`） |
| created_at / updated_at | 时间 | GORM 自动维护 |

**build_results** —— 构建结果（`store.BuildResult`）：

| 列 | 类型 | 说明 |
|---|---|---|
| id | 主键 | 自增 |
| project + branch | varchar | **联合唯一索引**（同项目同分支只保留最新结果） |
| status | bool | passing / failing |
| sender | varchar | `cli` / `api` / `webhook` |
| btime | varchar | 构建结束时间 `YYYY-MM-DD HH:MM:SS` |
| usedtime | int | 耗时秒数 |
| created_at / updated_at | 时间 | 首次构建 / 最近更新 |

> 旧版本（Redis）的四类 Key：`projects`(set)、`domains`(set)、`project:{name}`(string JSON)、
> `builder:{name}`(hash) 已废弃；迁移方式见「12. 迁移说明」。

### 3.2 Options —— 项目配置（`pkg/lib/lib.go`）

| 字段 | JSON 语义 | 说明 |
|---|---|---|
| Name | 项目名 | 唯一标识，小写，正则 `^[a-zA-Z][0-9a-zA-Z_\-]{1,100}$` |
| URL | git 地址 | 仅 http(s)，私有仓在协议后携带编码的 `username:password` |
| Latest | 分支 | latest 软链指向的分支（系统默认 `default_branch`） |
| Version | Python 版本 | 字符串，如 3、3.10、3.12，需在配置 `[py]` 分区中定义 |
| Single | 是否单版本 | true 时站点根目录指向 {lang}/latest |
| SourceDir | 文档目录 | 相对仓库根，如 `docs`、`.` |
| Lang | 语言列表 | 逗号分隔，如 `en,zh_CN` |
| Requirement | 依赖文件 | 逗号分隔多个，相对仓库根 |
| Install | 是否 `pip install .` | bool |
| Index | pip 源 | 默认官方源 |
| ShowNav / HideGit | 导航开关 | 决定 rtfd.js 是否展示及 git 链接 |
| Secret | 密钥 | build/webhook 校验用 |
| DefaultDomain | 默认域名 | `{name}.{caddy.dn}`（只读，自动生成） |
| CustomDomain / SSL / SSLPublic / SSLPrivate | 自定义域名 | 支持 HTTPS |
| Builder | 构建器 | `html` / `dirhtml` / `singlehtml` |
| GSP / IsPublic | git 服务商 | `GitHub` / `Gitee` / `N/A` 及公私有 |
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

`ProjectManager` 是核心对象：持有配置路径 + `conf.Config` + GORM `*gorm.DB`（由 `pkg/store.Open` 创建，
启动时自动迁移表结构；项目名统一小写）。

### 4.1 创建 create

```
cli create → GenerateOption(name, url) 生成默认 Options
  ├─ 校验 name / git URL（仅 github.com、gitee.com，判定 public/private）
  ├─ 识别 GSP，去 .git 后缀，拼默认域名 {name}.{caddy.dn}
  → SetOption 逐项覆盖 CLI 入参（reflect 赋值）
  → Create:
      ├─ 排除保留名 www 与 default.unallowed_name
      ├─ 校验必填项；校验 python 版本已在 [py] 分区定义；自定义域名校验且须未占用；SSL 证书文件须存在
      ├─ renderCaddy：汇总全部项目渲染 {base}/caddy/Caddyfile（站点块 + latest 跳转 + 缓存，自动 HTTPS）
      │    → 执行 `caddy reload` 热加载（无需重启）
      └─ 写入 projects 表（GORM Create；GitHub 项目再异步创建仓库 webhook，失败仅告警不阻断）
```

### 4.2 查询

- `GetSourceName`：原始 JSON；`GetName`：反序列化为 Options
- `GetNameOption(name, key)`：反射读单字段；`Meta@key` 语法读 meta；`OptionKeyMap` 兼容字段名大小写
- `ListBuildset / GetBuildset / GetNameWithBuildset`：构建集与配置合并输出

### 4.3 更新 update（两种模式）

1. `-t "Field:Value,..."`：`updateHook.handle()` 按字段分发到处理函数，逐字段
   校验/落 Options，设置 `render` 标志；收集 ok/fail 列表逐个打印；
   完成后整体写回 projects 表（`SaveOptions`），`render=true` 字段（lang/single/domain/ssl…）重渲染 Caddy 配置。
2. `-f .rtfd.ini`：构建期自动回写，白名单抽取 `latest` 等字段，比对 meta
   `_update_file_md5` 未变化则跳过（输出 `not updated`）。

安全细节：`latest`、`sourcedir`、`requirement` 禁止以 `/`、`..` 开头（防目录穿越）。

### 4.4 删除 remove

按 name 清理：`docs/{name}` 目录，删除后重新汇总渲染 Caddy 配置并热加载、
GitHub webhook（若 GSP=GitHub），最后在一个事务内删除 projects 记录及其 build_results 记录。

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
      ├─ conf.py 追加注入（自动生成标记 + 组合已有 setup，不覆盖）:
      │    setup(app): app.add_js_file("<server_url>/rtfd/assets/rtfd.js?v=<ver>",
      │        **{'data-name':.., 'data-branch':.., 'data-api':..})
      ├─ 对每种 lang: sphinx-build -E -T -D language=<lang> -b <builder>
      │        <SourceDir>  {docs}/{name}/{lang}/{branch}
      │    并 ln -nsf ../.. 使 {lang}/latest → {lang}/{Latest}
      ├─ deactivate
      └─ 若存在 .rtfd.ini → rtfd project update -f .rtfd.ini <name>（回写 DB）
主流程结束打印 "Build Successfully, N seconds passed." 并删除临时目录
trap SIGINT/SIGTERM 兜底清理退出
```

### 5.3 参数优先级

`.rtfd.ini（仓库内）` > `项目 Options（数据库）` > `系统 rtfd.cfg 默认值`。
builder.sh 通过 `_getDocsConf`（`rtfd project get`）与 `_getRtfdConf`（`rtfd cfg`）反查配置。

---

## 6. 托管与 URL 布局（Caddy）

生成的文档目录布局（`{base_dir}/docs/{name}/`）：

```
docs/{name}/
└── {lang}/
    ├── latest/          → 符号链接，指向当前 Latest 分支目录
    ├── master/          ← 每次构建按 branch 全量覆盖
    └── v1.0/ …          ← tag/release 构建的版本目录
```

汇总全部项目渲染为一份 Caddyfile（text/template，`pkg/lib/caddy.go`）：

| 配置 | 位置 | 域名 | SSL |
|---|---|---|---|
| 默认域名 | `{caddy.conf_dir}/Caddyfile` 内一个站点块 | `{name}.{caddy.dn}` | Caddy 自动申请 |
| 自定义域名 | 同站点块的第二个地址（共用根目录） | 项目 CustomDomain | Caddy 自动申请 |

模板特性：

- **多版本**（Single=false）：`root {docs}/{name}/`，`set $home /{lang}/latest;`
  对未带版本前缀的 URL，若在 latest 路径下存在则 `302 → $home$document_uri`。
- **单版本**（Single=true）：`root {docs}/{name}/{lang}/latest/`。
- **SSL**：`listen 443 ssl http2` + http→https 301 + 完整 TLS 配置（TLS1.0~1.3、HSTS）。
- 渲染后执行 `caddy reload --config <file>`（优雅热加载；是否 sudo 由 `caddy.sudo` 控制）。

### 6.1 性能与缓存

文档站的特征是「海量静态小文件 + 内容随重建变化 URL 不变」，因此采用**分级缓存**：

| 资源 | 策略 | 原因 |
|---|---|---|
| `_static/`、`_images/`、css/js/图片/字体 | `header @static Cache-Control "public, max-age=<static_expires>"` | 内容稳定，可强缓存 |
| HTML | `header @html Cache-Control "no-cache"`（仍可 304 协商，依赖 ETag/Last-Modified） | 重建后同 URL 内容变化，不能复用过期副本 |

**通用设置（Caddyfile 全局选项仅 `email`；监听 80/443、HTTP→HTTPS 跳转、HTTP/2、HTTP/3 与 gzip/zstd 压缩均由 Caddy 自动处理；Docker 镜像由 supervisord 托管 `caddy run`）**：
每个文档项目渲染为独立站点块（地址 = 默认域名 + 自定义域名），证书按域名自动申请；`auto_https = off` 时站点地址加 `http://` 前缀、仅提供 HTTP（内网/开发环境）。

**模板（`pkg/lib/caddy.go`）** 按上述分级缓存渲染，开关来自 `[caddy]` 分区：

| 配置键 | 默认 | 说明 |
|---|---|---|
| `static_expires` | 3600 | 静态资源缓存秒数，0 表示不下发缓存头 |
| `html_nocache` | on | HTML 使用 `no-cache` 协商缓存 |

注意：

- 修改缓存配置后，已存在项目需触发一次重渲染才会生效（如 `rtfd p u -t single:<原值> <name>`）
- 静态资源强缓存期内若重建了同名静态文件，浏览器可能仍用旧副本（Sphinx 资源名不带 hash，
  建议取值不宜过长，或用版本号目录隔离）

访问页面的语言/版本切换由注入页面的 `rtfd.js` 挂件完成：构建时 URL query 携带
`name/branch/rtfd_api`，运行时用原生 `fetch` 向 `GET /rtfd/:name/desc` 拉取元数据，
在页面右下角渲染浮动面板（默认只显示当前语言与版本，点击展开语言/版本/仓库链接）。

---

## 7. 对外 API（`api` 包，均挂 `/rtfd` 前缀）

路由注册集中在 `api.registerRoutes`（`api.New` 供启动与测试复用），兼容两种路径风格
（`:name` 在前或在后），新增接口建议同时注册两种。

**公开接口**（无需密钥）：

| 路由 | 方法 | 功能 | 说明 |
|---|---|---|---|
| `/:name/desc`、`/desc/:name` | GET | 项目描述 | 返回 URL、langs、latest、versions、builder、showNav 等，供 rtfd.js 渲染 |
| `/:name/badge`、`/badge/:name` | GET | 文档状态徽章 SVG | `?branch=`，默认 Latest；passing/failing/unknown |
| `/:name/build`、`/build/:name` | POST | 触发构建 | 动态签名（X-Rtfd-Ts/Nonce/Sign，HMAC-SHA256，密钥为项目 secret；空 secret 免鉴权）；参数 branch、debug；异步执行返回 201 |
| `/:name/webhook`、`/webhook/:name` | POST | git webhook | 校验 GitHub `X-Hub-Signature`(sha1=HMAC) / Gitee `X-Gitee-Token`；识别 UA 分派；ping→pong；排除 `excluded_branch` |
| `/assets/rtfd.js` | GET/HEAD | 静态挂件脚本 | 内嵌 assets.RtfdJS |
| `/github/app` | POST | GitHub App 事件 | `installation`/`installation_repositories`，校验 App ID 后 `Dispatch` |

**管理接口**（对应 CLI 的 `project` 子命令，方便实现 Web 管理端）：

| 路由 | 方法 | 功能 | 参数（支持表单/query/JSON） |
|---|---|---|---|
| `/projects` | GET | 项目列表 | `verbose=1` 返回完整 Options 数组，否则仅名称数组 |
| `/projects` | POST | 创建项目 | `name`、`url` 必需，其余同 CLI create（`latest/version/single/sourcedir/lang/requirement/install/index/builder/secret/domain/sslcrt/sslkey`），空值沿用系统默认 |
| `/:name/info`、`/info/:name` | GET | 项目详情 | `key=Field` 返回单字段、`build=1` 附带构建集，否则返回 Options 结构体（字段名为结构体字段名） |
| `/:name/update`、`/update/:name` | POST | 更新配置 | `text=Field:Value,…`（`sep` 可自定义分隔符）、`file=服务端规则文件路径`、或直接以字段名传参；响应含 `updated/failed` 字段列表 |
| `/:name/remove`、`/remove/:name` | POST/DELETE | 删除项目 | 同时清理构建结果并从 Caddy 配置移除站点 |
| `/:name/export`、`/export/:name` | GET | 导出 base64 配置 | `sysmeta=1` 保留内置 meta |
| `/import` | POST | 导入 base64 配置 | `export` 必需、`name` 可选（别名覆盖） |

管理接口鉴权：动态签名，随请求携带 `X-Rtfd-Ts`（Unix 秒）、`X-Rtfd-Nonce`（随机串）、`X-Rtfd-Sign`（HMAC-SHA256），
签名串 = `ts + "\n" + nonce + "\n" + METHOD(大写) + "\n" + path + "\n" + sha256hex(body)`，密钥取自配置 `[api] secret`；
项目级接口（info/update/remove/export）同时接受项目自身密钥。`[api] secret` 未配置时
管理接口一律拒绝（`api secret is not configured`），避免 Web 管理能力被意外暴露；
构建与 webhook 仍按项目密钥的原逻辑运行（空 secret 免鉴权）。可用 `rtfd sign` 生成签名，Swagger UI 首页填入密钥后自动签名。

统一响应：`{"success": bool, "message": string, "data": any}`；错误由 `customHTTPErrorHandler`
输出（默认 HTTP 200，echo.HTTPError 除外）。

### 7.1 接口文档（Swagger/OpenAPI）

- 注解写在每个 handler 上方（swaggo 风格）：`@Summary/@Description/@Tags/@Produce/@Param/@Success/@Failure/@Security/@Router`；
  通用信息（`@title/@version/@host/@securityDefinitions.apikey RtfdSign` 等）在 `main.go`
- 生成：`make docs`（等价 `swag init -g main.go -o docs --ot go,json,yaml --propertyStrategy pascalcase`，swag 版本在 Makefile 里固定）；
  产物 `docs/{docs.go,swagger.json,swagger.yaml}` **需提交**，`api` 空白导入 `docs` 包后注册到 swag 注册表
- 在线查看：`GET /rtfd/docs`（301 到 `/rtfd/docs/index.html`）、规范文件 `/rtfd/docs/doc.json`、`/rtfd/docs/doc.yaml`；
  Swagger UI 静态资源由 `swaggo/files` 内嵌（离线可用、无需鉴权），会使二进制增大约 8MB
- 生成约定：swagger 2.0 中同一状态码只能有一个响应，因此业务错误统一记为 `@Failure default {object} res`
  （实际仍为 HTTP 200）；`data` 用 `resd{data=<类型>}` 语法展开为具体模型；
  `--propertyStrategy pascalcase` 是为了让未加 json tag 的 `lib.Options/Result` 字段名与实际响应（首字母大写）一致
- 接口与文档一致性由 `api/docs_test.go`（UI/规范可访问、路径与安全定义齐全）和 CI 的 `make docs && git diff --exit-code docs`
  兜底；不需要内嵌 UI 时可去掉 echo-swagger 依赖与 `/docs` 路由，仅保留 `docs/swagger.json` 供外部工具使用

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

除 `api` 外，`project`、`build` 的每个动作都有对应的 HTTP 接口（见第 7 节管理接口表），
`cmd` 与 `api` 两层都只做参数解析，业务统一下沉 `pkg/lib`（如 `CreateProject`、`ParseUpdateRule`、
`Export/Import`、`Update`），保证两种入口行为一致。

根命令 `initConfig`：除 `-h/-v/-i/--init` 外，所有子命令要求配置文件存在，否则打印提示并 `os.Exit(127)`。
`rtfd --init` 生成默认配置时会读取环境变量预填两个必填项：`RTFD_API_SERVER_URL`→`[api] server_url`、`RTFD_CADDY_DN`→`[caddy] dn`；未提供则留空并在输出中提示需手动设置。
错误处理约定：cmd 层 `fmt.Println(err)` + `os.Exit`，退出码大致分为
127（配置/项目不可用）、128（项目/域名已存在）、129（参数非法）、130（操作失败），
含义有一定重叠，新命令保持一致风格即可。

---

## 9. GitHub App 集成（`pkg/lib/app.go`）

目的：创建 GitHub 项目时**自动注册仓库 webhook**，删除时自动清理，无需用户手动配置。

- 配置段 `[ghapp]`：`app_id/private_key`（PEM）+ `api.server_url`；`app_id` 与 `private_key` 同时有效即启用（无独立开关）。
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
| （default） | base_dir* / default_branch / unallowed_name / log_level | 数据根目录（初始化后勿改，否则丢数据）等 |
| [database] | type* / dsn* | 数据库类型 `sqlite`/`mysql`/`pgsql`（默认sqlite）、连接串（sqlite为文件路径，支持 `%(base_dir)s` 插值）；是否打印SQL 由顶层 `log_level = debug` 控制（不再单独提供 `database.debug`） |
| [caddy] | dn* / exec / sudo / email / auto_https / conf_dir | 托管域名后缀（必填）；caddy 路径与是否 sudo；ACME 邮箱；是否自动 HTTPS（默认on，内网/开发可 off）；配置目录；`dn` 可由 `rtfd --init` 读取 `RTFD_CADDY_DN` 环境变量自动填入 |
| [caddy] | static_expires / html_nocache | 静态资源缓存秒数（默认3600）、HTML 协商缓存（默认on），详见 6.1 |
| [py] | `<版本号>`* / default / index | 可用 Python 版本映射（至少一项，键为版本号如 3、3.10、3.12，值为程序路径，要求带 pip+virtualenv）、默认版本（缺省取第一个可用版本）、pip 源 |
| [api] | host / port / server_url* / secret | 监听；`server_url` 为**必填**项（rtfd.cfg 默认留空），供 webhook 回跳、文档挂件脚本注入，须配置为对外可达地址；`rtfd --init` 会读取 `RTFD_API_SERVER_URL` 环境变量自动填入，未设置则留空需手动补填；配置若为 `0.0.0.0`/`::` 等不可路由地址会归一化为 `127.0.0.1` 并告警、管理接口密钥（见第7节） |
| [ghapp] | app_id / private_key | GitHub Apps 凭据（两者同时有效即启用，无独立开关） |

> * = 必需。conf_dir 默认 `%(base_dir)s/caddy`。
> 配置默认值同步维护于 `assets/rtfd.cfg`（`rtfd --init` 写入），变更需同步 `main_test.go` 断言。

---

## 11. 目录结构与包职责

```
rtfd/
├── main.go           程序入口，仅空白导入 assets（embed 生效）+ 执行 cmd.Execute
├── main_test.go      默认配置结构断言（base_dir/caddy/py/api 分区完整性）
├── api/              API 层：api.go(路由注册/New/Start) · view.go(挂件/构建/webhook处理器)
│                     · manage.go(项目管理处理器) · tool.go(参数解析/密钥校验等公共函数)
├── docs/             swag 生成的接口文档（make docs）：docs.go · swagger.json · swagger.yaml
├── assets/           静态资源(go:embed)：rtfd.cfg · builder.sh · rtfd.js · VERSION · swagger.html · rtfd.ini(样例)
├── cmd/              cobra 命令层：root/api/cfg/build/project(+create/get/list/remove/transfer/update)
├── pkg/
│   ├── build/        构建编排：Builder + builder.sh 落盘与输出流解析
│   ├── conf/         ini 配置封装（SecHash/GetKey/BaseDir/DefaultBranch…）
│   ├── lib/          ★ 核心业务：
│   │   ├── lib.go     ProjectManager 项目 CRUD（含 CreateProject 参数入口）、模型转换、Caddy 配置汇总渲染调度
│   │   ├── caddy.go   Caddyfile 模板渲染（站点块/跳转/缓存）
│   │   ├── update.go  更新字段分派（updateHook）、规则解析（ParseUpdateRule/ParseUpdateFile）与安全校验
│   │   ├── transfer.go 项目配置导出/导入（Export/DecodeExport/Import）
│   │   └── app.go     GitHub App（JWT/installation token/webhook 同步）
│   └── util/         纯工具：命令执行、正则校验、git URL/域名解析、HMAC-SHA1
├── vars/             跨包常量（Sender、GitHub/Gitee 常量、ResetEmpty、默认版本…）
├── scripts/          部署：supervisord.conf / rtfd.service / start.sh
├── Makefile          构建/测试/发布目标
├── Dockerfile        多阶段构建：golang:1.26-alpine 编译 → ubuntu:24.04 运行镜像(+ caddy + supervisor)，
│                     运行镜像用 apt + deadsnakes PPA 固定预装 3.10/3.12（系统自带 3.12，deadsnakes 补 3.10），
│                     版本列表写死在 assets/rtfd.cfg 的 [py] 分区；不再依赖 uv 动态安装
└── .github/workflows/ gotest.yml(测试) · publish.yml(镜像 master→latest、dev→dev、release) · goreleaser.yml(tag→多平台二进制)
```

代码风格要点：每个 `.go` 文件带 Apache-2.0 License 头；导入分组
标准库 / 项目内部（`pkg.tcw.im/rtfd/v2/*`）/ 外部依赖（`github.com`、`pkg.tcw.im`）；
注释与用户提示以中文为主；全局常量集中在 `vars`。

---

## 12. 构建、发布与 CI

- 本地：`make dev`（go run ./ api）、`make test`（`go test -count=1 ./...`）、
  `make build-arm/amd`（linux 交叉编译，`-ldflags -X cmd.commitID/-X cmd.built`）、
  `make release`（编译并打 tar.gz）。
- 版本：版本号存 `assets/VERSION`（当前 1.5.0），`-v` 输出；
  tag `v*` 推送触发 GoReleaser（linux amd64/386/arm64 + checksum）。
- 镜像：`publish.yml` 在 master→`latest`、dev→`dev`、release published 时构建，
  运行时镜像内含 caddy + python3(3.12) + supervisor（supervisord 拉起 `rtfd api` 与 caddy）。
- 镜像内多版本 Python：运行镜像基于 `ubuntu:24.04`，系统自带 `python3`(=3.12，版本号 `3`)；
  通过 `apt` + deadsnakes PPA 固定预装 `3.10`（`python3.10 -m ensurepip` 自举 pip 后
  `pip install --break-system-packages virtualenv`，系统 3.12 的 virtualenv 由 apt `python3-virtualenv` 提供），
  三者均以 `版本号 = /usr/bin/pythonX.Y` 写死在 `assets/rtfd.cfg` 的 `[py]` 分区，`default = 3`；
  不再使用 uv 动态安装，版本调整需同时改 Dockerfile 的 apt 安装与 rtfd.cfg 的 [py] 分区。
- 测试：纯单元测试与 sqlite 集成测试（store 连接/迁移、项目 CRUD、conf、Caddy 渲染、默认配置），
  不依赖网络与外部数据库服务。

---

## 13. 安全要点

| 风险面 | 防护 |
|---|---|
| 私有仓库 | URL 内嵌凭据仅 http(s) 的 github.com/gitee.com；对外接口经 `PublicGitURL` 剥敏 |
| 构建/Webhook 伪造 | build API：HMAC-SHA256 动态签名（密钥为项目 secret，空 secret 免鉴权）；GitHub：`X-Hub-Signature` HMAC-SHA1；Gitee：`X-Gitee-Token` 比对；`ping` 仅回 pong |
| 管理接口滥用 | 管理接口需 `[api] secret`（未配置则直接拒绝），项目级接口额外接受项目密钥；CORS 仅放开必要方法与 `X-Rtfd-Ts/Nonce/Sign` 头 |
| 路径穿越 | `latest/sourcedir/requirement` 禁以 `/`、`..` 开头；meta key 白名单正则 |
| 名称/域名注入 | name 正则、域名 `IsDomain` 校验、自定义域名占用检查（`HasCustomDomain`） |
| GitHub App | 事件类型/目标类型白名单、installation App ID 与 header 强校验；webhook 仅 push/release |
| 保留名 | `www` 及 `unallowed_name` 列表内名称禁止创建 |
| 存储敏感信息 | secret 存 projects 表，日志不输出明文；导出时默认剔除系统 meta |
| 数据库连接串 | dsn 含账号密码，配置文件权限应收敛（建议 0600）；`log_level = debug` 时 SQL 会入日志，勿在生产开启 |

---

## 14. 迁移说明（Redis → 关系型数据库）

rtfd 自本次重构起不再使用 Redis，元数据存 sqlite / mysql / pgsql。
旧版本数据可通过**项目转储（transfer）机制**迁移，它与存储实现无关（导出/导入的是 Options JSON）：

```bash
# 1. 旧版本（Redis）：列出并逐个导出为 base64
rtfd p l
rtfd p t -e <NAME>

# 2. 新版本：配置好 [database] 后逐个导入
rtfd p t -i <BASE64> [新名称]
```

要点：

- 导入走的是 `Create`，会重新校验名称/域名/python 版本并渲染 Caddy 配置，因此**需先确认新环境的
  `[py]` 版本、`[caddy] dn` 等与旧环境一致**
- `Meta` 中的系统字段（`_webhook_id`、`_installation_id`）默认不导出（`--export-sys-meta` 可含），
  GitHub App 项目导入后可重新触发 webhook 同步
- 构建结果（Redis 中的 `builder:{name}`）不迁移，迁移后重新构建即可生成
- sqlite 文件建议放在 `base_dir` 内（`dsn = %(base_dir)s/rtfd.db`），并纳入备份
