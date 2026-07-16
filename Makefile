#! /usr/bin/make
#
# Makefile for Loom
#
# Targets:
# - "depend" retrieves the Go packages needed to run the linter and tests
# - "lint" runs the linter
# - "test" runs the tests
# - "release" verifies a staged release, atomically publishes its commit and tag, and
#   waits for the matching substantive GitHub Release. It requires VERSION=vX.Y.Z.
#
# Meta targets:
# - "all" is the default target, it runs "lint" and "test"
#
VERSION?=

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GO_FILES=$(shell find . -type f -name '*.go')
GOPATH=$(shell go env GOPATH)
GOBIN_DIR=$(GOPATH)/bin
GOLANGCI_LINT_VERSION?=v2.12.2
GOLANGCI_LINT=$(GOBIN_DIR)/golangci-lint
PROTOC_GEN_GO_VERSION?=v1.36.11
PROTOC_GEN_GO_GRPC_VERSION?=v1.6.2
PROTOC_BIN=protoc
PROTOC_DEST=$(GOBIN_DIR)/$(PROTOC_BIN)

.PHONY: all all-tests ci clean depend install-hooks lint lint-docs lint-filesize lint-legacy-middleware lint-namescope lint-toolchain test test-race test-release integration-test integration-test-fast generated-code-quality openapi-contract build-loom build-loom-cached loom-local loom-remote loom-status release release-preflight
.NOTPARALLEL: release

# Only list test and build dependencies
# Standard dependencies are installed via go get
DEPEND=\
	google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION) \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

all: lint test integration-test

all-tests: lint test integration-test

ci: depend all

# Install protoc
PROTOC_VERSION=25.0
UNZIP=unzip
ifeq ($(GOOS),linux)
	PROTOC=protoc-$(PROTOC_VERSION)-linux-x86_64
	PROTOC_EXEC=$(PROTOC)/bin/protoc
endif
ifeq ($(GOOS),darwin)
	ifeq ($(GOARCH),arm64)
		PROTOC=protoc-$(PROTOC_VERSION)-osx-aarch_64
		PROTOC_EXEC=$(PROTOC)/bin/protoc
	else
		PROTOC=protoc-$(PROTOC_VERSION)-osx-x86_64
		PROTOC_EXEC=$(PROTOC)/bin/protoc
	endif
endif
ifeq ($(GOOS),windows)
	PROTOC=protoc-$(PROTOC_VERSION)-win32
	PROTOC_EXEC="$(PROTOC)\bin\protoc.exe"
	PROTOC_BIN=protoc.exe
	GOPATH:=$(subst \,/,$(GOPATH))
endif

depend:
	@echo INSTALLING DEPENDENCIES...
	@mkdir -p "$(GOBIN_DIR)"
	@go mod download
	@for package in $(DEPEND); do GOBIN="$(GOBIN_DIR)" go install $$package; done
	@GOBIN="$(GOBIN_DIR)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@$(GOLANGCI_LINT) version
	@go mod tidy -compat=1.17
	@echo INSTALLING PROTOC...
	@rm -rf "$(PROTOC)"
	@mkdir -p "$(PROTOC)"
	@cd $(PROTOC); \
	curl -O -L https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC).zip; \
	$(UNZIP) $(PROTOC).zip
	@rm -f "$(PROTOC_DEST)" && \
		cp $(PROTOC_EXEC) "$(PROTOC_DEST)" && \
		chmod 0755 "$(PROTOC_DEST)" && \
		rm -rf $(PROTOC) && \
		"$(PROTOC_DEST)" --version

install-hooks:
	git config core.hooksPath .githooks
	chmod 0755 .githooks/pre-push
	@echo "Configured git hooks to use .githooks"

lint:
ifneq ($(GOOS),windows)
	@bash ./scripts/lint_filesize.sh || (echo "^ - file size lint errors!" && echo && exit 1)
	@bash ./scripts/lint_legacy_middleware.sh || (echo "^ - legacy middleware lint errors!" && echo && exit 1)
	@bash ./scripts/lint_name_scope.sh || (echo "^ - name-scope lint errors!" && echo && exit 1)
	@bash ./scripts/lint_toolchain.sh || (echo "^ - toolchain lint errors!" && echo && exit 1)
	@go run ./scripts/docscheck || (echo "^ - documentation lint errors!" && echo && exit 1)
	@$(GOLANGCI_LINT) run ./... || (echo "^ - lint errors!" && echo && exit 1)
else
	@echo "SKIPPED: lint does not run on Windows"
endif

lint-filesize:
	@bash ./scripts/lint_filesize.sh

lint-legacy-middleware:
	@bash ./scripts/lint_legacy_middleware.sh

lint-namescope:
	@bash ./scripts/lint_name_scope.sh

lint-toolchain:
	@bash ./scripts/lint_toolchain.sh

