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

### Fase 3 — Users & Auth Session ✅
- Tabel: `users` (+ `is_platform_admin`, CHECK email lowercase), `auth_sessions` (unique `refresh_token_hash`, index sesi aktif; FK `tenant_id` menyusul Fase 5)
- Domain:
  - `User` (VO `Email` ter-normalisasi, `UserStatus`, `ValidatePassword` 8–72 karakter)
  - `AuthSession` (`EnsureUsable`, `Rotate` dengan masa berlaku sliding, `Revoke` idempotent)
- Usecase:
  - `AuthUseCase`: `Login`, `Refresh`, `Logout`, `Me`, `ResolveUserPrincipal`
  - `UserUseCase`: `Create`, `EnsurePlatformAdmin`
- Keamanan:
  - Email/password salah & email tidak terdaftar → 401 dengan pesan sama + waktu setara (dummy bcrypt)
  - Status user dicek setelah password benar (tidak membocorkan status akun)
  - Refresh token opaque 256 bit, disimpan sebagai SHA-256; rotasi compare-and-swap (`SaveRotation`)
  - Setiap request token user divalidasi ke DB: sesi aktif, user `active`, solution `active`, claim cocok dengan sesi; `IsPlatformAdmin` diisi dari DB
- Endpoint:
  - [x] `POST /auth/login` (MVP; wajib `solution`)
  - [x] `POST /auth/refresh` (MVP)
  - [x] `POST /auth/logout` (MVP)
  - [x] `GET /auth/me` (MVP)
  - [x] `POST /users` (enabler, admin)
- Seeder: `platform-admin` (`PLATFORM_ADMIN_EMAIL`, `PLATFORM_ADMIN_PASSWORD`, `PLATFORM_ADMIN_NAME`), idempotent, tidak menimpa password user lama
- Test: `auth_handler_test.go`, `user_handler_test.go`, entity user & session, repository (termasuk konflik rotasi & kolom INET), seeder. `testAdminTok` kini hasil login sungguhan via seeder
- Catatan:
  - Platform admin login lewat solution aktif mana pun (spec tidak punya solution "console")
  - `ResolveUserPrincipal` = 3 query per request; digabung jadi 1 query bila perlu optimasi
  - Belum ada deteksi pemakaian ulang refresh token lama (reuse detection → revoke sesi) — kandidat L1

### Fase 4 — Solution Clients & Service Auth ✅
- Tabel: `solution_clients` (unique `client_id`, `scopes TEXT[]`, CHECK status, index `solution_id`)
- Domain: `SolutionClient` (`SecretMatches` constant-time, `RotateSecret`, `Revoke`, `GrantableScopes`), VO `Scopes` (whitelist 11 scope spec §2.8)
- Kredensial: `client_id` = `cli_` + 24 hex; secret 256 bit base64url, disimpan SHA-256, tampil sekali
- Endpoint:
  - [x] `POST /solutions/{solutionId}/clients` (MVP, platform admin)
  - [x] `POST /solution-clients/{clientId}/rotate-secret` (MVP, platform admin; `{clientId}` = UUID `id`; client revoked → 409)
  - [x] `POST /oauth/token` (MVP) — RFC 6749 §4.4: form-urlencoded, kredensial via HTTP Basic **atau** body (tidak boleh keduanya), parameter `scope` opsional (subset), respons/error format OAuth2 tanpa envelope, `Cache-Control: no-store`, `WWW-Authenticate` saat Basic gagal
- Authenticator: token service divalidasi ke DB tiap request (`ResolveServicePrincipal`): client `active`, solution `active`, claim `sol` cocok, scope di-intersect dengan scope client terkini (revoke/pengurangan scope langsung berlaku)
- Test: `solution_client_handler_test.go`, `oauth_handler_test.go`, entity, usecase (`ResolveServicePrincipal` dengan repo fake), repository (`TEXT[]` round trip)
- Test infra: `TestServer.POSTForm`, helper `mustCreateClient`, `mustServiceToken`

