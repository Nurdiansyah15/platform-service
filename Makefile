.PHONY: help dev-up dev-down dev-logs dev-ps dev-restart dev-migrate dev-seed dev-shell dev-build dev-web web-install web-dev web-build web-typecheck run run-worker build tidy migrate-up migrate-down migrate-fresh migrate-version migrate-force migrate-create seed test test-unit test-handler test-integration lint swagger docker-build

# Lewat `sudo make`, id -u = 0; pakai UID pemanggil asli agar file di bind mount tidak root-owned.
DEV_UID       ?= $(or $(SUDO_UID),$(shell id -u))
DEV_GID       ?= $(or $(SUDO_GID),$(shell id -g))
COMPOSE_DEV   := DEV_UID=$(DEV_UID) DEV_GID=$(DEV_GID) docker compose -f docker-compose.dev.yml
GOLANGCI_VER  := v2.5.0
TEST_TIMEOUT  ?= 300s

# ── Help ──────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  platform-service — perintah yang tersedia:"
	@echo ""
	@echo "  Development (Docker — source di-bind-mount dari host):"
	@echo "    make dev-up                    Build + jalankan postgres, migrate, app, worker"
	@echo "    make dev-down                  Hentikan container dev"
	@echo "    make dev-logs SVC=app          Tail log (kosong = semua service)"
	@echo "    make dev-ps                    Status container dev"
	@echo "    make dev-restart               Restart app + worker (muat ulang kode)"
	@echo "    make dev-build                 Build ulang image dev"
	@echo "    make dev-web                   Jalankan hanya UI Nuxt (web/)"
	@echo "    make dev-migrate CMD=up        Jalankan migrasi di container (default: up; CMD=fresh ikut restart app + worker)"
	@echo "    make dev-seed ONLY=\"a b\"       Jalankan seeder di container"
	@echo "    make dev-shell                 Shell masuk container app"
	@echo ""
	@echo "  UI / Nuxt (web/, bun di container oven/bun):"
	@echo "    make web-install               bun install"
	@echo "    make web-dev                   bun dev"
	@echo "    make web-build                 bun build"
	@echo "    make web-typecheck             Vue typecheck"
	@echo ""
	@echo "  Development (host, tanpa Docker):"
	@echo "    make run                       Jalankan API (go run ./cmd/app)"
	@echo "    make run-worker                Jalankan worker webhook (go run ./cmd/worker)"
	@echo "    make build                     Build semua binary ke ./bin"
	@echo "    make tidy                      go mod tidy"
	@echo ""
	@echo "  Migrasi (host):"
	@echo "    make migrate-up                Jalankan semua migrasi"
	@echo "    make migrate-down              Rollback satu migrasi terakhir"
	@echo "    make migrate-fresh             Drop semua tabel lalu migrasi ulang (DEV ONLY)"
	@echo "    make migrate-version           Tampilkan versi migrasi"
	@echo "    make migrate-force VERSION=x   Paksa versi (perbaiki state dirty)"
	@echo "    make migrate-create NAME=x     Buat pasangan file migrasi bertimestamp"
	@echo ""
	@echo "  Seeder (host):"
	@echo "    make seed                      Jalankan semua seeder (opsional: ONLY=\"a b\")"
	@echo ""
	@echo "  Quality:"
	@echo "    make test                      Semua test (butuh Docker untuk testcontainers)"
	@echo "    make test-unit                 Test tanpa DB (domain, usecase, middleware, config)"
	@echo "    make test-handler              Handler test (opsional: RUN=TestCompany)"
	@echo "    make test-integration          Test persistence"
	@echo "    make lint                      golangci-lint (via Docker)"
	@echo "    make swagger                   Generate OpenAPI ke docs/swagger"
	@echo "    make docker-build              Build image app"
	@echo ""

# ── Development (Docker) ──────────────────────────────────────────────────────
# Source di-bind-mount dari host; postgres memakai volume bernama.
dev-up: web-install
	$(COMPOSE_DEV) up -d --build

dev-down:
	$(COMPOSE_DEV) --profile tools down

dev-logs:
	$(COMPOSE_DEV) logs -f --tail=$(or $(TAIL),100) $(SVC)

