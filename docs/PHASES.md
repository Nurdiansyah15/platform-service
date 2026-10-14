# Roadmap Fase Pengerjaan — Central SaaS Platform Service

Sumber kebutuhan: [`central-saas-platform-spec.md`](central-saas-platform-spec.md).
Konvensi arsitektur & test: [`CLAUDE.md`](../CLAUDE.md) (diadaptasi dari `k-forum-api`).

## Aturan Umum

- **Fase 0** = fondasi; wajib selesai sebelum domain pertama masuk.
- **Fase 1–17** = per domain, urut dependensi. Hanya endpoint **MVP (spec §4)** + *enabler*.
- **Fase L1–L9** = endpoint non-MVP (spec §3), dikerjakan setelah Milestone MVP.
- *Enabler* = endpoint non-MVP yang ikut dikerjakan karena dibutuhkan agar alur MVP bisa dipakai.
- **Definisi selesai per fase:**
  - Migrasi up/down bersih (`make migrate-up` / `make migrate-down`).
  - Semua endpoint fase terdaftar di router, dengan method usecase di `<Ctx>UseCase` (satu file per domain) + DTO + swag annotation.
  - `<domain>_handler_test.go` ada dan memuat skenario wajib (lihat di bawah).
  - `go build ./...`, `go test ./... -count=1`, `make lint` hijau.
  - Seed helper baru ditambahkan ke `internal/testhelper/seed.go` bila domain dipakai fase berikutnya.
- **Skenario handler test wajib per endpoint:**
  - `Unauthenticated` → 401 (semua endpoint terproteksi)
  - `Forbidden` → 403 (bukan platform admin / bukan member tenant / scope service kurang / tenant tidak subscribe / `LICENSE_MANAGED`)
  - `InvalidPayload` → 400/422 (endpoint ber-body)
  - `NotFound` → 404 (endpoint by ID)
  - `Duplicate` / `Conflict` → 409 (bila ada unique constraint)
  - `Success` → 2xx + assert isi `data` (dan `meta` untuk list)

Keputusan desain di luar spec:

- Platform admin diidentifikasi lewat kolom `users.is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE` + middleware `RequirePlatformAdmin`.

---

## Fase 0 — Fondasi ✅

### 0.1 Scaffolding & tooling
- [x] `go.mod` (module `platform-service`, Go 1.25) + layout direktori (lihat `CLAUDE.md` §1)
- [x] `Makefile`: `dev-up`, `dev-down`, `run`, `build`, `migrate-up/down/fresh/version/force`, `migrate-create NAME=`, `seed`, `test`, `test-unit`, `test-handler`, `test-integration`, `lint`, `swagger`, `docker-build`
- [x] `Dockerfile` multi-stage (target `app`, `migrate`, `seeder`; target `worker` ditambah di Fase 16)
- [x] `docker-compose.dev.yml` (postgres:16-alpine, port host 5442)
- [x] `.env.example`, `.gitignore`, `.dockerignore`, `test.http`
- [x] `.golangci.yml` (golangci-lint v2, dijalankan via Docker)
- [x] CI `.github/workflows/ci.yml`: lint + build + vet + `go test ./...`

### 0.2 Core runtime
- [x] `internal/config`: env + godotenv, `DEPLOYMENT_MODE=cloud|on_prem` (divalidasi), `ISSUER_URL`, `ACCESS_TOKEN_TTL`, `REFRESH_TOKEN_TTL`, `CORS_ALLOW_ORIGINS`
- [x] `internal/logger`: slog (JSON di production) + request_id dari context
- [x] `infrastructure/database`: pool `database/sql` + `pgx/stdlib`
- [x] `cmd/migrate` (up/down/fresh/version/force) dengan migrasi di-embed (`internal/migrations`)
- [x] Migrasi awal: fungsi trigger `set_updated_at()` (UUID memakai `gen_random_uuid()` bawaan PG13+, tanpa `pgcrypto`)
- [x] `port.Transactor` + `persistence/postgres_transactor.go` (`TxFromContext`, `NewTxContext`, `execFromContext`) + test
- [x] `cmd/seeder` + `internal/seeders` (registry kosong; seeder platform admin di Fase 3)