### Fase 5 — Tenant & Membership ✅
- Tabel: `tenants` (unique slug, CHECK status), `tenant_members` (PK `(tenant_id, user_id)`, CHECK status, index `user_id`), FK `auth_sessions.tenant_id` → `tenants(id)`
- Domain:
  - `Tenant` (VO `Slug`), `TenantMember` (`NewTenantMember`, `Rejoin` untuk member inactive)
  - Domain service `AccessPolicy.EnsureCanAccess`: platform admin bebas; user lain wajib `tid` = tenantId **dan** member `active` (spec §2.7). `tid` kosong → 403 `AUTH_TENANT_NOT_SELECTED`
  - Event `tenant_member.added` (payload tenant_id, user_id, status, joined_at)
- Keputusan:
  - Semua user login boleh membuat tenant; pembuat non-admin otomatis member active. Platform admin tidak otomatis jadi member
  - `POST /tenants` & `GET /tenants` boleh tanpa `tid` (onboarding / pemilihan tenant); endpoint per-tenant wajib `tid`
  - Tambah member via `user_id` **atau** `email` (user harus sudah ada), status `active` (default) / `invited`; member inactive diaktifkan ulang; sudah active/invited → 409
  - Endpoint khusus token user; akses token service (`users:read` / `users:write`) menyusul Fase 10 (butuh cek subscription)
- Endpoint:
  - [x] `POST /tenants` (MVP)
  - [x] `GET /tenants` (MVP) — admin: semua; user: tenant tempat ia member active
  - [x] `GET /tenants/{tenantId}` (MVP)
  - [x] `GET /tenants/{tenantId}/members` (MVP) — join data user, filter status/search, pagination
  - [x] `POST /tenants/{tenantId}/members` (MVP)
- Test: `tenant_handler_test.go`, `tenant_member_handler_test.go`, entity, `access_policy_test.go`, repository
- Test infra: `tenantToken(userID, tenantID)` (FakeAuthenticator, sementara), `mustCreateTenant`, `mustTenantOwner`, `countEvents`
- Lainnya: pencarian `ILIKE` di query model kini meng-escape `%` / `_` (`likePattern`)

### Fase 6 — Company & Branch ✅
- Tabel: `companies`, `branches` (soft delete `deleted_at`, partial unique `(tenant_id, code)` / `(company_id, code)` `WHERE deleted_at IS NULL`, index FK, CHECK status)
- Domain `organization`: `Company`, `Branch`, VO `OrgCode` (huruf/angka/`.`/`_`/`-`, dinormalisasi UPPERCASE), `OrgStatus`; event `company.created`, `branch.created` (payload memuat `tenant_id` & `updated_at`)
- Akses:
  - `/tenants/{tenantId}/companies` → `AccessPolicy` tenant
  - `/companies/{companyId}/branches` → company dimuat lalu `AccessPolicy` terhadap tenant pemiliknya; company tenant lain / sudah dihapus → **404** (tidak membocorkan keberadaan). `tid` kosong tetap 403
  - Khusus token user; akses token service (`org:read` / `org:write`) menyusul Fase 10
- Endpoint:
  - [x] `GET /tenants/{tenantId}/companies` (MVP) — tanpa yang terhapus, filter status/search, pagination
  - [x] `POST /tenants/{tenantId}/companies` (MVP)
  - [x] `GET /companies/{companyId}/branches` (MVP)
  - [x] `POST /companies/{companyId}/branches` (MVP)
- Test: `company_handler_test.go`, `branch_handler_test.go`, entity, repository (duplikat, soft delete, reuse code)
- Test infra: `mustCreateCompany`, `principalUser`

### Fase 7 — Benefits (enabler) ✅
- Tabel: `benefits` (unique `(solution_id, code)`, `UNIQUE (id, solution_id)` target composite FK, CHECK 4 enum, index `solution_id`)
- Domain `benefit`: entity `Benefit`, VO `BenefitCode` (snake_case), `ValueType`, `AggregationStrategy`, `Metering`, `UsageReset`
- Aturan kombinasi (keputusan Fase 7, spec hanya mendaftar enum):
  - `boolean`: `any_true` / `override`; `integer` & `decimal`: `sum` / `max` / `min` / `override`; `string` & `json`: `override`
  - `metering` `counter` / `gauge` hanya untuk `integer` / `decimal`
  - `usage_reset` selain `never` hanya untuk `counter` (gauge tidak di-reset)
