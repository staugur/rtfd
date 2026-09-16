## rtfd

Build, read your exclusive and fuck docs.

[![Go Reference](https://pkg.go.dev/badge/pkg.tcw.im/rtfd/v2.svg)](https://pkg.go.dev/pkg.tcw.im/rtfd/v2)
[![Documentation Status](https://hub.saintic.com/rtfd/saintic-docs/badge)](https://docs.saintic.com/rtfd/)
[![Go test](https://github.com/staugur/rtfd/actions/workflows/gotest.yml/badge.svg)](https://github.com/staugur/rtfd/actions/workflows/gotest.yml)

### 依赖

理论上 rtfd 仅支持 linux 操作系统！

构建脚本还需要 bash 运行环境，git命令，python3.10+环境（并安装了pip、virtualenv模块），Caddy 服务。

另外，元数据存储使用关系型数据库（sqlite、mysql、pgsql 任选其一，默认sqlite，无需外部服务）；
如使用 GitHub App 功能则需要能访问 GitHub API。

### 安装

#### 使用编译好的可执行程序

```bash
version=2.0.0
wget -c https://github.com/staugur/rtfd/releases/download/v${version}/rtfd.${version}-linux-amd64.tar.gz
tar zxf rtfd.${version}-linux-amd64.tar.gz
mv rtfd ~/bin/
rtfd -v
```

#### 使用源码编译最新版

1. 安装golang环境，版本1.26+

2. 编译安装（以下两种方式）

    2.1 下载源码编译：

    ```bash
    git clone https://github.com/staugur/rtfd && cd rtfd
    make build
    mv bin/rtfd ~/bin
    rtfd -v
    ```

    2.2 使用`go get`命令：

    ```bash
    go get -u pkg.tcw.im/rtfd/v2      # 可使用 @tag 安装某个正式版本，如 @v2.0.0
    mv ~/go/bin/rtfd ~/bin/
    rtfd -v
    ```

### 使用

```bash
rtfd --init
rtfd p create --url {git-url} --other-options {ProjectName}
rtfd build {ProjectName}
```

More options with `--help / -h` option.

### 文档

使用文档请查阅 [detailed documentation](https://docs.saintic.com/rtfd)

HTTP 接口文档（Swagger UI）由 `api` 服务提供，启动后访问 `http://{host}:{port}/rtfd/docs` 即可在线调试；
源码注解变更后需执行 `make docs` 重新生成 `docs/{swagger.json,swagger.yaml,docs.go}` 并一并提交。

### 从旧版本（Redis存储）迁移

1. 用旧版 rtfd(2.0.0之前) 导出全部项目：`rtfd p l | jq -r '.[]' | xargs -I{} rtfd p t -e {}` 得到 base64 串
2. 部署新版并确认 `[database]` 可用后逐个导入：`rtfd p t -i <base64>`（名称已存在时可加新名称作为别名）
