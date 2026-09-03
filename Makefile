SHELL := /bin/bash

.PHONY: dev build test lint generate docs-sync docs-build docs-check css css-watch setup clean run check smoke help

SMOKE_URL ?= https://detent.build

BINARY_NAME=detent.build
DETENT_TEMP_ROOT=$(or $(TMPDIR),$(TMP),$(TEMP),/tmp)
HYPE_VERSION=$(shell tr -d '[:space:]' < docs/site/hype.version)
HYPE_BIN_DIR=$(DETENT_TEMP_ROOT)/detent-build-tools/hype-$(HYPE_VERSION)
HYPE_BIN=$(HYPE_BIN_DIR)/hype
HYPE_GOPATH=$(DETENT_TEMP_ROOT)/go-path
HYPE_GOMODCACHE=$(DETENT_TEMP_ROOT)/go-mod
HYPE_GOCACHE=$(DETENT_TEMP_ROOT)/go-build

dev:
	@mkdir -p tmp
	@if [ -f tmp/air-combined.log ]; then \
	    mv tmp/air-combined.log tmp/air-combined-$$(date +%Y%m%d-%H%M%S).log; \
	fi
	@ls -t tmp/air-combined-*.log 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
	@air 2>&1 | tee tmp/air-combined.log

build: generate css
	go build -o $(BINARY_NAME) ./cmd/server

test:
	go test -race ./...

lint:
	golangci-lint run
	templ fmt templates/ ui/

generate:
	templ generate

docs-sync:
	go run ./cmd/docs-sync

$(HYPE_BIN): docs/site/hype.version
	@mkdir -p $(HYPE_BIN_DIR) $(HYPE_GOPATH) $(HYPE_GOMODCACHE) $(HYPE_GOCACHE)
	GOPATH=$(HYPE_GOPATH) GOMODCACHE=$(HYPE_GOMODCACHE) GOCACHE=$(HYPE_GOCACHE) GOBIN=$(HYPE_BIN_DIR) go install github.com/gopherguides/hype/cmd/hype@$(HYPE_VERSION)

docs-build: $(HYPE_BIN)
	GOPATH=$(HYPE_GOPATH) GOMODCACHE=$(HYPE_GOMODCACHE) GOCACHE=$(HYPE_GOCACHE) go run ./cmd/docs-build -hype $(HYPE_BIN)

docs-check: $(HYPE_BIN)
	GOPATH=$(HYPE_GOPATH) GOMODCACHE=$(HYPE_GOMODCACHE) GOCACHE=$(HYPE_GOCACHE) go run ./cmd/docs-build -check -hype $(HYPE_BIN)

css:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --minify

css-watch:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --watch

check: docs-check generate css
	go vet ./...
	go test -race ./...
	golangci-lint run

setup:
	go install github.com/air-verse/air@latest
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/templui/templui/cmd/templui@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	npm install

smoke:
	@./scripts/smoke.sh $(SMOKE_URL)

clean:
	rm -f $(BINARY_NAME)
	rm -rf tmp/
	rm -f static/css/output.css

run: build
	./$(BINARY_NAME)

help:
	@echo "Available targets:"
	@echo "  dev        - Run with Air hot reload"
	@echo "  build      - Build the binary (templ + css + go build)"
	@echo "  test       - Run tests"
	@echo "  lint       - Run golangci-lint and templ fmt"
	@echo "  generate   - Generate templ code"
	@echo "  docs-sync  - Verify and vendor the pinned Detent documentation"
	@echo "  docs-build - Generate committed site-authored Markdown with pinned Hype"
	@echo "  docs-check - Verify site-authored Markdown is current"
	@echo "  css        - Build Tailwind CSS"
	@echo "  css-watch  - Watch and rebuild Tailwind CSS"
	@echo "  check      - Full validation gate: generate, vet, test, lint"
	@echo "  smoke      - Post-deploy checks against SMOKE_URL (default https://detent.build)"
	@echo "  setup      - Install development tools"
	@echo "  clean      - Remove build artifacts"
	@echo "  run        - Build and run the server"
