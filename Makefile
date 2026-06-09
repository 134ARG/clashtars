NAME ?= clashtars
VERSION ?= 0.1.0
RELEASE ?= 1

BUILD_DIR := build
GO ?= go
GO_CACHE := $(abspath $(BUILD_DIR)/go-cache)
RPM_TOPDIR := $(abspath $(BUILD_DIR)/rpmbuild)
SOURCE_TAR := $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION).tar.gz
SPEC_FILE := packaging/$(NAME).spec

.PHONY: all build build-dev test stage-assets rpm source clean FORCE

all: build

stage-assets:
	./scripts/stage-assets.sh

build: stage-assets
	GOCACHE="$(GO_CACHE)" $(GO) build -trimpath -o "$(BUILD_DIR)/clashtars" ./cmd/clashtars

build-dev:
	GOCACHE="$(GO_CACHE)" $(GO) build -trimpath -o "$(BUILD_DIR)/clashtars-dev" ./cmd/clashtars

test:
	GOCACHE="$(GO_CACHE)" $(GO) test ./...

rpm: $(SOURCE_TAR)
	rpmbuild -bb \
		--define "_topdir $(RPM_TOPDIR)" \
		--define "_tmppath $(RPM_TOPDIR)/TMP" \
		--define "name $(NAME)" \
		--define "version $(VERSION)" \
		--define "release $(RELEASE)" \
		$(SPEC_FILE)
	@printf '\nRPM package(s):\n'
	@find "$(RPM_TOPDIR)/RPMS" -type f -name '*.rpm' -print

source: $(SOURCE_TAR)
	@printf '%s\n' "$(SOURCE_TAR)"

$(SOURCE_TAR): FORCE
	rm -rf "$(RPM_TOPDIR)"
	mkdir -p "$(RPM_TOPDIR)/SOURCES" "$(RPM_TOPDIR)/SPECS" \
		"$(RPM_TOPDIR)/BUILD" "$(RPM_TOPDIR)/BUILDROOT" "$(RPM_TOPDIR)/RPMS" "$(RPM_TOPDIR)/SRPMS" \
		"$(RPM_TOPDIR)/TMP"
	tar -czf "$(SOURCE_TAR)" \
		--transform 's,^\.,$(NAME)-$(VERSION),' \
		--exclude='./.git' \
		--exclude='./build' \
		--exclude='./clash.conf' \
		--exclude='./config.yaml' \
		--exclude='./subscription.yaml' \
		--exclude='./converted.yaml' \
		--exclude='./cache' \
		--exclude='./ui' \
		--exclude='./runtime' \
		--exclude='./internal/assets/mihomo/mihomo' \
		--exclude='./internal/assets/subconverter/*.tar.gz' \
		--exclude='./internal/assets/subconverter/*.tgz' \
		--exclude='./internal/assets/ui/*.tar.gz' \
		--exclude='./internal/assets/ui/*.tgz' \
		.

FORCE:

clean:
	rm -rf "$(BUILD_DIR)"
