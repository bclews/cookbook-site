.PHONY: help install build-tool validate convert build serve clean test lint import clean-imports clean-recipes stats quick-build _auto-import

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[36m
GREEN := \033[32m
RED := \033[31m
RESET := \033[0m

# Number of parallel image downloads
PARALLEL ?= 10

help: ## Show this help message
	@echo "$(BLUE)Available targets:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-15s$(RESET) %s\n", $$1, $$2}'

install: ## Check that Hugo and Go are installed
	@echo "$(BLUE)Checking dependencies...$(RESET)"
	@command -v hugo >/dev/null 2>&1 || { echo "$(RED)Hugo not found. Install it from https://gohugo.io/installation/$(RESET)"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go not found. Install it from https://go.dev/dl/$(RESET)"; exit 1; }
	@echo "$(GREEN)Dependencies found$(RESET)"
	@hugo version
	@go version

build-tool: ## Build the recipe-tool binary
	@echo "$(BLUE)Building recipe-tool...$(RESET)"
	@CGO_ENABLED=0 go build -o recipe-site/recipe-tool ./recipe-site/cmd/recipe-tool

validate: build-tool ## Validate YAML recipe files
	@echo "$(BLUE)Validating recipe files...$(RESET)"
	@cd recipe-site && ./recipe-tool validate

convert: build-tool ## Convert YAML recipes to Hugo markdown
	@echo "$(BLUE)Converting recipes (parallel=$(PARALLEL))...$(RESET)"
	@cd recipe-site && ./recipe-tool convert --parallel $(PARALLEL)

import: build-tool ## Extract ZIP file from imports/ directory
	@echo "$(BLUE)Importing recipe ZIP file...$(RESET)"
	@cd recipe-site && ./recipe-tool import

clean-imports: build-tool ## Remove extracted files and ZIP from imports/
	@echo "$(BLUE)Cleaning imports directory...$(RESET)"
	@cd recipe-site && ./recipe-tool cleanup

clean-recipes: ## Remove all converted recipe markdown files (for full sync)
	@echo "$(BLUE)Cleaning existing recipes for full sync...$(RESET)"
	@find recipe-site/content/recipes -name '*.md' ! -name '_index.md' -delete 2>/dev/null || true
	@echo "$(GREEN)Cleaned recipe-site/content/recipes/$(RESET)"

# Internal helper: auto-import ZIP if present
_auto-import: build-tool
	@if [ -n "$$(find imports -maxdepth 1 -name '*.zip' 2>/dev/null | head -1)" ]; then \
		echo "$(BLUE)ZIP detected in imports/, importing...$(RESET)"; \
		cd recipe-site && ./recipe-tool import; \
		find recipe-site/content/recipes -name '*.md' ! -name '_index.md' -delete 2>/dev/null || true; \
	fi

build: _auto-import validate convert ## Build the static site into recipe-site/public/
	@echo "$(BLUE)Building site...$(RESET)"
	@cd recipe-site && hugo --minify
	@if [ -d "imports/.extracted" ]; then cd recipe-site && ./recipe-tool cleanup; fi
	@echo "$(GREEN)Site built in recipe-site/public/$(RESET)"

serve: build-tool ## Run development server
	@if [ -z "$$(find recipe-site/content/recipes -name '*.md' ! -name '_index.md' 2>/dev/null | head -1)" ]; then \
		echo "$(RED)No recipes found. Run 'make build' first.$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Starting development server...$(RESET)"
	@cd recipe-site && hugo server -D --bind 0.0.0.0

clean: ## Clean generated files (FULL=1 includes images)
	@echo "$(BLUE)Cleaning generated files...$(RESET)"
	@find recipe-site/content/recipes -name '*.md' ! -name '_index.md' -delete 2>/dev/null || true
	@rm -rf recipe-site/public/*
	@rm -f recipe-site/recipe-tool
ifdef FULL
	@rm -rf recipe-site/static/images/recipes/*
	@echo "$(GREEN)Cleaned content, public, and images$(RESET)"
else
	@echo "$(GREEN)Cleaned content/recipes/ and public/$(RESET)"
endif

test: ## Run Go unit tests
	@echo "$(BLUE)Running tests...$(RESET)"
	@CGO_ENABLED=0 go test ./recipe-site/... -v

lint: ## Run Go formatting check and vet
	@echo "$(BLUE)Checking formatting...$(RESET)"
	@test -z "$$(gofmt -l ./recipe-site/)" || { echo "$(RED)Files need formatting:$(RESET)"; gofmt -l ./recipe-site/; exit 1; }
	@echo "$(BLUE)Running go vet...$(RESET)"
	@go vet ./recipe-site/...
	@echo "$(GREEN)Lint passed$(RESET)"

stats: ## Show repository statistics
	@echo "$(BLUE)Repository Statistics:$(RESET)"
	@echo ""
	@echo "YAML Source Files:"
	@find . -path "./CookBook-Recipes-YAML/*.yml" -o -path "./data/recipes/*.yml" -o -path "./examples/recipes/*.yml" 2>/dev/null | wc -l | xargs echo "  "
	@echo ""
	@echo "Converted Markdown Files:"
	@find recipe-site/content/recipes -name "*.md" ! -name "_index.md" 2>/dev/null | wc -l | xargs echo "  "
	@echo ""
	@echo "Downloaded Images:"
	@find recipe-site/static/images/recipes -type f 2>/dev/null | wc -l | xargs echo "  "
	@echo ""
	@if [ -d recipe-site/public ]; then \
		echo "Built HTML Files:"; \
		find recipe-site/public -name "*.html" 2>/dev/null | wc -l | xargs echo "  "; \
		echo ""; \
		echo "Total Build Size:"; \
		du -sh recipe-site/public 2>/dev/null | cut -f1 | xargs echo "  "; \
	fi

# Quick rebuild without re-downloading images
quick-build: build-tool validate ## Quick rebuild (skip image downloads)
	@echo "$(BLUE)Quick rebuild (skipping image downloads)...$(RESET)"
	@cd recipe-site && ./recipe-tool convert --skip-images
	@cd recipe-site && hugo --minify
	@echo "$(GREEN)Quick build complete$(RESET)"

