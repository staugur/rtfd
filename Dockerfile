ARG buildos=golang:1.26-alpine
ARG runos=python:3.11-slim
# uv镜像，用于安装预编译的多版本Python（免源码编译）
ARG uvimage=ghcr.io/astral-sh/uv:0.4.20
# 额外安装的Python版本，空格分隔；系统自带的python3（版本号3）始终可用
ARG python_versions="3.10 3.12"

# -- uv 静态二进制来源，单独定义阶段 --
# buildx 不支持在 COPY --from 中做变量展开，需先用 FROM $ARG 定义具名阶段
FROM $uvimage AS uv

# -- build dependencies with alpine --
FROM $buildos AS builder
WORKDIR /build
COPY . .
ARG goproxy
ARG TARGETARCH
RUN if [ "x$goproxy" != "x" ]; then go env -w GOPROXY=${goproxy},direct; fi ;\
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags "-s -w -X pkg/tcw.im/rtfd/cmd.built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" .

# -- run application with a small image --
FROM $runos

# uv安装Python的目录，以及pythonX.Y可执行文件链接目录
ENV UV_PYTHON_INSTALL_DIR=/opt/python \
    XDG_BIN_HOME=/usr/local/bin

COPY --from=uv /uv /usr/local/bin/uv

RUN apt update -y && \
    apt install -y --no-install-recommends ca-certificates nginx python3 python3-pip python3-venv \
    git procps curl gcc g++ make && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# 安装额外的Python版本（预编译包，免源码编译），并为每个版本装上virtualenv：
# 构建时可用 --build-arg python_versions="3.9 3.12" 覆盖，置空则只保留系统python3
ARG python_versions
RUN if [ -n "$python_versions" ]; then \
        uv python install $python_versions && \
        for v in $python_versions; do \
            ln -sf "$(uv python find $v)" /usr/local/bin/python$v && \
            python$v -m pip install --no-cache-dir --upgrade pip virtualenv; \
        done; \
    fi

RUN python3 -m pip install --upgrade pip && \
    python3 -m pip install --no-cache-dir virtualenv setuptools supervisor

COPY --from=builder /build/rtfd /bin/
COPY scripts/supervisord.conf /etc/
COPY scripts/nginx.conf /etc/nginx/
COPY assets/rtfd.cfg /

# 将额外Python版本登记到rtfd配置的[py]分区，rtfd才能按版本号选用
ARG python_versions
RUN if [ -n "$python_versions" ]; then \
        for v in $python_versions; do \
            sed -i "/^\[py\]/a $v = /usr/local/bin/python$v" /rtfd.cfg; \
        done; \
        sed -i "/^\[py\]/a default = 3" /rtfd.cfg; \
    fi

ENV RTFD_CFG=/rtfd.cfg
EXPOSE 80 443 5000
ENTRYPOINT ["supervisord"]
