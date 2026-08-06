APP := hps
VERSION ?= 0.1.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PREFIX ?= /usr/local
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)

.PHONY: all build run fmt fmt-check vet test test-race check install clean

all: check build

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/hps

run:
	go run ./cmd/hps

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "Run 'make fmt' on:"; gofmt -l .; exit 1; }

vet:
	go vet ./...

test:
	go test -cover ./...

test-race:
	go test -race ./...

check: fmt-check vet test

install: build
	install -Dm0755 bin/$(APP) $(DESTDIR)$(PREFIX)/bin/$(APP)

clean:
	rm -rf bin coverage.out
