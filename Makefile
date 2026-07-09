.PHONY: all build clean dev lint test help

all: build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Go (Core) ───────────────────────────────────────────────────────────────

core-deps: ## Install Go dependencies
	cd core && go mod tidy

core-build: ## Build RenCore binary
	cd core && go build -o renos-core ./cmd/renos-core/

core-run: ## Run RenCore locally
	cd core && go run ./cmd/renos-core/

core-test: ## Run Core tests
	cd core && go test -v -count=1 ./...

core-lint: ## Lint Core code
	cd core && golangci-lint run ./...

# ─── Rust (Agent) ────────────────────────────────────────────────────────────

agent-deps: ## Fetch Agent dependencies
	cd agent && cargo fetch

agent-build: ## Build RenAgent binary
	cd agent && cargo build --release

agent-run: ## Run RenAgent locally
	cd agent && cargo run --

agent-test: ## Run Agent tests
	cd agent && cargo test

agent-lint: ## Lint Agent code
	cd agent && cargo clippy -- -D warnings

agent-check: ## Check Agent compiles (fast)
	cd agent && cargo check

# ─── Next.js (Hub) ───────────────────────────────────────────────────────────

hub-install: ## Install Hub dependencies
	cd hub && npm install

hub-dev: ## Start Hub dev server
	cd hub && npm run dev

hub-build: ## Build Hub for production
	cd hub && npm run build

hub-test: ## Run Hub tests
	cd hub && npm test

hub-lint: ## Lint Hub code
	cd hub && npm run lint

hub-typecheck: ## TypeScript type check
	cd hub && npx tsc --noEmit

# ─── Docker Compose ──────────────────────────────────────────────────────────

up: dc-up ## Start development dependencies (alias)
down: dc-down ## Stop development dependencies (alias)

dc-up: ## Start development dependencies
	docker compose up -d

dc-down: ## Stop development dependencies
	docker compose down

dc-logs: ## View dependency logs
	docker compose logs -f

dc-restart: ## Restart dependencies (preserves data)
	docker compose restart

dc-ps: ## List running containers and health
	docker compose ps

dc-status: ## Show health status of all services
	docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

dc-reset: ## Remove volumes, reset all data (IRREVERSIBLE)
	docker compose down -v

# Short aliases
logs: dc-logs ## View dependency logs (alias)
ps: dc-ps ## List running containers (alias)
status: dc-status ## Show health status (alias)
reset: dc-reset ## Reset all data, destroy volumes (alias)
restart: dc-restart ## Restart containers (alias)

# ─── Proto ───────────────────────────────────────────────────────────────────

proto-gen: ## Generate proto stubs (requires protoc)
	cd proto && buf generate

proto-lint: ## Lint proto files
	cd proto && buf lint

# ─── Quality ─────────────────────────────────────────────────────────────────

lint: core-lint agent-lint hub-lint hub-typecheck ## Run all linters

test: core-test agent-test hub-test ## Run all tests

build: core-build agent-build hub-build ## Build all components

fmt: ## Format all code
	cd core && gofmt -w -s ./..
	cd agent && cargo fmt
	cd hub && npx prettier --write .

check: lint test build ## Full pre-commit check

# ─── CI (used in GitHub Actions) ──────────────────────────────────────────────

ci-core: core-deps core-lint core-build core-test ## CI target for Core
ci-agent: agent-deps agent-lint agent-build agent-test ## CI target for Agent
ci-hub: hub-install hub-lint hub-typecheck hub-build hub-test ## CI target for Hub

# ─── Development ─────────────────────────────────────────────────────────────

dev: dc-up ## Start development environment (dependencies only)
	@echo ""
	@echo "  RenOS development environment is ready."
	@echo "  ───────────────────────────────────────────"
	@echo ""
	@echo "  Start Core in another terminal:"
	@echo "    make core-run"
	@echo ""
	@echo "  Start Hub in another terminal:"
	@echo "    make hub-install hub-dev"
	@echo ""
	@echo "  Start Agent in another terminal:"
	@echo "    make agent-run"
	@echo ""
	@echo "  Verify dependencies:"
	@echo "    make dc-status"
	@echo "    curl http://localhost:${RENOS_PORT:-8443}/api/v1/healthz"
	@echo ""

dev-full: dc-up ## Start full dev environment
	@echo "Development environment is ready."
	@echo "  Core:  cd core && go run ./cmd/renos-core/"
	@echo "  Agent: cd agent && cargo run --"
	@echo "  Hub:   cd hub && npm run dev"
	@echo "  DB:    psql postgres://renos:renos@localhost:5432/renos"
