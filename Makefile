export CGO_ENABLED=0
VERSION := $(shell git describe --tags --always)
BUILD_INFO := -X "main.Version=$(VERSION)"
FLAGS := -tags='osusergo' -trimpath -ldflags='$(BUILD_INFO) -s -extldflags "-static"'

build: clean
	go build -o ./bin/broom $(FLAGS) cmd/broom/*

release: clean
	GOOS=darwin  GOARCH=arm64 go build -o ./bin/broom-$(VERSION)-macos-arm64 $(FLAGS) cmd/broom/*
	GOOS=darwin  GOARCH=amd64 go build -o ./bin/broom-$(VERSION)-macos-x64   $(FLAGS) cmd/broom/*
	GOOS=linux   GOARCH=amd64 go build -o ./bin/broom-$(VERSION)-linux-x64   $(FLAGS) cmd/broom/*
	GOOS=windows GOARCH=amd64 go build -o ./bin/broom-$(VERSION)-win64.exe   $(FLAGS) cmd/broom/*

clean:
	rm -rf ./bin

lint: lint-gofmt lint-gomod lint-govet lint-staticcheck

lint-gofmt:
ifneq ($(shell gofmt -l . | wc -l),0)
	gofmt -l -d .
	@false
endif

lint-gomod:
ifneq ($(shell go mod tidy -v 2>/dev/stdout | tee /dev/stderr | grep -c 'unused '),0)
	@false
endif

lint-govet:
	go vet ./...

lint-staticcheck:
	staticcheck ./...

test:
	go clean -testcache
	go test $(FLAGS) ./...

.DEFAULT_GOAL := build
.PHONY: build release clean lint lint-gofmt lint-gomod lint-govet lint-staticcheck test
