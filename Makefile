NAME  ?= node
CTL   ?= wmctl
REPO  ?= ghcr.io/webmeshproj
IMAGE ?= $(REPO)/$(NAME):latest
GOBGP_IMAGE ?= ghcr.io/webmeshproj/gobgp:latest

ARCH ?= $(shell go env GOARCH)
OS   ?= $(shell go env GOOS)

VERSION_PKG := github.com/webmeshproj/$(NAME)/pkg/version
VERSION     := $(shell git describe --tags --always --dirty)
COMMIT      := $(shell git rev-parse HEAD)
DATE        := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     ?= -s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildDate=$(DATE)

build: fmt vet generate ## Build node binary.
	CGO_ENABLED=0 go build \
		-tags netgo \
		-ldflags "$(LDFLAGS)" \
		-o dist/$(NAME)_$(OS)_$(ARCH) \
		cmd/$(NAME)/main.go

build-ctl: fmt vet ## Build wmctl binary.
	CGO_ENABLED=0 go build \
		-tags netgo \
		-ldflags "$(LDFLAGS)" \
		-o dist/$(CTL)_$(OS)_$(ARCH) \
		cmd/$(CTL)/main.go

PLATFORMS ?= linux/arm64 linux/amd64 linux/s390x linux/ppc64le
DIST      := $(CURDIR)/dist

tidy:
	go mod tidy

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

.PHONY: dist
dist: fmt vet generate ## Build node binaries for all platforms.
	go install github.com/mitchellh/gox@latest
	CGO_ENABLED=0 gox \
		-tags netgo \
		-ldflags "$(LDFLAGS)" \
		-osarch="$(PLATFORMS)" \
		-output="$(DIST)/$(NAME)_{{.OS}}_{{.Arch}}" \
		github.com/webmeshproj/$(NAME)/cmd/$(NAME)

dist-ctl: fmt vet ## Build wmctl binaries for all platforms.
	go install github.com/mitchellh/gox@latest
	CGO_ENABLED=0 gox \
		-tags netgo \
		-ldflags "$(LDFLAGS)" \
		-osarch="$(PLATFORMS)" \
		-output="$(DIST)/$(CTL)_{{.OS}}_{{.Arch}}" \
		github.com/webmeshproj/$(NAME)/cmd/$(CTL)

DOCKER ?= docker

docker-build: build ## Build the node docker image
	IMAGE=$(IMAGE) docker-compose build

docker-push: docker-build ## Push the node docker image
	IMAGE=$(IMAGE) docker-compose push

docker-build-gobgp: ## Build docker image with gobgp.
	$(DOCKER) build . \
		-t $(GOBGP_IMAGE) \
		-f Dockerfile.gobgp

docker-push-gobgp: ## Push docker image with gobgp.
	$(DOCKER) push $(GOBGP_IMAGE)

compose-up: ## Run docker-compose stack.
	IMAGE=$(IMAGE) docker-compose up

pull-db:
	docker-compose cp bootstrap-node-1:/data/webmesh.sqlite ./webmesh.sqlite

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

SQLC_CONFIG := pkg/meshdb/models/sql/sqlc.yaml
generate: ## Generate SQL code.
	go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
	sqlc -f $(SQLC_CONFIG) generate

install-ctl:
	go install github.com/webmeshproj/$(NAME)/cmd/$(CTL)