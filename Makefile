.PHONY: build install clean test help

# Binary name
BINARY_NAME=wt

# Installation paths
INSTALL_PATH_SYSTEM=/usr/local/bin
INSTALL_PATH_USER=$(HOME)/bin

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME)
	@echo "✓ Build complete: ./$(BINARY_NAME)"

install: build ## Build and install the binary (tries system-wide, falls back to user)
	@echo "Installing $(BINARY_NAME)..."
	@if [ -w $(INSTALL_PATH_SYSTEM) ] || sudo -n true 2>/dev/null; then \
		echo "Installing to $(INSTALL_PATH_SYSTEM) (system-wide)..."; \
		sudo mv $(BINARY_NAME) $(INSTALL_PATH_SYSTEM)/$(BINARY_NAME); \
		sudo chmod +x $(INSTALL_PATH_SYSTEM)/$(BINARY_NAME); \
		echo "✓ Installed to $(INSTALL_PATH_SYSTEM)/$(BINARY_NAME)"; \
		echo ""; \
		echo "Run 'wt install' to set up shell integration"; \
	else \
		echo "No sudo access, installing to $(INSTALL_PATH_USER) (user-local)..."; \
		mkdir -p $(INSTALL_PATH_USER); \
		mv $(BINARY_NAME) $(INSTALL_PATH_USER)/$(BINARY_NAME); \
		chmod +x $(INSTALL_PATH_USER)/$(BINARY_NAME); \
		echo "✓ Installed to $(INSTALL_PATH_USER)/$(BINARY_NAME)"; \
		echo ""; \
		echo "Make sure $(INSTALL_PATH_USER) is in your PATH:"; \
		echo "  echo 'export PATH=\"\$$HOME/bin:\$$PATH\"' >> ~/.zshrc"; \
		echo "  source ~/.zshrc"; \
		echo ""; \
		echo "Then run 'wt install' to set up shell integration"; \
	fi

clean: ## Remove built binary
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "✓ Cleaned"

test: ## Run tests
	@echo "Running tests..."
	@go test ./...
	@echo "✓ Tests complete"

uninstall: ## Uninstall the binary
	@echo "Uninstalling $(BINARY_NAME)..."
	@if [ -f "$(INSTALL_PATH_SYSTEM)/$(BINARY_NAME)" ]; then \
		sudo rm -f $(INSTALL_PATH_SYSTEM)/$(BINARY_NAME); \
		echo "✓ Removed from $(INSTALL_PATH_SYSTEM)"; \
	elif [ -f "$(INSTALL_PATH_USER)/$(BINARY_NAME)" ]; then \
		rm -f $(INSTALL_PATH_USER)/$(BINARY_NAME); \
		echo "✓ Removed from $(INSTALL_PATH_USER)"; \
	else \
		echo "$(BINARY_NAME) not found in common installation paths"; \
	fi
	@echo ""
	@echo "Note: Shell function in ~/.zshrc will remain. Run 'wt install' to remove it manually if needed."

.DEFAULT_GOAL := help

