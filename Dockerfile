ARG buildos=golang:1.26-alpine
ARG runos=ubuntu:24.04
# static-web-server 官方镜像（仅用于提取二进制）；升级时调整标签即可
ARG sws=joseluisq/static-web-server:3.0.0-beta.1-alpine

# -- build dependencies with alpine --
FROM $buildos AS builder
WORKDIR /build
COPY . .
ARG goproxy
ARG TARGETARCH
RUN if [ "x$goproxy" != "x" ]; then go env -w GOPROXY=${goproxy},direct; fi ;\
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags "-s -w -X pkg.tcw.im/rtfd/v2/cmd.built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" .

# -- static-web-server 二进制（从官方镜像提取） --
FROM $sws AS swsbinary

# -- run application with a small image --
FROM $runos

# 固定多版本 Python：基础镜像自带 python3(=3.12)；通过 deadsnakes PPA 安装 3.10，
# 各版本独立落在 /usr/bin/python3.X，互不冲突（相比 uv 动态安装更可控、可复现）。
# 版本列表如需调整，改下面两处 apt 安装 + assets/rtfd.cfg 的 [py] 分区即可。
RUN apt-get update -y && \
    apt-get install -y --no-install-recommends \
        software-properties-common ca-certificates curl git procps tzdata \
        gcc g++ make supervisor && \
    add-apt-repository -y ppa:deadsnakes/ppa && \
    apt-get update -y && \
    apt-get install -y --no-install-recommends \
        python3 python3-venv python3-dev python3-virtualenv \
        python3.10 python3.10-venv python3.10-dev && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# 为 deadsnakes 的 3.10 装入 virtualenv 模块（系统 python3 的 virtualenv 由上面的
# apt python3-virtualenv 提供，已可直接 `python3 -m virtualenv`）。
# deadsnakes 版本不自带 pip，先用 ensurepip 自举；Ubuntu 系统 Python 带 PEP 668 标记，
# 故统一加 --break-system-packages 以免 pip install 被拒绝。
RUN set -eux; \
    for v in 3.10; do \
        python$v -m ensurepip --upgrade; \
        python$v -m pip install --no-cache-dir --break-system-packages virtualenv; \
    done

COPY --from=builder /build/rtfd /bin/
COPY --from=swsbinary /usr/local/bin/static-web-server /usr/local/bin/static-web-server
COPY scripts/supervisord.conf /etc/
COPY assets/rtfd.cfg /

ENV RTFD_CFG=/rtfd.cfg \
    TZ=Asia/Shanghai \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8
EXPOSE 80 443 5000
ENTRYPOINT ["supervisord"]
