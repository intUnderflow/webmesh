SHELL := /bin/bash

NAME  ?= node
CTL   ?= wmctl
REPO  ?= ghcr.io/webmeshproj
IMAGE ?= $(REPO)/$(NAME):latest
DISTROLESS_IMAGE ?= $(REPO)/$(NAME)-distroless:latest

GO    ?= go
ARCH  ?= $(shell $(GO) env GOARCH)
OS    ?= $(shell $(GO) env GOOS)

GOPATH ?= $(shell $(GO) env GOPATH)
ifeq ($(OS),Windows_NT)
	OS := windows
# Double escape the backslashes for Windows paths.
	GOBIN  := $(subst \,\\,$(GOPATH))\\bin\\
	RICHGO := $(subst \,\\,$(GOBIN))richgo.exe
else
	GOBIN := $(GOPATH)/bin
	RICHGO := $(GOBIN)/richgo
endif

default: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

GORELEASER ?= $(GO) run github.com/goreleaser/goreleaser@latest
BUILD_ARGS ?= --snapshot --clean --parallelism=$(PARALLEL)
PARALLEL   ?= $(shell nproc)

build: ## Build node and wmctl binaries for the current architecture.
	$(GORELEASER) build --single-target --id node --id wmctl $(BUILD_ARGS)

# build-wasm: fmt vet ## Build node wasm binary for the current architecture.
# 	$(GORELEASER) build $(BUILD_ARGS) --id node-wasm --parallelism=$(PARALLEL)

.PHONY: dist
dist: ## Build distribution binaries and packages for all platforms.
	$(GORELEASER) release --skip=sign $(BUILD_ARGS)

DOCKER ?= docker

docker-build: docker-build-bin ## Build the node docker image for the current architecture.
	$(DOCKER) build \
		-f Dockerfile \
		--build-arg PREFIX=node-docker-linux \
		--build-arg TARGETOS=linux \
		--build-arg TARGETARCH=$(ARCH) \
		-t $(IMAGE) .

docker-build-distroless: docker-build-bin ## Build the distroless node docker image for the current architecture.
	$(DOCKER) build \
		-f Dockerfile.distroless \
		--build-arg PREFIX=node-docker-linux \
		--build-arg TARGETOS=linux \
		--build-arg TARGETARCH=$(ARCH) \
		-t $(DISTROLESS_IMAGE) .

docker-build-bin:
	$(GORELEASER) build $(BUILD_ARGS) --id node-docker-linux --single-target

docker-push: docker-build ## Push the node docker image
	$(DOCKER) push $(IMAGE)

docker-push-distroless: docker-build-distroless ## Push the distroless node docker image
	$(DOCKER) push $(DISTROLESS_IMAGE)

##@ Testing

CI ?= false
ifeq ($(CI),true)
# We are running in CI, so skip fmt and vet on Windows and macOS.
ifeq ($(OS),linux)
CI_TARGETS := mod-download fmt vet test
else
CI_TARGETS := mod-download test
endif
else
# We are running locally, so we can run all tests.
ifeq ($(OS),windows)
# Don't fmt on Windows, it screws up the line endings.
CI_TARGETS := mod-download vet lint test
else
CI_TARGETS :=  mod-download fmt vet lint test
endif
endif

ci-test: ## Run all CI tests.
	set -eo pipefail ; $(MAKE) $(CI_TARGETS) | xargs -IL date +"[%Y-%m-%d %H:%M:%S]: L"

RICHGO_INSTALLED := $(shell test -f $(RICHGO) && echo true || echo false)
COVERAGE_FILE    := coverage.out
TEST_PARALLEL    ?= $(shell nproc 2>/dev/null || echo 8)
TEST_ARGS        := -v -cover -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic -parallel=$(TEST_PARALLEL)

test: ## Run unit tests.
ifeq ($(RICHGO_INSTALLED),false)
	$(GO) install github.com/kyoh86/richgo@latest
endif
	$(RICHGO) test $(TEST_ARGS) ./...
	$(GO) tool cover -func=$(COVERAGE_FILE)

LINT_TIMEOUT := 10m
lint: ## Run linters.
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout=$(LINT_TIMEOUT)

mod-download:
	$(GO) mod download -x

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	$(GO) vet ./...

##@ Misc

generate: ## Run go generate against code.
	$(GO) generate ./...

clean: ## Clean up build and development artifacts.
	rm -rf dist/ $(COVERAGE_FILE)

build-ctl:
	$(GORELEASER) build --single-target $(BUILD_ARGS) --id $(CTL) -o dist/$(CTL)

install-ctl: build-ctl
	install -m 755 dist/$(CTL) $(shell go env GOPATH)/bin/$(CTL)

latest-api: ## Used for development and forces a pull of the API off the main branch.
	GOPRIVATE=github.com/webmeshproj $(GO) get -u github.com/webmeshproj/api@main