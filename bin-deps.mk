GOLANGCI_BIN ?= $(GOBIN)/golangci-lint
GOLANGCI_VERSION ?= v1.56.2

.PHONY: $(GOLANGCI_BIN)
$(GOLANGCI_BIN):
	test -s $(GOLANGCI_BIN) || GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)


TEMPL_BIN ?= $(GOBIN)/templ
TEMPL_VERSION ?= v0.2.590

.PHONY: $(TEMPL_BIN)
$(TEMPL_BIN):
	test -s $(TEMPL_BIN) || GOBIN=$(GOBIN) go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)


TAILWINDCSS_BIN ?= node_modules/.bin/tailwindcss

.PHONY: $(TAILWINDCSS_BIN)
$(TAILWINDCSS_BIN):
	test -s $(TAILWINDCSS_BIN) || npm i


AIR_BIN ?= $(GOBIN)/air
AIR_VERSION ?= v1.51.0

.PHONY: $(AIR_BIN)
$(AIR_BIN):
	test -s $(AIR_BIN) || GOBIN=$(GOBIN) go install github.com/cosmtrek/air@$(AIR_VERSION)


TERN_BIN ?= $(GOBIN)/tern
TERN_VERSION ?= latest

$(TERN_BIN):
	go install github.com/jackc/tern@$(TERN_VERSION)
