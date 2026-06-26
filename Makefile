NAME ?= clashtars
VERSION ?= 0.1.0
RELEASE ?= 1

BUILD_DIR := build
GO ?= go
GO_CACHE := $(abspath $(BUILD_DIR)/go-cache)
BUILDVCS ?= auto
RPM_TOPDIR := $(abspath $(BUILD_DIR)/rpmbuild)
SOURCE_TAR := $(RPM_TOPDIR)/SOURCES/$(NAME)-$(VERSION).tar.gz
SPEC_FILE := packaging/$(NAME).spec

DEB_TOPDIR := $(abspath $(BUILD_DIR)/debbuild)
DEB_ROOT := $(DEB_TOPDIR)/$(NAME)_$(VERSION)-$(RELEASE)_amd64
DEB_FILE := $(BUILD_DIR)/$(NAME)_$(VERSION)-$(RELEASE)_amd64.deb

.PHONY: all build build-dev test stage-assets rpm deb source clean FORCE

all: build

stage-assets:
	./scripts/stage-assets.sh

build: stage-assets
	GOCACHE="$(GO_CACHE)" $(GO) build -trimpath -buildvcs=$(BUILDVCS) -o "$(BUILD_DIR)/clashtars" ./cmd/clashtars

build-dev:
	GOCACHE="$(GO_CACHE)" $(GO) build -trimpath -buildvcs=$(BUILDVCS) -o "$(BUILD_DIR)/clashtars-dev" ./cmd/clashtars

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

deb:
	@test -f "$(BUILD_DIR)/clashtars" || { echo "$(BUILD_DIR)/clashtars not found; run 'make build' first" >&2; exit 1; }
	rm -rf "$(DEB_TOPDIR)"
	mkdir -p "$(DEB_ROOT)/DEBIAN"
	mkdir -p "$(DEB_ROOT)/usr/bin"
	mkdir -p "$(DEB_ROOT)/lib/systemd/system"
	mkdir -p "$(DEB_ROOT)/var/lib/clashtars/ui"
	install -m 0755 "$(BUILD_DIR)/clashtars" "$(DEB_ROOT)/usr/bin/clashtars"
	install -m 0644 packaging/clashtars.service "$(DEB_ROOT)/lib/systemd/system/clashtars.service"
	install -m 0640 configs/clash.conf.example "$(DEB_ROOT)/var/lib/clashtars/clash.conf"
	install -m 0640 configs/template.yaml.example "$(DEB_ROOT)/var/lib/clashtars/template.yaml"
	sed -e 's/@VERSION@/$(VERSION)-$(RELEASE)/' packaging/debian/control.in > "$(DEB_ROOT)/DEBIAN/control"
	install -m 0755 packaging/debian/postinst "$(DEB_ROOT)/DEBIAN/postinst"
	install -m 0755 packaging/debian/prerm "$(DEB_ROOT)/DEBIAN/prerm"
	install -m 0755 packaging/debian/postrm "$(DEB_ROOT)/DEBIAN/postrm"
	install -m 0644 packaging/debian/conffiles "$(DEB_ROOT)/DEBIAN/conffiles"
	dpkg-deb --build --root-owner-group "$(DEB_ROOT)" "$(DEB_FILE)"
	@printf '\nDEB package:\n%s\n' "$(DEB_FILE)"

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
		--exclude='./template.yaml' \
		--exclude='./config.yaml' \
		--exclude='./subscription.yaml' \
		--exclude='./converted.yaml' \
		--exclude='./providers' \
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
