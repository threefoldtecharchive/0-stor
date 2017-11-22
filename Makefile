OUTPUT ?= bin
GOOS ?= linux
GOARCH ?= amd64

TIMEOUT ?= 10m

PACKAGE = github.com/zero-os/0-stor
COMMIT_HASH = $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE = $(shell date +%FT%T%z)

SERVER_PACKAGES = $(shell go list ./server/...)
CLIENT_PACKAGES = $(shell go list ./client/...)

ldflags = -extldflags "-static"
ldflagsversion = -X main.CommitHash=$(COMMIT_HASH) -X main.BuildDate=$(BUILD_DATE) -s -w

all: server cli

cli: $(OUTPUT)
ifeq ($(GOOS), darwin)
	GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags '$(ldflagsversion)' -o $(OUTPUT)/zerostorcli ./cmd/zerostorcli
else
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags '$(ldflags)$(ldflagsversion)' -o $(OUTPUT)/zerostorcli ./cmd/zerostorcli
endif

server: $(OUTPUT)
ifeq ($(GOOS), darwin)
	GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags '$(ldflagsversion)' -o $(OUTPUT)/zerostorserver ./cmd/zerostorserver
else
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags '$(ldflags)$(ldflagsversion)' -o $(OUTPUT)/zerostorserver ./cmd/zerostorserver
endif

install: all
	cp $(OUTPUT)/zerostorcli $(GOPATH)/bin/zerostorcli
	cp $(OUTPUT)/zerostorserver $(GOPATH)/bin/zerostorserver

test: testserver testclient

testrace: testserverrace testclientrace

testserver:
	go test -v -timeout $(TIMEOUT) $(SERVER_PACKAGES)

testclient:
	go test  -v -timeout $(TIMEOUT) $(CLIENT_PACKAGES)

testserverrace:
	go test  -v -race $(SERVER_PACKAGES)

testclientrace:
	go test  -v -race $(CLIENT_PACKAGES)

update_deps:
	dep ensure -update -v
	dep prune -v


$(OUTPUT):
	mkdir -p $(OUTPUT)

.PHONY: $(OUTPUT) zerostorcli 0storserver test testserver testclient testserverrace testclientrace