### 0.3 Error & response
- [x] `internal/domain/errors` (`domainerr.New(code)`, `domainerr.Wrap(code, err)`)
- [x] `internal/app/apperror` (`BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `Unprocessable`, `ServiceUnavailable`, `Internal`, `LicenseManaged`) + message key auth
- [x] `interfaces/http/respond` (`OK`, `Created`, `Success`, `SuccessWithMeta`, `Error`)
- [x] `interfaces/http/httperror` (`Handle`, `Middleware` recover, `NoRoute`, validator → 422 dengan nama field json) + `locales/{en,id}.json`
- [x] `dto.PaginationParams`, `dto.Meta`, `dto.NewMeta`, `resolvePaginationParams`, `httpquery.List`
- [x] `persistence/pg_errors.go`: `isUniqueViolation`, `isForeignKeyViolation`, `isCheckViolation`, `constraintName`

### 0.4 HTTP shell
- [x] `router.Setup` + `Routes{V1, Authenticated, Deployment}`; domain mendaftar lewat `RegisterXxxRoutes` di `registerDomainRoutes`
- [x] Middleware: request-id, request-logger, locale, recover, CORS
- [x] `GET /healthz` (system usecase → ping DB; 503 bila DB tidak sehat)
- [x] Auth middleware: `Authenticate(port.TokenAuthenticator)` + `principal.Principal` (Kind `user`/`service`) + guard `RequireUser`, `RequireService(scope)`, `RequireUserOrService(scope)`, `RequirePlatformAdmin`, `RequireActiveTenant`
  - Fase 0–1 `bootstrap` memakai `rejectAllAuthenticator`; diganti verifier JWT di Fase 2
- [x] `DeploymentGuard`: `LicenseManaged()` (403 `LICENSE_MANAGED` di on_prem), `CloudOnly()` / `OnPremOnly()` (404)

### 0.5 Outbox dasar
- [x] Migrasi `events` + index `idx_events_occurred_at`, `idx_events_tenant_id`
- [x] `port.EventOutbox.Append(ctx, events...)` → `PostgresEventOutbox` (menolak dipanggil di luar transaksi: `ErrOutboxRequiresTx`)

### 0.6 Test infrastructure
- [x] `internal/bootstrap.NewEngine`: wiring repo → usecase → handler → router, dipakai `cmd/app` **dan** testhelper
- [x] `testhelper/testdb.go`: testcontainers postgres:16-alpine + semua migrasi (embed)
- [x] `testhelper/testserver.go`: `MustStartTestServer(opts...)` (`WithDeploymentMode`, `WithAuthenticator`), `GET/POST/PUT/PATCH/DELETE/DELETEBody`, `BearerHeader`, `DecodeBody`, `DecodeError`
- [x] `testhelper/txhelper.go`, `assert.go`
- [x] `handler/api/main_test.go`, `system_handler_test.go`
- [x] Test lain: config, system usecase, middleware auth & deployment guard, httperror, persistence (transactor, outbox, pagination, pg errors)
- Ditunda ke fase yang membutuhkan: `fixtures.go` (Fase 17)

**Verifikasi (2026-10-14):** `make dev-up && make migrate-up/down/up` bersih, `GET /healthz` 200, `go test ./... -count=1` hijau, `make lint` 0 issue, `make swagger` & `docker build --target app` berhasil.

---

## Fase Domain (MVP)

### Fase 1 — Solution Catalog ✅
- Tabel: `solutions`
- Endpoint:
  - [x] `GET /solutions` (MVP) — token valid apa pun; non-admin hanya melihat `active`; filter `status` (admin), `search`, pagination & sort
  - [x] `POST /solutions` (enabler) — platform admin; `LICENSE_MANAGED` di on_prem
  - [x] `GET /solutions/{solutionId}` (enabler) — solution `inactive` hanya terlihat oleh platform admin (lainnya 404)
- Test: `solution_handler_test.go` (21 skenario), `solution_test.go` (entity), `postgres_solution_repository_test.go`
- Seed: `testhelper.MustSeedSolution`, `testhelper.UniqueCode`
- Test infra tambahan:
  - `testhelper.FakeAuthenticator` (`UserToken`, `PlatformAdminToken`, `ServiceToken`) — dipakai default oleh `MustStartTestServer` sampai login nyata ada
  - `TestServer.MustDerive(opts...)` → `testOnPremSrv` (mode on_prem, DB sama) untuk skenario `LICENSE_MANAGED`

### Fase 2 — Signing Keys & JWKS ✅
- Tabel: `signing_keys` (unique `kid`, CHECK status, partial unique `uq_signing_keys_single_active`)
- Domain: entity `SigningKey` (kid `YYYYMMDD-xxxxxxxx`, transisi active → retiring → retired), `KeyStatus.IsPublished()`
- Port:
  - `KeyStore` bergaya KMS: `Generate` → (`private_key_ref`, public key PEM), `Signer(ref)` → `crypto.Signer`
  - `TokenIssuer`: `IssueUserToken`, `IssueServiceToken` (dipakai Fase 3 & 4)
- Infrastruktur:
  - `keystore.FileKeyStore` (`KEYSTORE_DIR`, file 0600, ref `file:<uuid>.pem`) dan `keystore.MemoryKeyStore` (test)
  - `jwttoken.KeyRing` (cache signer & public key per kid, TTL 1 menit), `Issuer` (RS256 manual via `crypto.Signer`), `Authenticator` (golang-jwt, `alg` dikunci RS256, cek `iss`/`exp`/`iat`/`typ`/`aud`)
- Startup: `SigningKeyUseCase.EnsureActiveKey` membuat key bila belum ada; aman untuk banyak instance
- Endpoint:
  - [x] `GET /.well-known/jwks.json` (MVP, publik) — format RFC 7517 tanpa envelope, `Cache-Control: public, max-age=300`
- `bootstrap.New` → `App{Engine, TokenIssuer}`; authenticator produksi = JWT (menggantikan `rejectAllAuthenticator`). Test memakai rantai `FakeAuthenticator` → JWT
- Test: `jwks_handler_test.go`, `jwttoken_test.go` (round trip, rotasi, token ditolak: payload diubah / HS256 / issuer lain / kedaluwarsa / kid tak dikenal / claim kurang), `key_store_test.go`, `signing_key_test.go`, `postgres_signing_key_repository_test.go`
- Belum: pengecekan sesi & status user pada token user (Fase 3), status solution client pada token service (Fase 4), endpoint rotasi (L7)

### Fase 3 — Users & Auth Session
- Tabel: `users` (+ `is_platform_admin`), `auth_sessions`
- Domain:
  - `User` (status, password bcrypt)
  - `AuthSession` (refresh token opaque, simpan hash SHA-256, rotate, revoke)
  - Claim user token: `iss, sub, aud, tid, sid, typ=user, iat, exp, jti`
- Endpoint:
  - [ ] `POST /auth/login` (MVP; wajib `solution`)
  - [ ] `POST /auth/refresh` (MVP)
  - [ ] `POST /auth/logout` (MVP)
  - [ ] `GET /auth/me` (MVP)
  - [ ] `POST /users` (enabler, admin)
- Lainnya: middleware user token aktif (cek signature + sesi tidak revoked), seeder platform admin
- Tambah pengecekan di atas `jwttoken.Authenticator` untuk token user: sesi tidak revoked/expired, user `active`, isi `IsPlatformAdmin` dari DB
- Test: `testAdminTok` di `main_test.go` diganti hasil login platform admin sungguhan. `FakeAuthenticator` tetap tersedia (sudah dirantai sebelum JWT)
- Test: `auth_handler_test.go`, `user_handler_test.go`
  - password salah 401, user suspended 403, solution tidak ada 422/404
  - refresh rotate → refresh token lama ditolak
  - logout → refresh ditolak
  - `/auth/me` 200 / 401
- Catatan: `ip_address` pakai `NULLIF($n, '')::inet` (`RemoteAddr` kosong di test)

### Fase 4 — Solution Clients & Service Auth
- Tabel: `solution_clients`
- Domain:
  - `SolutionClient` (secret hash, VO `Scopes` tervalidasi ke daftar scope spec §2.8)
  - Claim service token: `iss, sub, aud=platform, sol, scope, typ=service, iat, exp, jti`
- Endpoint:
  - [ ] `POST /solutions/{solutionId}/clients` (MVP, admin; secret tampil sekali)
  - [ ] `POST /solution-clients/{clientId}/rotate-secret` (MVP, admin)
  - [ ] `POST /oauth/token` (MVP, `grant_type=client_credentials`)
- Lainnya: aktifkan `RequireService(scope)`
- Test: `solution_client_handler_test.go`, `oauth_handler_test.go`
  - secret salah 401, client revoked 401, scope tidak dikenal 422, token memuat claim yang benar
- Seed: `MustSeedSolutionClient` (return token service siap pakai)

### Fase 5 — Tenant & Membership
- Tabel: `tenants`, `tenant_members`
- Domain:
  - `Tenant`, `TenantMember`
  - Policy akses sementara: platform admin **atau** member aktif (lihat Item Terbuka)
  - Pembuat tenant otomatis jadi member
  - Outbox: `tenant.updated`, `tenant_member.added`
- Endpoint:
  - [ ] `POST /tenants` (MVP)
  - [ ] `GET /tenants` (MVP)
  - [ ] `GET /tenants/{tenantId}` (MVP)
  - [ ] `GET /tenants/{tenantId}/members` (MVP)
  - [ ] `POST /tenants/{tenantId}/members` (MVP)
- Test: `tenant_handler_test.go`, `tenant_member_handler_test.go`
  - 403 non-member, 409 slug duplikat, list hanya tenant milik user, event outbox tertulis

### Fase 6 — Company & Branch
- Tabel: `companies`, `branches` (soft delete, partial unique code)
- Outbox: `company.created`, `branch.created`
- Endpoint:
  - [ ] `GET /tenants/{tenantId}/companies` (MVP)
  - [ ] `POST /tenants/{tenantId}/companies` (MVP)
  - [ ] `GET /companies/{companyId}/branches` (MVP)
  - [ ] `POST /companies/{companyId}/branches` (MVP)
- Test: `company_handler_test.go`, `branch_handler_test.go`
  - 403 tenant lain, 409 code duplikat, 404 company tidak ada / soft-deleted

### Fase 7 — Benefits (enabler)
- Tabel: `benefits` (+ `UNIQUE (id, solution_id)`)
- Domain: VO `ValueType`, `AggregationStrategy`, `Metering`, `UsageReset`; validasi nilai JSONB sesuai `value_type` (dipakai Fase 8–9)
- Endpoint:
  - [ ] `POST /solutions/{solutionId}/benefits` (enabler)
  - [ ] `GET /solutions/{solutionId}/benefits` (enabler)
  - [ ] `GET /benefits/{benefitId}` (enabler)
- Test: `benefit_handler_test.go`

### Fase 8 — Plans & Plan Benefits
- Tabel: `plans`, `plan_benefits` (composite FK)
- Endpoint:
  - [ ] `POST /solutions/{solutionId}/plans` (MVP)
  - [ ] `GET /solutions/{solutionId}/plans` (MVP)
  - [ ] `GET /plans/{planId}/benefits` (MVP)
  - [ ] `PUT /plans/{planId}/benefits` (MVP, replace-all)
- Test: `plan_handler_test.go`
  - benefit dari solution lain 422, value tidak cocok `value_type` 422, PUT idempotent

### Fase 9 — Add-ons & Add-on Benefits
- Tabel: `addons`, `addon_benefits` (composite FK)
- Endpoint:
  - [ ] `POST /solutions/{solutionId}/addons` (MVP)
  - [ ] `GET /solutions/{solutionId}/addons` (MVP)
  - [ ] `GET /addons/{addonId}/benefits` (MVP)
  - [ ] `PUT /addons/{addonId}/benefits` (MVP)
- Test: `addon_handler_test.go` (pola sama Fase 8)

### Fase 10 — Subscriptions, Snapshot & Tenant Switch
- Tabel: `subscriptions`, `subscription_addons`, `subscription_benefit_snapshots` (`UNIQUE NULLS NOT DISTINCT`)
- Domain service `SnapshotBuilder`:
  - create subscription → salin `plan_benefits`
  - change-plan → hapus snapshot plan lama, salin plan baru
  - tambah add-on → salin `addon_benefits`
  - cancel add-on → hapus snapshot add-on
  - semua dalam satu tx + outbox `subscription.*`, `subscription_addon.*`, `entitlement.changed`
- Endpoint:
  - [ ] `POST /tenants/{tenantId}/subscriptions` (MVP)
  - [ ] `GET /tenants/{tenantId}/subscriptions` (MVP)
  - [ ] `POST /subscriptions/{subscriptionId}/change-plan` (MVP)
  - [ ] `POST /subscriptions/{subscriptionId}/cancel` (MVP)
  - [ ] `GET /subscriptions/{subscriptionId}/addons` (MVP)
  - [ ] `POST /subscriptions/{subscriptionId}/addons` (MVP)
  - [ ] `GET /auth/tenants` (MVP)
  - [ ] `POST /auth/switch-tenant` (MVP)
- Lainnya: login auto-mengisi `tid` bila user hanya punya 1 tenant yang subscribe solution sesi
- Test: `subscription_handler_test.go`, `subscription_addon_handler_test.go`, `auth_tenant_handler_test.go`
  - plan dari solution lain 422
  - snapshot tidak berubah saat `plan_benefits` diubah setelah subscribe
  - switch ke tenant tanpa subscription aktif 403
  - token dengan `tid` kosong ditolak endpoint tenant-scoped

### Fase 11 — Usage & Metering
- Tabel: `usage_records`, `tenant_usage`
- Domain:
  - Resolver periode (`never` / `billing_period` / `monthly`)
  - counter = delta (`used += quantity`), gauge = absolute (`used = quantity`)
  - `metering = none` ditolak
  - Idempotent per `(solution_id, idempotency_key)`
- Endpoint (service token, scope `usage:write`):
  - [ ] `POST /tenants/{tenantId}/solutions/{solutionId}/usage` (MVP)
  - [ ] `PUT /tenants/{tenantId}/solutions/{solutionId}/usage/{benefitCode}` (MVP)
- Test: `usage_handler_test.go`
  - scope kurang 403, client dari solution lain 403, idempotency key berulang tidak menambah, benefit `none` 422

### Fase 12 — Effective Entitlements
- Domain service `EntitlementCalculator`:
  - aggregation `sum` / `max` / `min` / `override` / `any_true`
  - nilai add-on × `quantity` bila relevan
  - `limit`, `used`, `remaining` untuk benefit yang di-meter
  - **unit test domain** table-driven
- Outbox `usage.limit_reached` saat `used >= limit` (dipicu dari usecase usage Fase 11)
- Endpoint (user tenant aktif atau service `entitlements:read`):
  - [ ] `GET /tenants/{tenantId}/solutions/{solutionId}/entitlements` (MVP)
  - [ ] `GET /tenants/{tenantId}/solutions/{solutionId}/entitlements/{benefitCode}` (MVP)
- Test: `entitlement_handler_test.go`

### Fase 13 — Permissions & Catalog Sync
- Tabel: `permissions`
- Endpoint:
  - [ ] `PUT /solutions/{solutionId}/permissions/sync` (MVP, service `catalog:sync`)
    - idempotent: `version` sama dengan `permission_catalog_version` → no-op
    - permission yang tidak ada di daftar → `deprecated`
  - [ ] `GET /solutions/{solutionId}/permissions` (enabler)
- Test: `catalog_sync_handler_test.go`
  - version sama = tidak ada perubahan, permission deprecated tidak terhapus

### Fase 14 — Roles (Global & Tenant Custom)
- Tabel: `roles`, `role_permissions` (CHECK, partial unique, composite FK sesuai spec)
- Domain:
  - `Role` (system role immutable, kepemilikan tenant)
  - custom role hanya untuk solution yang di-subscribe tenant (`trialing` / `active`)
  - permission `deprecated` tidak boleh ditambahkan
- Endpoint:
  - [ ] `GET /solutions/{solutionId}/roles` (MVP)
  - [ ] `POST /solutions/{solutionId}/roles` (MVP)
  - [ ] `GET /roles/{roleId}/permissions` (MVP)
  - [ ] `PUT /roles/{roleId}/permissions` (MVP)
  - [ ] `GET /tenants/{tenantId}/solutions/{solutionId}/roles` (MVP)
  - [ ] `POST /tenants/{tenantId}/solutions/{solutionId}/roles` (MVP)
  - [ ] `PUT /tenants/{tenantId}/roles/{roleId}/permissions` (MVP)
- Test: `role_handler_test.go`, `tenant_role_handler_test.go`
  - role system tidak bisa diubah 403, custom role milik tenant lain 404, permission dari solution lain 422, tenant tidak subscribe 403

### Fase 15 — Role Assignment & Authz Check
- Tabel: `user_role_assignments`
- Domain service:
  - `ScopeValidator` (scope tenant/company/branch harus milik tenant)
  - `PermissionEvaluator` (assignment di tenant berlaku ke company & branch di bawahnya; company berlaku ke branch-nya)
- Endpoint:
  - [ ] `POST /tenants/{tenantId}/users/{userId}/role-assignments` (MVP)
  - [ ] `POST /authz/check` (MVP)
  - [ ] `POST /authz/check-batch` (MVP)
- Test: `role_assignment_handler_test.go`, `authz_handler_test.go`
  - user bukan member 422, role tenant lain 403, scope bukan milik tenant 422, duplikat 409, check allow/deny per scope

### Fase 16 — Webhooks & Event Delivery
- Tabel: `webhook_endpoints`, `webhook_deliveries`
- `cmd/worker` (+ target `worker` di `Dockerfile`):
  - dispatcher outbox → deliveries (event tenant hanya ke solution yang di-subscribe tenant)
  - header `X-Platform-Event-Id`, `X-Platform-Event-Type`, `X-Platform-Timestamp`, `X-Platform-Signature` (HMAC-SHA256 `timestamp.body`)
  - retry exponential backoff → `dead`
  - secret webhook dienkripsi (AES-GCM, key dari config)
- Endpoint:
  - [ ] `POST /solutions/{solutionId}/webhooks` (MVP; secret tampil sekali)
  - [ ] `GET /events` (MVP; service `events:read`, dibatasi solution pemanggil, filter `since`, `event_type`, `tenant_id`)
- Test: `webhook_handler_test.go`, `event_handler_test.go`; dispatcher diuji dengan `httptest.Server` (header, signature, retry)

### Fase 17 — Licensing (on_prem)
- Tabel: `installations`, `licenses`, `license_checkins`
- Domain:
  - `LicenseVerifier` — public key vendor di-embed di binary (`//go:embed`)
  - state machine `valid` / `grace` / `read_only` / `invalid`
  - deteksi jam mundur (HMAC `last_seen_at`)
  - state di-cache di memori + re-verify berkala
- Aktifkan guard:
  - `LICENSE_MANAGED` (403) untuk endpoint katalog/subscription di spec §2.14
  - `ReadOnlyGuard` untuk operasi tulis saat `read_only` / `invalid`
  - endpoint license → 404 di mode `cloud`
- Endpoint:
  - [ ] `GET /installation` (MVP)
  - [ ] `GET /license` (MVP)
  - [ ] `POST /license` (MVP)
  - [ ] `GET /license/raw` (MVP)
  - [ ] `GET /license/usage-report` (MVP)
- Test: `license_handler_test.go` memakai server kedua `testOnPremSrv` (`MustStartTestServer(WithDeploymentMode("on_prem"))`, database terpisah di container yang sama) + license fixture yang ditandatangani test key
  - cloud → 404, signature invalid 422, fingerprint berbeda → `invalid`
  - `POST /solutions/{id}/plans` → 403 `LICENSE_MANAGED`
  - `grace` → normal + peringatan, `read_only` → tulis ditolak

### Milestone MVP
- [ ] 56 endpoint spec §4 + enabler terimplementasi
- [ ] `go test ./... -count=1` hijau, `make lint` bersih

---

## Fase Lanjutan (non-MVP, spec §3)

Pola sama: usecase + handler + handler test wajib.

### L1 — Identity & Session
- [ ] `GET /auth/sessions`, `DELETE /auth/sessions/{sessionId}`, `POST /auth/logout-all`
- [ ] `GET /users/me`, `PATCH /users/me`
- [ ] `GET /users`, `GET /users/{userId}`, `PATCH /users/{userId}`, `PATCH /users/{userId}/status`

### L2 — Tenant & Organization
- [ ] `PATCH /tenants/{tenantId}`
- [ ] `PATCH /tenants/{tenantId}/members/{userId}`, `DELETE /tenants/{tenantId}/members/{userId}` (status `inactive` + hapus role assignment)
- [ ] `GET/PATCH/DELETE /companies/{companyId}`, `GET/PATCH/DELETE /branches/{branchId}`

### L3 — Catalog
- [ ] `PATCH /solutions/{solutionId}`
- [ ] `GET/PATCH/DELETE /plans/{planId}`, `PUT/DELETE /plans/{planId}/benefits/{benefitId}`
- [ ] `GET/PATCH/DELETE /addons/{addonId}`, `PUT/DELETE /addons/{addonId}/benefits/{benefitId}`
- [ ] `PATCH /benefits/{benefitId}`, `GET /benefits/{benefitId}/plans`, `GET /benefits/{benefitId}/addons`

### L4 — Subscription
- [ ] `GET/PATCH /subscriptions/{subscriptionId}`, `POST /subscriptions/{subscriptionId}/reactivate`
- [ ] `PATCH/DELETE /subscriptions/{subscriptionId}/addons/{subscriptionAddonId}`

### L5 — Entitlement & Usage (read)
- [ ] `POST /tenants/{tenantId}/solutions/{solutionId}/entitlements/check`
- [ ] `GET /tenants/{tenantId}/solutions/{solutionId}/usage`, `GET .../usage/{benefitCode}`

### L6 — RBAC
- [ ] `GET/PATCH/DELETE /roles/{roleId}`
- [ ] `GET/PATCH/DELETE /tenants/{tenantId}/roles/{roleId}`, `GET /tenants/{tenantId}/roles/{roleId}/permissions`
- [ ] `POST /solutions/{solutionId}/permissions`, `GET/PATCH /permissions/{permissionId}`
- [ ] `PUT /solutions/{solutionId}/roles/sync`, `GET /solutions/{solutionId}/catalog-version`
- [ ] `GET /tenants/{tenantId}/users/{userId}/role-assignments`, `PATCH/DELETE /role-assignments/{assignmentId}`
- [ ] `GET /tenants/{tenantId}/users/{userId}/effective-permissions`

### L7 — Service Auth & Keys
- [ ] `GET /.well-known/openid-configuration`
- [ ] `GET /admin/signing-keys`, `POST /admin/signing-keys/rotate`
- [ ] `GET /solutions/{solutionId}/clients`, `GET/PATCH /solution-clients/{clientId}`, `POST /solution-clients/{clientId}/revoke`

### L8 — Webhooks & Events
- [ ] `GET /solutions/{solutionId}/webhooks`, `GET/PATCH/DELETE /webhooks/{webhookId}`
- [ ] `POST /webhooks/{webhookId}/rotate-secret`, `POST /webhooks/{webhookId}/test`
- [ ] `GET /webhooks/{webhookId}/deliveries`, `POST /webhook-deliveries/{deliveryId}/retry`
- [ ] `GET /event-types`

### L9 — License
- [ ] `GET /license/history`, `POST /license/activate`, `POST /license/checkin`, `GET /license/checkins`
- [ ] Job check-in berkala di `cmd/worker`

---

## Item Terbuka

- **Policy akses endpoint tenant-scoped untuk user** — saat ini "platform admin atau member aktif". Perlu diputuskan sebelum Fase 15 apakah diperketat memakai RBAC (mis. permission platform untuk kelola member/company).
- **Implementasi `KeyStore` produksi** (KMS / Vault) — dev & test memakai file lokal / fake.
