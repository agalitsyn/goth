BUILD_DIR := $(CURDIR)/bin

GIT_TAG := $(shell git describe --tags --always --abbrev=0)

BUILD_ARGS ?= -ldflags \
	"-X github.com/agalitsyn/goth/internal/version.Tag=$(GIT_TAG)"

export PATH := $(BUILD_DIR):$(PATH)

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

include bin-deps.mk

.PHONY: go-run
go-run:
	go run -mod=vendor ./cmd/app

.PHONY: go-watch
go-watch: $(AIR_BIN)
	$(AIR_BIN)

.PHONY: go-build
go-build: $(BUILD_DIR)
	go build -mod=vendor -v $(BUILD_ARGS) -o $(BUILD_DIR) ./cmd/...

.PHONY: go-test
go-test:
	go test -v -race ./...

.PHONY: go-fmt
go-fmt:
	${GOLANGCI_BIN} run --fix ./...

.PHONY: go-lint
go-lint:
	@go version
	@${GOLANGCI_BIN} version
	${GOLANGCI_BIN} run ./...

.PHONY: go-generate
go-generate:
	go generate ./...

.PHONY: css-build
css-build: $(TAILWINDCSS_BIN)
	$(TAILWINDCSS_BIN) --minify -i cmd/app/templates/tw.css -o cmd/app/assets/static/css/main.css

.PHONY: css-watch
css-watch: $(TAILWINDCSS_BIN)
	$(TAILWINDCSS_BIN) --watch -i cmd/app/templates/tw.css -o cmd/app/assets/static/css/main.css

.PHONY: js-vendor
js-vendor:
	mkdir -p cmd/app/assets/static/vendor/htmx.org@1.9.10 && wget -O cmd/app/assets/static/vendor/htmx.org@1.9.10/htmx.min.js https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
	mkdir -p cmd/app/assets/static/vendor/alpinejs@3.13.5 && wget -O cmd/app/assets/static/vendor/alpinejs@3.13.5/alpinejs.min.js https://cdn.jsdelivr.net/npm/alpinejs@3.13.5/dist/cdn.min.js

.PHONY: templ-watch
templ-watch: $(TEMPL_BIN)
	$(TEMPL_BIN) generate -watch -proxy=http://localhost:8080

.PHONY: templ-build
templ-build: $(TEMPL_BIN)
	$(TEMPL_BIN) generate
