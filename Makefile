export GOBIN ?= $(CURDIR)/bin
export PATH := $(GOBIN):$(PATH)

GIT_TAG := $(shell git describe --tags --always --abbrev=0)

BUILD_ARGS ?= -ldflags \
	"-X github.com/agalitsyn/goth/internal/version.Tag=$(GIT_TAG)"

export PATH := $(GOBIN):$(PATH)

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: go-build
go-build: $(GOBIN)
	go build -mod=vendor -v $(BUILD_ARGS) -o $(GOBIN) ./cmd/...

include bin-deps.mk

$(GOBIN):
	mkdir -p $(GOBIN)

.PHONY: run-admin
run-admin: $(AIR_BIN)
	$(AIR_BIN) -c .admin.air.toml

.PHONY: run-app
run-app: $(AIR_BIN)
	$(AIR_BIN) -c .app.air.toml

.PHONY: run-cli
run-cli:
	go run -mod=vendor $(CURDIR)/cmd/cli $(filter-out $@,$(MAKECMDGOALS))

.PHONY: test-short
test-short:
	go test -v -race -short ./...

.PHONY: test
test:
	go test -v -race ./...

.PHONY: fmt
fmt:
	${GOLANGCI_BIN} run --fix ./...

.PHONY: lint
lint:
	@go version
	@${GOLANGCI_BIN} version
	${GOLANGCI_BIN} run ./...

.PHONY: generate
generate:
	go generate ./...

.PHONY: tw-build
tw-build: $(TAILWINDCSS_BIN)
	$(TAILWINDCSS_BIN) --minify -i cmd/app/templates/tw.css -o cmd/app/assets/static/css/main.css

.PHONY: tw-watch
tw-watch: $(TAILWINDCSS_BIN)
	$(TAILWINDCSS_BIN) --watch -i cmd/app/templates/tw.css -o cmd/app/assets/static/css/main.css

.PHONY: templ-build
templ-build: $(TEMPL_BIN)
	$(TEMPL_BIN) generate

.PHONY: templ-watch
templ-watch: $(TEMPL_BIN)
	$(TEMPL_BIN) generate -watch -proxy=http://localhost:8080


.PHONY: vendor-app-static
vendor-app-static:
	mkdir -p cmd/app/assets/static/vendor/htmx.org@1.9.10 && wget -O cmd/app/assets/static/vendor/htmx.org@1.9.10/htmx.min.js https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
	mkdir -p cmd/app/assets/static/vendor/alpinejs@3.13.5 && wget -O cmd/app/assets/static/vendor/alpinejs@3.13.5/alpinejs.min.js https://cdn.jsdelivr.net/npm/alpinejs@3.13.5/dist/cdn.min.js

.PHONY: vendor-admin-static
vendor-admin-static:
	mkdir -p cmd/admin/static/vendor/htmx.org@1.9.10 && \
		wget -O cmd/admin/static/vendor/htmx.org@1.9.10/htmx.min.js https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
	mkdir -p cmd/admin/static/vendor/alpinejs@3.13.5 && \
		wget -O cmd/admin/static/vendor/alpinejs@3.13.5/alpinejs.min.js https://cdn.jsdelivr.net/npm/alpinejs@3.13.5/dist/cdn.min.js
	mkdir -p cmd/admin/static/vendor/bootstrap@5.3.3 && \
		wget -O cmd/admin/static/vendor/bootstrap@5.3.3/bootstrap.min.css https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css && \
		wget -O cmd/admin/static/vendor/bootstrap@5.3.3/bootstrap.bundle.min.js https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js
	mkdir -p cmd/admin/static/vendor/popperjs@@2.11.8 && \
		wget -O cmd/admin/static/vendor/popperjs@@2.11.8/popper.min.js https://cdn.jsdelivr.net/npm/@popperjs/core@2.11.8/dist/umd/popper.min.js
	mkdir -p cmd/admin/static/vendor/bootstrap-icons@1.11.3 && \
		wget -O cmd/admin/static/vendor/bootstrap-icons@1.11.3/bootstrap-icons.min.css https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css
	mkdir -p cmd/admin/static/vendor/sweetalert2@11 &&
		wget -O cmd/admin/static/vendor/sweetalert2@11/sweetalert2.min.js https://cdn.jsdelivr.net/npm/sweetalert2@11/dist/sweetalert2.min.js && \
		wget -O cmd/admin/static/vendor/sweetalert2@11/sweetalert-bootstrap-4.min.css https://cdn.jsdelivr.net/npm/@sweetalert2/theme-bootstrap-4/bootstrap-4.min.css


.PHONY: db-migrate-run
db-migrate-run: $(TERN_BIN)
	#$(TERN_BIN) migrate --migrations=./migrations
	go run $(CURDIR)/cmd/cli admin db migrate

.PHONY: db-migrate-status
db-migrate-status: $(TERN_BIN)
	$(TERN_BIN) status --migrations=./migrations

.PHONY: db-new-migration
db-new-migration: $(TERN_BIN)
	$(TERN_BIN) new --migrations=./migrations $(filter-out $@,$(MAKECMDGOALS))

.PHONY: db-create-superuser
db-create-superuser:
	go run $(CURDIR)/cmd/cli admin user create --login='admin' --password='admin'

