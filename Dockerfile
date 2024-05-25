ARG buildos=golang:1.20-alpine
ARG runos=python:2.7-slim

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
RUN apt update -y && \
    apt install -y --no-install-recommends nginx python3 python3-pip git procps curl gcc g++ gnupg unixodbc-dev openssl && \
    apt-get install -y software-properties-common ca-certificates &&\
    apt-get install -y build-essential zlib1g-dev libncurses5-dev libgdbm-dev libssl-dev libreadline-dev libffi-dev wget libbz2-dev libsqlite3-dev && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir /python && cd /python && \
    wget https://www.python.org/ftp/python/3.11.1/Python-3.11.1.tgz && \
    tar -zxvf Python-3.11.1.tgz && \
    cd Python-3.11.1 && \
    ls -lhR && \
    ./configure --enable-optimizations && \
    make install && \
    cd / && rm -rf /python

RUN python2 -m pip install --no-cache-dir virtualenv && \
    python3 -m pip install --upgrade pip && \
    python3 -m pip install --no-cache-dir virtualenv setuptools supervisor

COPY --from=builder /build/rtfd /bin/
COPY scripts/supervisord.conf /etc/
COPY scripts/nginx.conf /etc/nginx/
COPY assets/rtfd.cfg /
ENV RTFD_CFG=/rtfd.cfg
EXPOSE 80 443 5000
ENTRYPOINT ["supervisord"]
