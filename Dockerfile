ARG buildos=golang:1.17-alpine
ARG runos=python:2.7-slim

# -- build dependencies with alpine --
FROM $buildos AS builder
WORKDIR /build
COPY . .
ARG goproxy
ARG TARGETARCH
RUN if [ "x$goproxy" != "x" ]; then go env -w GOPROXY=${goproxy},direct; fi ;\
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags "-s -w -X tcw.im/rtfd/cmd.built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" .

# -- run application with a small image --
FROM $runos
RUN apt update && \
    apt install -y --no-install-recommends nginx python3 python3-pip git procps && \
    python2 -m pip install --no-cache-dir virtualenv && \
    python3 -m pip install --upgrade pip && \
    python3 -m pip install --no-cache-dir virtualenv setuptools supervisor && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/rtfd /bin/
COPY scripts/supervisord.conf /etc/
COPY scripts/nginx.conf /etc/nginx/
COPY assets/rtfd.cfg /
ENV RTFD_CFG=/rtfd.cfg
EXPOSE 80 443 5000
ENTRYPOINT ["supervisord"]
