SHELL := /bin/bash

.PHONY: dev build test lint generate css css-watch setup clean run check help

BINARY_NAME=detent.build

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

css:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --minify

css-watch:
	npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css --watch

check: generate css
	go vet ./...
	go test -race ./...
	golangci-lint run

setup:
	go install github.com/air-verse/air@latest
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/templui/templui/cmd/templui@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	npm install

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
	@echo "  css        - Build Tailwind CSS"
	@echo "  css-watch  - Watch and rebuild Tailwind CSS"
	@echo "  check      - Full validation gate: generate, vet, test, lint"
	@echo "  setup      - Install development tools"
	@echo "  clean      - Remove build artifacts"
	@echo "  run        - Build and run the server"
