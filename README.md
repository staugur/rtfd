rtfd
====

Build, read your exclusive and fuck docs.

[![Go test](https://github.com/staugur/rtfd/actions/workflows/go.yml/badge.svg)](https://github.com/staugur/rtfd/actions/workflows/go.yml)
[![Documentation Status](https://open.saintic.com/rtfd/badge/saintic-docs)](https://docs.saintic.com/rtfd/)
[![Go Reference](https://pkg.go.dev/badge/tcw.im/rtfd.svg)](https://pkg.go.dev/tcw.im/rtfd)

安装
-------

rtfd 仅支持 linux 操作系统

### **使用已编译的正式版本**

```bash
version=1.0.0
wget -c https://github.com/staugur/rtfd/releases/download/v${version}/rtfd.${version}-linux-amd64.tar.gz
tar zxf rtfd.${version}-linux-amd64.tar.gz
mv rtfd ~/bin/
rtfd -v
```

### **自行编译最新版**

1. 安装golang环境，版本1.16+

2. 编译安装

    2.1 下载源码编译：
    ```bash
    git clone https://github.com/staugur/rtfd && cd rtfd
    make build
    mv bin/rtfd ~/bin
    rtfd -v
    ```

    2.2 使用`go get`命令：
    ```bash
    go get -u tcw.im/rtfd
    mv ~/go/bih/rtfd ~/bin/
    rtfd -v
    ```

    ps：这种方式没有 `-v/-i` 选项无法输出版本号及信息。

### **使用docker安装**

`docker pull staugur/rtfd`

或下载源码自行构建

```bash
git clone https://github.com/staugur/rtfd && cd rtfd
docker build -t staugur/rtfd .
```

ps：此方式主要是用来运行 API 服务

使用
------

```bash
rtfd --init
rtfd p create --url {git-url} --other-options {ProjectName}
rtfd build {ProjectName}
```

More options with `--help / -h` option.

文档
------

More please see the [detailed documentation](https://docs.saintic.com/rtfd)
