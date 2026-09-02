SHELL := /bin/bash

GO ?= go
GOFMT ?= gofmt
VERSION ?= dev
DIST_DIR ?= dist
MODULE := github.com/amxv/fidelius
LDFLAGS := -s -w -X $(MODULE)/internal/buildinfo.Version=$(VERSION)

.PHONY: help fmt test vet site-check site-build app-check check build build-universal install-local clean release-tag

help:
	@echo "fidelius command runner"
	@echo ""
	@echo "Targets:"
	@echo "  make check          - Go tests/vet + Astro check + native app compile"
	@echo "  make build          - build fidelius + Fidelius.app for this Mac"
	@echo "  make build-universal - build universal macOS release artifacts"
	@echo "  make install-local  - symlink the local build into ~/.local/bin"
	@echo "  make site-build     - build the Astro landing page"
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
	@tmp="$$(mktemp -d)"; trap 'rm -rf "$$tmp"' EXIT; \
	FIDELIUS_VERSION=0.0.0 apps/macos/build.sh "$$tmp/Fidelius.app" native; \
	codesign --verify --deep --strict "$$tmp/Fidelius.app"

check: fmt test vet site-check app-check

build:
	@mkdir -p $(DIST_DIR)
	@rm -f $(DIST_DIR)/fidelius
	@$(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/fidelius ./cmd/fidelius
	@FIDELIUS_VERSION=$(if $(filter dev,$(VERSION)),0.0.0,$(VERSION)) apps/macos/build.sh $(DIST_DIR)/Fidelius.app native

build-universal:
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)/.arch
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/.arch/fidelius-arm64 ./cmd/fidelius
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/.arch/fidelius-amd64 ./cmd/fidelius
	@lipo -create $(DIST_DIR)/.arch/fidelius-arm64 $(DIST_DIR)/.arch/fidelius-amd64 -output $(DIST_DIR)/fidelius
	@FIDELIUS_VERSION=$(if $(filter dev,$(VERSION)),0.0.0,$(VERSION)) apps/macos/build.sh $(DIST_DIR)/Fidelius.app universal
	@rm -rf $(DIST_DIR)/.arch

install-local: build
	@mkdir -p $$HOME/.local/bin
	@ln -sfn "$(CURDIR)/$(DIST_DIR)/fidelius" $$HOME/.local/bin/fidelius
	@ln -sfn "$(CURDIR)/$(DIST_DIR)/Fidelius.app" $$HOME/.local/bin/Fidelius.app
	@echo "Linked fidelius into $$HOME/.local/bin"

clean:
	@rm -rf $(DIST_DIR)

release-tag:
	@test -n "$(VERSION)" && test "$(VERSION)" != "dev" || (echo "Usage: make release-tag VERSION=x.y.z" && exit 1)
	@git tag "v$(VERSION)"
	@git push origin "v$(VERSION)"
