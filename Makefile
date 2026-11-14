ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.DEFAULT_GOAL := help
YARN ?= corepack yarn
DLV_VERSION ?= v1.27.2

.PHONY: help setup doctor api.deps api.dev api.debug api.vet api.test api.race api.build debugger.deps docs.dev check e2e multiuser.check test build

help:
	@echo "make setup           Install project dependencies and create .env"
	@echo "make doctor          Check local prerequisites"
	@echo "make api.dev         Run Go API on 127.0.0.1:4110"
	@echo "make api.debug       Run API under Delve in the terminal"
	@echo "make check           Run Go and interface checks"
	@echo "make api.race        Run Go race tests (requires a C compiler)"
	@echo "make build           Build the Go application"
	@echo "make e2e             Run end-to-end journeys on disposable data (part of make check)"
	@echo "make multiuser.check Validate multiuser journeys on disposable databases"
	@echo "make ui.preview      Run the isolated Go visual preview (no database)"
	@echo "make ui.check        Check Go pages, preview and their JavaScript"
	@echo "make issue.new T=..  Create a Backlog.md task (TYPE=bug L=ui M=m-6)"
	@echo "make issue.start ISSUE=N   Branch <type>/issue-NNN-slug and mark it in progress"
	@echo "make issue.review ISSUE=N MSG=..  Run checks and send for acceptance"
	@echo "make issue.publish ISSUE=N     Publish an accepted task: version, tag (DRY=1, PUSH=1)"

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

# check is the quality gate used locally, in CI and by issue.review/issue.publish.
check: api.vet api.test ui.check e2e

# Starts isolated loopback servers on fictional data; never uses the database in .env.
e2e: api.build
	python3 scripts/e2e.py --binary ./tunetrail

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

# The TuneTrail 0.x version increases once per accepted task, in the commit
# created by issue.publish; version.bump stays available for manual use.
.PHONY: version.check version.bump
version.check:
	python3 scripts/version.py

version.bump:
	python3 scripts/version.py --bump

# Backlog.md task cycle (see /.agents/issue-first). ISSUE accepts 60, 060 or ISSUE-060.
# issue.publish pushes only with PUSH=1, which requires the user's authorization.
TYPE ?= bug
.PHONY: issue.new issue.start issue.review issue.publish issue.test
issue.new:
	@test -n "$(T)" || (echo 'use: make issue.new T="Título" [TYPE=bug] [L=ui] [M=m-6]' && exit 1)
	python3 scripts/issue.py new "$(T)" --type $(TYPE) --labels "$(L)" --milestone "$(M)"

issue.start:
	@test -n "$(ISSUE)" || (echo 'use: make issue.start ISSUE=60' && exit 1)
	python3 scripts/issue.py start $(ISSUE)

issue.review:
	@test -n "$(ISSUE)" -a -n "$(MSG)" || (echo 'use: make issue.review ISSUE=60 MSG="Verificação: ..."' && exit 1)
	python3 scripts/issue.py review $(ISSUE) -m "$(MSG)"

issue.publish:
	@test -n "$(ISSUE)" || (echo 'use: make issue.publish ISSUE=60 [DRY=1] [PUSH=1]' && exit 1)
	python3 scripts/issue.py publish $(ISSUE) $(if $(DRY),--dry-run) $(if $(PUSH),--push)

issue.test:
	python3 -m unittest scripts/test_version.py scripts/test_issue.py

# Export a new fixture reference; refuse to overwrite an existing baseline.
.PHONY: ui.reference ui.compare
ui.reference:
	@test -n "$(OUT)" || (echo 'Use make ui.reference OUT=/absolute/new-directory'; exit 1)
	TZ=UTC TUNETRAIL_UI_REFERENCE="$(abspath $(OUT))" CGO_ENABLED=0 go test ./internal/webui -run '^TestUIReference$$' -count=1

ui.compare:
	python3 scripts/compare_ui_reference.py "$(BASE)" "$(NEW)"

.PHONY: ui.reference.test
ui.reference.test:
	python3 -m unittest discover -s scripts -p test_compare_ui_reference.py

ui.check: ui.reference.test
