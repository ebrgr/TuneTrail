ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.DEFAULT_GOAL := help
YARN ?= corepack yarn
DLV_VERSION ?= v1.27.2

.PHONY: help setup doctor api.deps api.dev api.debug api.vet api.test api.race api.build debugger.deps docs.dev check multiuser.check test build

help:
	@echo "make setup           Install project dependencies and create .env"
	@echo "make doctor          Check local prerequisites"
	@echo "make api.dev         Run Go API on 127.0.0.1:4110"
	@echo "make api.debug       Run API under Delve in the terminal"
	@echo "make check           Run Go and interface checks"
	@echo "make api.race        Run Go race tests (requires a C compiler)"
	@echo "make build           Build the Go application"
	@echo "make multiuser.check Validate multiuser journeys on disposable databases"
	@echo "make ui.preview      Run the isolated Go visual preview (no database)"
	@echo "make ui.check        Check Go pages, preview and their JavaScript"

setup: doctor
	@test -f .env || cp .env.example .env
	$(MAKE) api.deps debugger.deps

doctor:
	go version
	node --version

api.deps:
	go mod download

debugger.deps:
	GOBIN="$(CURDIR)/.bin" go install github.com/go-delve/delve/cmd/dlv@$(DLV_VERSION)

api.dev:
	CGO_ENABLED=0 go run ./cmd/api

api.debug:
	CGO_ENABLED=0 ./.bin/dlv debug ./cmd/api

api.vet:
	CGO_ENABLED=0 go vet ./...

api.test:
	CGO_ENABLED=0 go test ./... -timeout 60s

api.race:
	CGO_ENABLED=1 go test -race ./... -timeout 120s

api.build:
	CGO_ENABLED=0 go build -ldflags='-s -w' -o tunetrail ./cmd/api/main.go

docs.dev:
	cd docs && $(YARN) dev

check: api.vet api.test ui.check

.PHONY: ui.preview ui.check
ui.preview:
	CGO_ENABLED=0 go run ./cmd/ui-preview

ui.check:
	CGO_ENABLED=0 go vet ./internal/webui ./internal/webquery ./internal/listening ./cmd/ui-preview
	CGO_ENABLED=0 go test ./internal/webui ./internal/webquery ./internal/listening ./cmd/ui-preview
	node --check internal/webui/static/theme.js
	node --check internal/webui/static/ui.js
	node --check internal/webui/static/live.js
	node --check internal/webui/static/admin.js
	node --test internal/webui/theme.test.mjs

test: api.test

build: api.build

# Starts isolated loopback APIs; never uses the database configured in .env.
multiuser.check: api.build
	python3 scripts/validate_multiuser.py --binary ./tunetrail

# Increment the TuneTrail 0.x version once per new commit.
.PHONY: version.check version.bump
version.check:
	python3 scripts/version.py

version.bump:
	python3 scripts/version.py --bump