dev-ps:
	$(COMPOSE_DEV) --profile tools ps

dev-restart:
	$(COMPOSE_DEV) restart app worker web

dev-build:
	$(COMPOSE_DEV) build --no-cache app

dev-web: web-install
	$(COMPOSE_DEV) up -d web

# `fresh` mengosongkan semua tabel termasuk signing_keys, padahal app hanya membuat
# signing key saat startup → restart app + worker agar key active dibuat ulang.
dev-migrate:
	$(COMPOSE_DEV) run --rm migrate $(or $(CMD),up)
	$(if $(filter fresh,$(CMD)),$(COMPOSE_DEV) restart app worker,)

dev-seed:
	$(COMPOSE_DEV) --profile tools run --rm seeder $(ONLY)

dev-shell:
	$(COMPOSE_DEV) run --rm --entrypoint sh app

# ── UI / Nuxt (container oven/bun, tidak bergantung bun di host) ─────────────
# bun dijalankan di image service `web` sebagai UID host (cache di /tmp
# container). Untuk bun di host: make web-install WEB_BUN="cd web && bun".
WEB_BUN ?= $(COMPOSE_DEV) run --rm --no-deps -T web bun

web-install:
	$(WEB_BUN) install

web-dev:
	$(COMPOSE_DEV) run --rm --no-deps --service-ports web bun --bun run dev --host 0.0.0.0

web-build:
	$(WEB_BUN) --bun run build

web-typecheck:
	$(WEB_BUN) run typecheck

# ── Development (host) ────────────────────────────────────────────────────────
run:
	go run ./cmd/app

run-worker:
	go run ./cmd/worker

build:
	CGO_ENABLED=0 go build -trimpath -o bin/app ./cmd/app
	CGO_ENABLED=0 go build -trimpath -o bin/migrate ./cmd/migrate
	CGO_ENABLED=0 go build -trimpath -o bin/seeder ./cmd/seeder
	CGO_ENABLED=0 go build -trimpath -o bin/worker ./cmd/worker

tidy:
	go mod tidy

# ── Migrasi ───────────────────────────────────────────────────────────────────
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-fresh:
	go run ./cmd/migrate fresh
	@echo "Catatan: signing_keys ikut terhapus. Restart app (make run / make dev-restart) agar signing key dibuat ulang."

migrate-version:
	go run ./cmd/migrate version

migrate-force:
	@test -n "$(VERSION)" || (echo "VERSION wajib diisi, contoh: make migrate-force VERSION=20261014070000" && exit 1)
	go run ./cmd/migrate force $(VERSION)

migrate-create:
	@test -n "$(NAME)" || (echo "NAME wajib diisi, contoh: make migrate-create NAME=create_companies" && exit 1)
	@ts=$$(date -u +%Y%m%d%H%M%S); \
	up=internal/migrations/$${ts}_$(NAME).up.sql; down=internal/migrations/$${ts}_$(NAME).down.sql; \
	touch $$up $$down; echo "dibuat: $$up"; echo "dibuat: $$down"

# ── Seeder ────────────────────────────────────────────────────────────────────
seed:
	go run ./cmd/seeder $(ONLY)

# ── Quality ───────────────────────────────────────────────────────────────────
test:
	go test ./... -count=1 -timeout $(TEST_TIMEOUT)

test-unit:
	go test ./internal/domain/... ./internal/app/... ./internal/config/... ./internal/interfaces/http/middleware/... ./internal/interfaces/http/httperror/... -count=1

test-handler:
	go test ./internal/interfaces/http/handler/... -count=1 -timeout $(TEST_TIMEOUT) $(if $(RUN),-run "$(RUN)" -v,)

test-integration:
	go test ./internal/infrastructure/persistence/... -count=1 -timeout $(TEST_TIMEOUT)

lint:
	docker run --rm -v $(CURDIR):/app -w /app \
		-v golangci-cache:/root/.cache \
		golangci/golangci-lint:$(GOLANGCI_VER) golangci-lint run ./...

swagger:
	swag init -g cmd/app/main.go -o docs/swagger --outputTypes json,yaml --parseInternal

docker-build:
	docker build --target app -t platform-service/app .
