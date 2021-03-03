.PHONY: help

BINARY=rtfd
CommitID=$(shell git log --pretty=format:"%h" -1)
Built=$(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
Version=$(shell cat VERSION)
LDFLAGS=-ldflags "-s -w -X tcw.im/rtfd/cmd.commitID=${CommitID} -X tcw.im/rtfd/cmd.built=${Built} -X tcw.im/rtfd/cmd.version=${Version}"

help:
	@echo "  make clean  - Remove binaries and vim swap files"
	@echo "  make gotool - Run go tool 'fmt' and 'vet'"
	@echo "  make build  - Compile go code and generate binary file"
	@echo "  make dev    - Run dev server"
	@echo "  make test   - Run go test"

gotool:
	go fmt ./
	go vet ./

build:
	go build ${LDFLAGS} -o bin/$(BINARY) && chmod +x bin/$(BINARY)

docker:
	docker build -t staugur/rtfd .

dev:
	@echo Starting service...
	@go run ./

test:
	@go test -count=1 ./...
