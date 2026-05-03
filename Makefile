# Makefile for Bellum Linux Installer Go
# Builds both binaries and creates the release tarball

.PHONY: all build clean release help

VERSION := 2.0.0

# Go configuration
GO := go
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0

# Directories
INSTALLER_DIR := bellum-installer
UNINSTALLER_DIR := bellum-uninstaller
PACKAGES_DIR := packages

# Output files
INSTALLER_BIN := installer
UNINSTALLER_BIN := uninstaller
RELEASE_TARBALL := bellum-installer-linux-amd64-$(VERSION).tar.gz
RELEASE_NAME := $(basename $(basename $(RELEASE_TARBALL)))
RELEASE_DIR := $(RELEASE_NAME)

# Default target
all: build

# Build both binaries
build: $(INSTALLER_BIN) $(UNINSTALLER_BIN)
	@echo "[BUILD] Done! Both binaries built successfully."

# Build bellum-installer
$(INSTALLER_BIN):
	@echo "[BUILD] Building installer..."
	$(GO) mod tidy
	$(GO) build -o $(INSTALLER_BIN) ./$(INSTALLER_DIR)

# Build bellum-uninstaller
$(UNINSTALLER_BIN):
	@echo "[BUILD] Building uninstaller..."
	$(GO) mod tidy
	$(GO) build -o $(UNINSTALLER_BIN) ./$(UNINSTALLER_DIR)

# Create release tarball
release: $(INSTALLER_BIN) $(UNINSTALLER_BIN)
	@echo "[BUILD] Creating release tarball..."
	@mkdir -p $(RELEASE_DIR)
	@cp $(INSTALLER_BIN) $(RELEASE_DIR)/
	@cp $(UNINSTALLER_BIN) $(RELEASE_DIR)/
	@cp -r $(PACKAGES_DIR) $(RELEASE_DIR)/
	@tar -czvf $(RELEASE_TARBALL) -C $(dir $(RELEASE_DIR)) $(notdir $(RELEASE_DIR))
	@rm -rf $(RELEASE_DIR)
	@echo "[BUILD] Done! Release tarball created: $(RELEASE_TARBALL)"
	@echo "[BUILD] Binary sizes:"
	@ls -lh $(RELEASE_TARBALL)

# Clean build artifacts
clean:
	@echo "[CLEAN] Removing build artifacts..."
	rm -f $(INSTALLER_BIN)
	rm -f $(UNINSTALLER_BIN)
	rm -f $(RELEASE_TARBALL)
	rm -rf $(RELEASE_DIR)
	@echo "[CLEAN] Done!"

# Help target
help:
	@echo "Bellum Linux Installer Go - Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  build   - Build both installer and uninstaller binaries"
	@echo "  release - Build binaries and create release tarball"
	@echo "  clean   - Remove all build artifacts"
	@echo "  help    - Show this help message"
