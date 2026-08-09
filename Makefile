.PHONY: all build test lint fmt vet clean docker-build docker-up docker-down migrate

# ─── Variables ────────────────────────────────────────────────────────────
GO := go
NPM := npm
DOCKER := docker
DOCKER_COMPOSE := docker compose

CORE_DIR := core
HUB_DIR := hub
AGENT_DIR := agent

VERSION ?= 1.0.0-rc1
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X 'github.com/renos-project/renos/core/api.Version=$(VERSION)' \
           -X 'github.com/renos-project/renos/core/api.Commit=$(COMMIT)' \
           -X 'github.com/renos-project/renos/core/api.BuildTime=$(BUILD_TIME)'

# ─── Build ────────────────────────────────────────────────────────────────
all: build

build: build-core build-hub build-agent

build-core:
	cd $(CORE_DIR) && $(GO) build -ldflags "$(LDFLAGS)" -o ../bin/renos-core ./cmd/renos-core

build-hub:
	cd $(HUB_DIR) && $(NPM) run build

build-agent:
	cd $(AGENT_DIR) && cargo build --release

# ─── Development ──────────────────────────────────────────────────────────
dev-core:
	cd $(CORE_DIR) && $(GO) run ./cmd/renos-core

dev-hub:
	cd $(HUB_DIR) && $(NPM) run dev

# ─── Test ─────────────────────────────────────────────────────────────────
test: test-core test-hub test-agent

test-core:
	cd $(CORE_DIR) && $(GO) test -v -count=1 ./...

test-hub:
	cd $(HUB_DIR) && $(NPM) test

test-agent:
	cd $(AGENT_DIR) && cargo test

test-coverage:
	cd $(CORE_DIR) && $(GO) test -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report: core/coverage.out"

test-race:
	cd $(CORE_DIR) && $(GO) test -race -count=1 ./...

# ─── Lint ─────────────────────────────────────────────────────────────────
lint: lint-core lint-hub lint-agent

lint-core:
	cd $(CORE_DIR) && golangci-lint run

lint-hub:
	cd $(HUB_DIR) && $(NPM) run lint

lint-agent:
	cd $(AGENT_DIR) && cargo clippy -- -D warnings

# ─── Format ───────────────────────────────────────────────────────────────
fmt: fmt-core fmt-agent fmt-hub

fmt-core:
	cd $(CORE_DIR) && gofmt -s -w .

fmt-hub:
	cd $(HUB_DIR) && npx prettier --write .

fmt-agent:
	cd $(AGENT_DIR) && cargo fmt

# ─── Vet ──────────────────────────────────────────────────────────────────
vet:
	cd $(CORE_DIR) && $(GO) vet ./...

# ─── Docker ───────────────────────────────────────────────────────────────
docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

# ─── Database ─────────────────────────────────────────────────────────────
migrate:
	cd $(CORE_DIR) && $(GO) run ./cmd/renos-core --migrate-only

# ─── Clean ────────────────────────────────────────────────────────────────
clean:
	rm -rf bin/ $(CORE_DIR)/bin/ $(HUB_DIR)/.next/ $(AGENT_DIR)/target/
	cd $(HUB_DIR) && $(NPM) run clean 2>/dev/null || true

# ─── Dependencies ─────────────────────────────────────────────────────────
deps: deps-core deps-hub

deps-core:
	cd $(CORE_DIR) && $(GO) mod tidy

deps-hub:
	cd $(HUB_DIR) && $(NPM) install

deps-agent:
	cd $(AGENT_DIR) && cargo fetch

# ─── Security ─────────────────────────────────────────────────────────────
security:
	cd $(CORE_DIR) && govulncheck ./...

# ─── Quick checks (for pre-commit) ────────────────────────────────────────
check: fmt vet lint test

# ─── Help ─────────────────────────────────────────────────────────────────
help:
	@echo "Targets:"
	@echo "  all            - Build everything"
	@echo "  build          - Build all components"
	@echo "  test           - Run all tests"
	@echo "  lint           - Run all linters"
	@echo "  fmt            - Format all code"
	@echo "  vet            - Go vet"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-up      - Start services"
	@echo "  docker-down    - Stop services"
	@echo "  migrate        - Run database migrations"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Tidy/install dependencies"
	@echo "  security       - Run security scan"
	@echo "  check          - Quick pre-commit checks"
	@echo "  dev-core       - Run core in dev mode"
	@echo "  dev-hub        - Run hub in dev mode"
	@echo "  test-race      - Run tests with race detector"
	@echo "  test-coverage  - Run tests with coverage report"
