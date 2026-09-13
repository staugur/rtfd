.PHONY: help gotool build build-arm build-amd docs dev test release

BINARY=rtfd
CommitID=$(shell git log --pretty=format:"%h" -1)
Built=$(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
Version=$(shell cat assets/VERSION)
LDFLAGS=-ldflags "-s -w -X pkg.tcw.im/rtfd/v2/cmd.commitID=${CommitID} -X pkg.tcw.im/rtfd/v2/cmd.built=${Built}"
SWAG=github.com/swaggo/swag/cmd/swag@v1.16.2

help:
	@echo "  make clean  - Remove binaries and vim swap files"
	@echo "  make gotool - Run go tool 'fmt' and 'vet'"
	@echo "  make build  - Compile go code and generate binary file"
	@echo "  make docs   - Generate api docs (docs/) from code annotations"
	@echo "  make dev    - Run dev api server"
	@echo "  make test   - Run go test"
	@echo "  make release- Format go code and compile to generate binary release"

gotool:
	go fmt ./
	go vet ./

build:
	go build ${LDFLAGS} -o bin/$(BINARY) && chmod +x bin/$(BINARY)

build-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o bin/$(BINARY) && chmod +x bin/$(BINARY)

build-amd:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/$(BINARY) && chmod +x bin/$(BINARY)

dev:
	@echo Starting service...
	@go run ./ api

test:
	@go test -count=1 ./...

docs:
	go run $(SWAG) init -g main.go -o docs --ot go,json,yaml --propertyStrategy pascalcase

release: gotool
	$(MAKE) build-arm && cd bin/ && tar zcvf $(BINARY).$(Version)-linux-arm64.tar.gz $(BINARY) && rm $(BINARY)
	$(MAKE) build-amd && cd bin/ && tar zcvf $(BINARY).$(Version)-linux-amd64.tar.gz $(BINARY) && rm $(BINARY)
