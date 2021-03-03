# -- build dependencies with alpine & go1.16+ --
FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY . .

RUN go env -w GOPROXY=https://goproxy.cn,direct && \
	go build -ldflags "-s -w -X tcw.im/rtfd/cmd.built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" -o bin/rtfd && chmod +x bin/rtfd

# run application with a small image
FROM scratch

COPY --from=builder /build/bin/rtfd /bin/

WORKDIR /rtfd

EXPOSE 5000

ENTRYPOINT ["rtfd", "api", "-c", "/rtfd/.rtfd.cfg"]