- `Benefit.ValidateValue(json)` — validasi nilai plan/addon benefit per `value_type` (integer harus bulat, json harus object/array); dipakai Fase 8–9
- Akses: create = platform admin + `LICENSE_MANAGED` di on_prem; baca = token valid apa pun, solution `inactive` disembunyikan (404) dari non-admin
- Endpoint:
  - [x] `POST /solutions/{solutionId}/benefits` (enabler)
  - [x] `GET /solutions/{solutionId}/benefits` (enabler) — filter `value_type`, `metering`, `search`; default 50/halaman
  - [x] `GET /benefits/{benefitId}` (enabler)
- Test: `benefit_handler_test.go`, `benefit_test.go` (matriks kombinasi & ValidateValue), repository
- Test infra: `mustCreateBenefit`
- Belum: representasi "unlimited" untuk limit angka (mis. `-1` / `null`) — diputuskan saat EntitlementCalculator (Fase 12)

### Fase 8 — Plans & Plan Benefits ✅
- Tabel: `plans` (soft delete, partial unique `(solution_id, code)`, `UNIQUE (id, solution_id)`, CHECK billing_cycle & `price >= 0`), `plan_benefits` (PK `(plan_id, benefit_id)`, composite FK ke `plans(id, solution_id)` & `benefits(id, solution_id)`, index `benefit_id`)
- Domain:
  - `pricing` (dipakai plan, add-on, subscription): VO `Price` (NUMERIC(18,2) sebagai string, tanpa float), `Currency` (ISO 4217, default IDR), `BillingCycle`
  - `plan`: entity `Plan`, `PlanBenefit`, VO `PlanCode`
  - `benefit/service.AssignmentValidator` (dipakai juga add-on): benefit unik, ada, satu solution, nilai cocok `value_type`
- Harga: request menerima angka atau string angka (maks 2 desimal, tanpa notasi eksponen); respons selalu string (`"150000.00"`)
- Akses: tulis = platform admin + `LICENSE_MANAGED`; baca = token valid, non-admin hanya plan `is_active` dari solution aktif (selain itu 404)
- Endpoint:
  - [x] `POST /solutions/{solutionId}/plans` (MVP)
  - [x] `GET /solutions/{solutionId}/plans` (MVP) — default urut harga termurah, filter `is_active` (admin) / `billing_cycle` / `search`
  - [x] `GET /plans/{planId}/benefits` (MVP)
  - [x] `PUT /plans/{planId}/benefits` (MVP) — replace-all dalam satu transaksi, idempotent, array kosong = hapus semua
- Tidak ada event outbox (perubahan plan tidak memengaruhi subscription berjalan)
- Test: `plan_handler_test.go`, pricing VO, plan entity, `assignment_validator_test.go`, repository (NUMERIC round trip, composite FK menolak benefit lintas solution)
- Bug yang ditemukan test: `ORDER BY price` merujuk kolom output `price::text` → urut teks; diperbaiki dengan kolom ter-qualify (`plans.price`)

### Fase 9 — Add-ons & Add-on Benefits ✅
- Tabel: `addons` (pola sama `plans`: soft delete, partial unique code, `UNIQUE (id, solution_id)`, NUMERIC price), `addon_benefits` (composite FK ke `addons` & `benefits`, index `benefit_id`)
- Domain `addon`: entity `Addon`, `AddonBenefit`, VO `AddonCode`; memakai ulang `pricing` VO dan `AssignmentValidator`
- `pricing.ParsePriceJSON` — parsing harga JSON (angka / string) dipindah ke domain pricing, dipakai plan & add-on
- Akses identik dengan plan (tulis = admin + `LICENSE_MANAGED`; baca = token valid, non-admin hanya add-on aktif dari solution aktif)
- Endpoint:
  - [x] `POST /solutions/{solutionId}/addons` (MVP)
  - [x] `GET /solutions/{solutionId}/addons` (MVP)
  - [x] `GET /addons/{addonId}/benefits` (MVP)
  - [x] `PUT /addons/{addonId}/benefits` (MVP) — replace-all
- Test: `addon_handler_test.go`, entity, repository (composite FK), `ParsePriceJSON`
- Test infra: `mustCreateAddon`

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
- Buka akses token service ke endpoint tenant & member (`users:read` / `users:write`) serta company & branch (`org:read` / `org:write`) untuk tenant yang subscribe solution client (perluas `AccessPolicy`)
- Ganti helper test `tenantToken` (FakeAuthenticator) dengan token hasil `POST /auth/switch-tenant`
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
