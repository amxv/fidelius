SHELL := /bin/bash

GO ?= go
GOFMT ?= gofmt
VERSION ?= dev
DIST_DIR ?= dist
MODULE := github.com/amxv/fidelius
LDFLAGS := -s -w -X $(MODULE)/internal/buildinfo.Version=$(VERSION)
UNAME_S := $(shell uname -s)

.PHONY: help fmt test vet site-check site-build app-check check build build-universal build-linux install-local clean release-tag

help:
	@echo "fidelius command runner"
	@echo ""
	@echo "Targets:"
	@echo "  make check           - Go tests/vet + Astro check + native macOS app check when available"
	@echo "  make build           - build Fidelius for this machine"
	@echo "  make build-universal - build universal macOS release artifacts"
	@echo "  make build-linux     - build Linux amd64 + arm64 release binaries"
	@echo "  make install-local   - symlink the local build into ~/.local/bin"
	@echo "  make site-build      - build the Astro landing page"
	@echo "  make release-tag VERSION=x.y.z - push a GitHub release tag"

fmt:
	@$(GOFMT) -w $$(find . -type f -name '*.go' -not -path './dist/*')

test:
	@$(GO) test ./...

vet:
	@$(GO) vet ./...

site-check:
	@cd docs && bun run check

site-build:
	@cd docs && bun run build

app-check:
ifeq ($(UNAME_S),Darwin)
	@tmp="$$(mktemp -d)"; trap 'rm -rf "$$tmp"' EXIT; \
	FIDELIUS_VERSION=0.0.0 apps/macos/build.sh "$$tmp/Fidelius.app" native; \
	codesign --verify --deep --strict "$$tmp/Fidelius.app"
else
	@echo "Skipping AppKit check on $(UNAME_S)"
endif

check: fmt test vet site-check app-check

build:
	@mkdir -p $(DIST_DIR)
	@rm -f $(DIST_DIR)/fidelius
	@$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/fidelius ./cmd/fidelius
ifeq ($(UNAME_S),Darwin)
	@FIDELIUS_VERSION=$(if $(filter dev,$(VERSION)),0.0.0,$(VERSION)) apps/macos/build.sh $(DIST_DIR)/Fidelius.app native
endif

build-universal:
	@test "$(UNAME_S)" = "Darwin" || (echo "build-universal requires macOS" >&2; exit 1)
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)/.arch
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/.arch/fidelius-arm64 ./cmd/fidelius
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/.arch/fidelius-amd64 ./cmd/fidelius
	@lipo -create $(DIST_DIR)/.arch/fidelius-arm64 $(DIST_DIR)/.arch/fidelius-amd64 -output $(DIST_DIR)/fidelius
	@FIDELIUS_VERSION=$(if $(filter dev,$(VERSION)),0.0.0,$(VERSION)) apps/macos/build.sh $(DIST_DIR)/Fidelius.app universal
	@rm -rf $(DIST_DIR)/.arch

build-linux:
	@mkdir -p $(DIST_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/fidelius-linux-amd64 ./cmd/fidelius
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/fidelius-linux-arm64 ./cmd/fidelius

install-local: build
	@mkdir -p $$HOME/.local/bin
	@ln -sfn "$(CURDIR)/$(DIST_DIR)/fidelius" $$HOME/.local/bin/fidelius
ifeq ($(UNAME_S),Darwin)
	@ln -sfn "$(CURDIR)/$(DIST_DIR)/Fidelius.app" $$HOME/.local/bin/Fidelius.app
endif
	@echo "Linked fidelius into $$HOME/.local/bin"

clean:
	@rm -rf $(DIST_DIR)

release-tag:
	@test -n "$(VERSION)" && test "$(VERSION)" != "dev" || (echo "Usage: make release-tag VERSION=x.y.z" && exit 1)
	@git tag "v$(VERSION)"
	@git push origin "v$(VERSION)"
