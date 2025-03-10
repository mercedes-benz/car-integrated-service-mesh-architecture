# SPDX-License-Identifier: Apache-2.0

################################################################################
# Variables                                                                    #
################################################################################

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
   TARGET_OS_LOCAL = linux
else ifeq ($(LOCAL_OS),Darwin)
   TARGET_OS_LOCAL = darwin
else
   TARGET_OS_LOCAL = windows
endif
export GOOS ?= $(TARGET_OS_LOCAL)

LOCAL_ARCH := $(shell uname -m)
ifeq ($(LOCAL_ARCH),x86_64)
	TARGET_ARCH_LOCAL=amd64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),armv8)
	TARGET_ARCH_LOCAL=arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 4),armv)
	TARGET_ARCH_LOCAL=arm
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),arm64)
	TARGET_ARCH_LOCAL=arm64
else
	TARGET_ARCH_LOCAL=amd64
endif
export GOARCH ?= $(TARGET_ARCH_LOCAL)

GIT_TAG		:= $(shell git describe --exact-match 2> /dev/null)
GIT_COMMIT  := $(shell git rev-list -1 HEAD)

ifdef GIT_TAG
	CARISMA_VERSION := $(GIT_TAG)
else
	CARISMA_VERSION := DEVELOPMENT
endif

BINARIES    ?= carisma-version carisma-status-manager carisma-control-plane carisma-orchestrator

OUT_DIR := ./dist

################################################################################
# Go build details                                                             #
################################################################################

BASE_PACKAGE_NAME := github.com/mercedes-benz/car-integrated-service-mesh-architecture

DEFAULT_LDFLAGS:=-X $(BASE_PACKAGE_NAME)/pkg/version.commit=$(GIT_COMMIT) \
  -X $(BASE_PACKAGE_NAME)/pkg/version.version=$(CARISMA_VERSION)

ifeq ($(origin DEBUG), undefined)
  BUILDTYPE_DIR:=release
  LDFLAGS:="$(DEFAULT_LDFLAGS) -s -w"
else ifeq ($(DEBUG),0)
  BUILDTYPE_DIR:=release
  LDFLAGS:="$(DEFAULT_LDFLAGS) -s -w"
else
  BUILDTYPE_DIR:=debug
  GCFLAGS:=-gcflags="all=-N -l"
  LDFLAGS:="$(DEFAULT_LDFLAGS)"
  $(info Build with debugger information)
endif

CARISMA_OUT_DIR := $(OUT_DIR)/$(GOOS)_$(GOARCH)/$(BUILDTYPE_DIR)

################################################################################
# Target: build                                                                #
################################################################################
.PHONY: build
CARISMA_BINS:=$(foreach ITEM,$(BINARIES),$(CARISMA_OUT_DIR)/$(ITEM)$(BINARY_EXT))
build: $(CARISMA_BINS)

# Generate builds for CARISMA binaries for the target
# Params:
# $(1): the binary name for the target
# $(2): the binary main directory
# $(3): the target os
# $(4): the target arch
# $(5): the output directory
define genBinariesForTarget
.PHONY: $(5)/$(1)
$(5)/$(1):
	GOOS=$(3) GOARCH=$(4) go build $(GCFLAGS) -ldflags=$(LDFLAGS) \
	-o $(5)/$(1) $(2)/;
endef

# Generate binary targets
$(foreach ITEM,$(BINARIES),$(eval $(call genBinariesForTarget,$(ITEM)$(BINARY_EXT),./cmd/$(ITEM),$(GOOS),$(GOARCH),$(CARISMA_OUT_DIR))))

################################################################################
# Target: release-arm                                                          #
################################################################################
.PHONY: release-arm
release-arm:
	$(MAKE) --no-print-directory build GOARCH=arm

################################################################################
# Target: release-arm64                                                        #
################################################################################
.PHONY: release-arm64
release-arm64:
	$(MAKE) --no-print-directory build GOARCH=arm64

################################################################################
# Target: release-amd64                                                        #
################################################################################
.PHONY: release-amd64
release-amd64:
	$(MAKE) --no-print-directory build GOARCH=amd64