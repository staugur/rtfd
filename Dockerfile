ARG buildos=golang:1.17.2-alpine
ARG runos=python:2.7.18-alpine

# -- build dependencies with alpine --
FROM $buildos AS builder

WORKDIR /build

COPY . .

ARG goproxy

ARG TARGETARCH=amd64

RUN if [ "x$goproxy" != "x" ]; then go env -w GOPROXY=${goproxy},direct; fi ; \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags "-s -w -X tcw.im/rtfd/cmd.built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" .

# -- run application with a small image --
FROM $runos

COPY --from=builder /build/rtfd /bin/

WORKDIR /rtfd

RUN apk add --no-cache nginx python3 py3-pip bash git && \
    pip2 install --no-cache-dir virtualenv && \
    pip3 install --no-cache-dir virtualenv supervisor

COPY scripts/supervisord.conf /etc/

COPY scripts/nginx.conf /etc/nginx/

COPY assets/rtfd.cfg /

ENV RTFD_CFG=/rtfd.cfg

EXPOSE 80 443 5000

ENTRYPOINT ["supervisord", "-c", "/etc/supervisord.conf"]
