ARG buildos=golang:1.17.1-alpine
ARG runos=python:2.7.18-alpine

# -- build dependencies with alpine --
FROM $buildos AS builder

WORKDIR /build

COPY . .

ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags "-s -w -X tcw.im/rtfd/cmd.built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" . && chmod +x rtfd

# run application with a small image
FROM $runos

COPY --from=builder /build/rtfd /bin/

WORKDIR /rtfd

RUN apk add --no-cache nginx python3 py3-pip && \
    pip2 install virtualenv && \
    pip3 install virtualenv supervisor

COPY assets/supervisord.conf /etc/

COPY assets/nginx.conf /etc/nginx/

COPY assets/rtfd.cfg /

ENV RTFD_CFG=/rtfd.cfg

EXPOSE 80

EXPOSE 443

EXPOSE 5000

CMD ["supervisord", "-c", "/etc/supervisord.conf"]
