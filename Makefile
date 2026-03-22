#! /usr/bin/make
#
# Makefile for Loom
#
# Targets:
# - "depend" retrieves the Go packages needed to run the linter and tests
# - "lint" runs the linter
# - "test" runs the tests
# - "release" creates a new release commit, tags the commit and pushes the tag to GitHub.
#
# Meta targets:
# - "all" is the default target, it runs "lint" and "test"
#
MAJOR=1
MINOR=0
BUILD=0

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GO_FILES=$(shell find . -type f -name '*.go')
GOPATH=$(shell go env GOPATH)
GOBIN_DIR=$(GOPATH)/bin
GOLANGCI_LINT_VERSION?=v2.11.3
GOLANGCI_LINT=$(GOBIN_DIR)/golangci-lint
PROTOC_BIN=protoc
PROTOC_DEST=$(GOBIN_DIR)/$(PROTOC_BIN)

.PHONY: all all-tests ci depend lint test test-release integration-test build-loom loom-local loom-remote loom-status goa-local goa-remote goa-status release release-preflight release-loom
.NOTPARALLEL: release release-loom

# Only list test and build dependencies
# Standard dependencies are installed via go get
DEPEND=\
	google.golang.org/protobuf/cmd/protoc-gen-go@latest \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 

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
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN_DIR) $(GOLANGCI_LINT_VERSION)
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

lint:
ifneq ($(GOOS),windows)
	@$(GOLANGCI_LINT) run ./... || (echo "^ - lint errors!" && echo && exit 1)
endif

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

integration-test: build-loom
ifneq ($(GOOS),windows)
	cd jsonrpc/integration_tests && PATH="$(GOBIN_DIR):$$PATH" go test -count=1 -timeout 10m ./...
	cd http/integration_tests && PATH="$(GOBIN_DIR):$$PATH" go test -count=1 -timeout 10m ./...
endif

loom-local:
	bash ./scripts/goa_source_mode.sh local

loom-remote:
	bash ./scripts/goa_source_mode.sh remote

loom-status:
	bash ./scripts/goa_source_mode.sh status

goa-local: loom-local

goa-remote: loom-remote

goa-status: loom-status

# Needed for CI to run integration tests
build-loom:
	cd cmd/loom && GOBIN="$(GOBIN_DIR)" go install .

release-preflight: lint test-release integration-test

release: release-loom
	@echo "Release v$(MAJOR).$(MINOR).$(BUILD) complete"

release-loom:
	# First make sure all is clean
	@status="$$(git status --porcelain)"; \
	if [ -n "$$status" ]; then \
		echo "error: loom repo has uncommitted changes:"; \
		echo "$$status"; \
		exit 1; \
	fi
	go mod tidy 
	# Bump version number, commit and push
	sed 's/Major = .*/Major = $(MAJOR)/' pkg/version.go > _tmp && mv _tmp pkg/version.go
	sed 's/Minor = .*/Minor = $(MINOR)/' pkg/version.go > _tmp && mv _tmp pkg/version.go
	sed 's/Build = .*/Build = $(BUILD)/' pkg/version.go > _tmp && mv _tmp pkg/version.go
	sed 's|github.com/CaliLuke/loom@v.*tab=doc|github.com/CaliLuke/loom@v$(MAJOR).$(MINOR).$(BUILD)/dsl?tab=doc|' README.md > _tmp && mv _tmp README.md
	$(MAKE) release-preflight
	git add .
	git commit -m "Release v$(MAJOR).$(MINOR).$(BUILD)"
	git tag v$(MAJOR).$(MINOR).$(BUILD)
	cd cmd/loom && go install .
	git push origin main
	git push origin v$(MAJOR).$(MINOR).$(BUILD)
	# Wait for Go proxy to update
	sleep 10
