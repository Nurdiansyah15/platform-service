.PHONY: help dev-up dev-down dev-logs run build tidy migrate-up migrate-down migrate-fresh migrate-version migrate-force migrate-create seed test test-unit test-handler test-integration lint swagger docker-build

COMPOSE_DEV   := docker compose -f docker-compose.dev.yml
GOLANGCI_VER  := v2.5.0
TEST_TIMEOUT  ?= 300s

# ── Help ──────────────────────────────────────────────────────────────────────
help:
	@echo ""
	@echo "  platform-service — perintah yang tersedia:"
	@echo ""
	@echo "  Development:"
	@echo "    make dev-up                    Jalankan postgres dev (tunggu healthy)"
	@echo "    make dev-down                  Hentikan container dev"
	@echo "    make dev-logs                  Tail log postgres dev"
	@echo "    make run                       Jalankan API (go run ./cmd/app)"
	@echo "    make build                     Build semua binary ke ./bin"
	@echo "    make tidy                      go mod tidy"
	@echo ""
	@echo "  Migrasi:"
	@echo "    make migrate-up                Jalankan semua migrasi"
	@echo "    make migrate-down              Rollback satu migrasi terakhir"
	@echo "    make migrate-fresh             Drop semua tabel lalu migrasi ulang (DEV ONLY)"
	@echo "    make migrate-version           Tampilkan versi migrasi"
	@echo "    make migrate-force VERSION=x   Paksa versi (perbaiki state dirty)"
	@echo "    make migrate-create NAME=x     Buat pasangan file migrasi bertimestamp"
	@echo ""
	@echo "  Seeder:"
	@echo "    make seed                      Jalankan semua seeder (opsional: ONLY=\"a b\")"
	@echo ""
	@echo "  Quality:"
	@echo "    make test                      Semua test (butuh Docker untuk testcontainers)"
	@echo "    make test-unit                 Test tanpa DB (domain, usecase, middleware, config)"
	@echo "    make test-handler              Handler test (opsional: RUN=TestTenant)"
	@echo "    make test-integration          Test persistence"
	@echo "    make lint                      golangci-lint (via Docker)"
	@echo "    make swagger                   Generate OpenAPI ke docs/swagger"
	@echo "    make docker-build              Build image app"
	@echo ""

# ── Development ───────────────────────────────────────────────────────────────
dev-up:
	$(COMPOSE_DEV) up -d --wait postgres

dev-down:
	$(COMPOSE_DEV) down

dev-logs:
	$(COMPOSE_DEV) logs -f --tail=$(or $(TAIL),100) postgres

run:
	go run ./cmd/app

build:
	CGO_ENABLED=0 go build -trimpath -o bin/app ./cmd/app
	CGO_ENABLED=0 go build -trimpath -o bin/migrate ./cmd/migrate
	CGO_ENABLED=0 go build -trimpath -o bin/seeder ./cmd/seeder

tidy:
	go mod tidy

# ── Migrasi ───────────────────────────────────────────────────────────────────
migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-fresh:
	go run ./cmd/migrate fresh

migrate-version:
	go run ./cmd/migrate version

migrate-force:
	@test -n "$(VERSION)" || (echo "VERSION wajib diisi, contoh: make migrate-force VERSION=20261014070000" && exit 1)
	go run ./cmd/migrate force $(VERSION)

migrate-create:
	@test -n "$(NAME)" || (echo "NAME wajib diisi, contoh: make migrate-create NAME=create_tenants" && exit 1)
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
