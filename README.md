rtfd
====

Build, read your exclusive and fuck docs.

[![Go Reference](https://pkg.go.dev/badge/tcw.im/rtfd.svg)](https://pkg.go.dev/tcw.im/rtfd)
[![Documentation Status](https://open.saintic.com/rtfd/saintic-docs/badge)](https://docs.saintic.com/rtfd/)
[![Go test](https://github.com/staugur/rtfd/actions/workflows/go.yml/badge.svg)](https://github.com/staugur/rtfd/actions/workflows/go.yml)

安装
-------

rtfd 仅支持 linux 操作系统

### **使用编译好的可执行程序**

```bash
version=1.2.0
wget -c https://github.com/staugur/rtfd/releases/download/v${version}/rtfd.${version}-linux-amd64.tar.gz
tar zxf rtfd.${version}-linux-amd64.tar.gz
mv rtfd ~/bin/
rtfd -v
```

### **使用源码编译最新版**

1. 安装golang环境，版本1.16+

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
    go get -u tcw.im/rtfd      # 可使用 @tag 安装某个正式版本，如 @v1.1.0
    mv ~/go/bih/rtfd ~/bin/
    rtfd -v
    ```

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