lint-docs:
	@go run ./scripts/docscheck

test:
ifneq ($(GOOS),windows)
	PATH="$(GOBIN_DIR):$$PATH" go test ./... --coverprofile=cover.out
else
	go test ./... --coverprofile=cover.out
endif

test-release:
ifneq ($(GOOS),windows)
	PATH="$(GOBIN_DIR):$$PATH" go test -count=1 ./...
else
	go test -count=1 ./...
endif

# Race + shuffled-order guard for the unit suite. Shuffling catches
# order-coupled tests (the failing seed is printed for reproduction) and the
# race detector catches data races the plain run cannot.
test-race:
ifneq ($(GOOS),windows)
	PATH="$(GOBIN_DIR):$$PATH" go test -race -shuffle=on -count=1 ./...
else
	go test -race -shuffle=on -count=1 ./...
endif

integration-test: build-loom
ifneq ($(GOOS),windows)
	cd jsonrpc/integration_tests && PATH="$(GOBIN_DIR):$$PATH" go test -count=1 -timeout 10m ./...
	cd http/integration_tests && PATH="$(GOBIN_DIR):$$PATH" go test -count=1 -timeout 10m ./...
else
	@echo "SKIPPED: integration-test does not run on Windows (no Windows integration coverage)"
endif

# integration-test-fast is the iteration loop for codegen work. Differences
# from `integration-test`:
#
#   1. Uses `build-loom-cached` so the loom binary is only rebuilt when cmd/loom
#      or the transport codegens actually changed. On no-op re-runs the build
#      step is skipped entirely.
#   2. Respects the SERVICE variable so a single fixture is exercised:
#
#          make integration-test-fast SERVICE=ticktock
#
#      Without SERVICE the target defaults to running just the ticktock
#      fixtures (the fastest useful coverage surface). Override with
#      SERVICE=... to pick a specific fixture, or RUN=... to pass a custom
#      `-run` regex to `go test`.
#
# This target is intentionally NOT wired into `all` or CI — it is an explicit
# developer shortcut. `integration-test` remains the canonical target.
SERVICE?=ticktock
RUN?=.
integration-test-fast: build-loom-cached
ifneq ($(GOOS),windows)
	cd jsonrpc/integration_tests && PATH="$(GOBIN_DIR):$$PATH" go test -count=1 -timeout 5m -run '$(RUN)' ./fixtures/$(SERVICE)/...
	cd http/integration_tests && PATH="$(GOBIN_DIR):$$PATH" go test -count=1 -timeout 5m -run '$(RUN)' ./fixtures/$(SERVICE)/...
endif

generated-code-quality: build-loom-cached
ifneq ($(GOOS),windows)
	GOLANGCI_LINT="$(GOLANGCI_LINT)" LOOM_BIN="$(GOBIN_DIR)/loom" bash ./scripts/generated_code_quality.sh
endif

openapi-contract:
ifneq ($(GOOS),windows)
	PATH="$(GOBIN_DIR):$$PATH" LOOM_OPENAPI_CONTRACT=1 go test -count=1 -run 'Test(RenderedSpecsPassContractLint|RepresentativeSpecsPassRedoclyLintAndConsumerSmoke)$$' ./http/codegen/openapi/v3
endif

# Remove gitignored artifacts that integration-test runs leave behind
# (per-run loom build dirs and server logs inside the integration trees).
clean:
	find jsonrpc/integration_tests http/integration_tests -type d -name 'loom[0-9]*' -prune -exec rm -rf {} + 2>/dev/null || true
	find jsonrpc/integration_tests http/integration_tests -type f -name 'server-*.log' -delete 2>/dev/null || true

loom-local:
	bash ./scripts/loom_source_mode.sh local

loom-remote:
	bash ./scripts/loom_source_mode.sh remote

loom-status:
	bash ./scripts/loom_source_mode.sh status

# Needed for CI to run integration tests
build-loom:
	cd cmd/loom && GOBIN="$(GOBIN_DIR)" go install .

# Cached build for fast dev iteration. Rebuild only when the loom CLI source
# or codegen tree has actually changed, using the binary's mtime as the
# cache key. The comparison scans cmd/loom and every */codegen directory
# because those are the trees whose edits change the emitted output.
build-loom-cached: $(GOBIN_DIR)/loom

CODEGEN_SOURCES := $(shell find cmd/loom codegen http/codegen grpc/codegen jsonrpc/codegen -name '*.go' -not -path '*/testdata/*' 2>/dev/null)
$(GOBIN_DIR)/loom: $(CODEGEN_SOURCES)
	@echo "rebuilding loom (codegen/cli source changed)"
	cd cmd/loom && GOBIN="$(GOBIN_DIR)" go install .

release-preflight: lint test-release integration-test openapi-contract generated-code-quality

release:
	@go run ./internal/cmd/release --version "$(VERSION)"
