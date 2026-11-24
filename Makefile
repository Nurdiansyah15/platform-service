.PHONY: help dev-up dev-down dev-logs dev-ps dev-restart dev-migrate dev-migrate-create dev-seed dev-shell dev-build dev-web dev-web-install dev-web-serve dev-web-build dev-web-typecheck dev-test dev-test-unit dev-test-handler dev-test-integration dev-swagger dev-lint

# Lewat `sudo make`, id -u = 0; pakai UID pemanggil asli agar file di bind mount tidak root-owned.
DEV_UID       ?= $(or $(SUDO_UID),$(shell id -u))
DEV_GID       ?= $(or $(SUDO_GID),$(shell id -g))
COMPOSE_DEV   := DEV_UID=$(DEV_UID) DEV_GID=$(DEV_GID) docker compose -f docker-compose.dev.yml
# Container toolbox (go + swag). Default root (akses socket Docker untuk testcontainers);
# TOOLBOX_USER untuk perintah yang menulis file ke bind mount agar tidak root-owned.
TOOLBOX       := $(COMPOSE_DEV) --profile tools run --rm --no-deps toolbox
TOOLBOX_USER  := $(COMPOSE_DEV) --profile tools run --rm --no-deps --user $(DEV_UID):$(DEV_GID) -e HOME=/tmp -e GOCACHE=/tmp/go-build toolbox
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
	@echo "    make dev-migrate CMD=up        Migrasi di container: up | down | fresh | version | \"force <versi>\""
	@echo "                                   (default: up; CMD=fresh ikut restart app + worker)"
	@echo "    make dev-migrate-create NAME=x Buat pasangan file migrasi bertimestamp"
	@echo "    make dev-seed ONLY=\"a b\"       Jalankan seeder di container"
	@echo "    make dev-shell                 Shell masuk container app"
	@echo ""
	@echo "  UI / Nuxt (web/, bun di container oven/bun via compose dev):"
	@echo "    make dev-web-install           bun install"
	@echo "    make dev-web-serve             bun dev (foreground, port di-publish)"
	@echo "    make dev-web-build             bun build"
	@echo "    make dev-web-typecheck         Vue typecheck"
	@echo ""
	@echo "  Quality (container toolbox; testcontainers lewat socket Docker host):"
	@echo "    make dev-test                  Semua test"
	@echo "    make dev-test-unit             Test tanpa DB (domain, usecase, middleware, config)"
	@echo "    make dev-test-handler          Handler test (opsional: RUN=TestCompany)"
	@echo "    make dev-test-integration      Test persistence"
	@echo "    make dev-swagger               Generate OpenAPI ke docs/swagger (+ web/public/openapi.json)"
	@echo "    make dev-lint                  golangci-lint (via Docker)"
	@echo ""

# ── Development (Docker) ──────────────────────────────────────────────────────
# Source di-bind-mount dari host; postgres memakai volume bernama.
dev-up: dev-web-install
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

dev-web: dev-web-install
	$(COMPOSE_DEV) up -d web

# `fresh` mengosongkan semua tabel termasuk signing_keys, padahal app hanya membuat
# signing key saat startup → restart app + worker agar key active dibuat ulang.
dev-migrate:
	$(COMPOSE_DEV) run --rm migrate $(or $(CMD),up)
	$(if $(filter fresh,$(CMD)),$(COMPOSE_DEV) restart app worker,)

dev-migrate-create:
	@test -n "$(NAME)" || (echo "NAME wajib diisi, contoh: make dev-migrate-create NAME=create_companies" && exit 1)
	@$(TOOLBOX_USER) sh -c 'ts=$$(date -u +%Y%m%d%H%M%S); \
		up=internal/migrations/$${ts}_$(NAME).up.sql; down=internal/migrations/$${ts}_$(NAME).down.sql; \
		touch $$up $$down; echo "dibuat: $$up"; echo "dibuat: $$down"'

dev-seed:
	$(COMPOSE_DEV) --profile tools run --rm seeder $(ONLY)

dev-shell:
	$(COMPOSE_DEV) run --rm --entrypoint sh app

# ── UI / Nuxt (container oven/bun, tidak bergantung bun di host) ─────────────
# bun dijalankan di image service `web` sebagai UID host (cache di /tmp
# container). Untuk bun di host: make dev-web-install WEB_BUN="cd web && bun".
WEB_BUN ?= $(COMPOSE_DEV) run --rm --no-deps -T web bun

dev-web-install:
	$(WEB_BUN) install

dev-web-serve:
	$(COMPOSE_DEV) run --rm --no-deps --service-ports web bun --bun run dev --host 0.0.0.0

dev-web-build:
	$(WEB_BUN) --bun run build

dev-web-typecheck:
	$(WEB_BUN) run typecheck

# ── Quality (container toolbox) ───────────────────────────────────────────────
dev-test:
	$(TOOLBOX) go test ./... -count=1 -timeout $(TEST_TIMEOUT)

dev-test-unit:
	$(TOOLBOX) go test ./internal/domain/... ./internal/app/... ./internal/config/... ./internal/interfaces/http/middleware/... ./internal/interfaces/http/httperror/... -count=1

dev-test-handler:
	$(TOOLBOX) go test ./internal/interfaces/http/handler/... -count=1 -timeout $(TEST_TIMEOUT) $(if $(RUN),-run "$(RUN)" -v,)

dev-test-integration:
	$(TOOLBOX) go test ./internal/infrastructure/persistence/... -count=1 -timeout $(TEST_TIMEOUT)

dev-swagger:
	$(TOOLBOX_USER) sh -c 'swag init -g cmd/app/main.go -o docs/swagger --outputTypes json,yaml --parseInternal && cp docs/swagger/swagger.json web/public/openapi.json'

dev-lint:
	docker run --rm -v $(CURDIR):/app -w /app \
		-v golangci-cache:/root/.cache \
		golangci/golangci-lint:$(GOLANGCI_VER) golangci-lint run ./...
