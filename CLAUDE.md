# CLAUDE.md — platform-service

Panduan ini dibaca otomatis oleh Claude Code setiap sesi. Isinya arsitektur, konvensi, dan aturan pengembangan. Diadaptasi dari `k-forum-api`.

- Spec domain & API: `docs/central-saas-platform-spec.md`
- Roadmap fase: `docs/PHASES.md`

---

## 0. Teori Umum DDD (Ringkas)

- **Ubiquitous Language**: istilah domain di spec, diskusi, dan kode harus sama (tenant, solution, plan, add-on, benefit, entitlement, snapshot, dst).
- **Bounded Context**: batas model domain agar istilah dan aturan tidak bercampur.
- **Entity**: objek domain dengan identitas dan siklus hidup.
- **Value Object**: objek tanpa identitas, dibandingkan berdasarkan nilai.
- **Aggregate**: kumpulan objek domain dengan satu aggregate root sebagai pintu perubahan.
- **Repository**: abstraksi akses aggregate ke storage.
- **Domain Service**: operasi domain yang tidak pas di satu entity.
- **Usecase**: orkestrasi alur bisnis lintas domain object dan port.

Domain model berisi rule bisnis inti. Usecase mengatur alur. Infrastruktur hanya detail teknis.

---

## 1. Stack & Layout

- Go 1.25, Gin, `database/sql` + `pgx/v5/stdlib` (SQL ditulis manual, tanpa ORM), golang-migrate (file di-embed), go-playground/validator (tag `binding`), golang-jwt/v5 (RS256), testify, testcontainers-go.
- PostgreSQL 16 (minimal 15 untuk `UNIQUE NULLS NOT DISTINCT`).

```
cmd/{app,migrate,seeder,worker}/main.go
internal/
  bootstrap/                 # wiring repo → usecase → handler → router (dipakai cmd/app & testhelper)
  config/ logger/ migrations/ seeders/ testhelper/
  domain/<ctx>/{entity,valueobject,constant,event,repository,service}/
  domain/errors/             # domainerr
  app/
    apperror/ dto/ port/
    usecase/<ctx>/           # <ctx>_usecase.go (semua usecase domain), helpers.go
  infrastructure/
    persistence/             # postgres_<ctx>_repository.go, postgres_<ctx>_query.go, postgres_transactor.go
    crypto/ keystore/ outbox/
  interfaces/http/
    handler/api/             # <ctx>_handler.go + <ctx>_handler_test.go
    middleware/ respond/ httperror/ httpquery/
    router/                  # router.go + <ctx>_routes.go (RegisterXxxRoutes)
locales/{en,id}.json
```

---

## 2. Alur Standar Pembuatan Endpoint

```
HTTP Handler -> Usecase -> Domain (Repository Interface / Entity / Domain Service) -> Persistence
```

Langkah membuat endpoint baru:

1. Migrasi (`make migrate-create NAME=...`) bila ada tabel/kolom baru.
2. Domain: entity/VO + error code di `domain/<ctx>/constant` + interface repository.
3. Persistence: implementasi repository (dan query model bila perlu).
4. Usecase: tambahkan method baru di `<Ctx>UseCase` pada `usecase/<ctx>/<ctx>_usecase.go` (tambah dependency di `<Ctx>Deps` bila perlu).
5. DTO request/response di `app/dto/<ctx>_dto.go`.
6. Handler + swag annotation (`@Router` pakai path penuh, mis. `/api/v1/tenants [post]`), lalu:
   - tambah field handler di `router.Handlers`,
   - buat `router/<ctx>_routes.go` berisi `RegisterXxxRoutes(r Routes, h *api.XxxHandler)` dan panggil dari `registerDomainRoutes`,
   - wiring repo → usecase → handler di `internal/bootstrap/bootstrap.go` (`bootstrap.New` → `App`). Tugas startup (mis. `EnsureActiveKey`) juga dijalankan di sana.
7. Handler test (§9). Endpoint tanpa test = belum selesai.
8. Centang endpoint di `docs/PHASES.md`.

---

## 3. Tanggung Jawab Tiap Layer

### Handler
- Terima request, parsing & validasi dasar (`ShouldBindJSON` / `ShouldBindQuery` / `ShouldBindUri`).
- Ambil principal dari context, panggil **satu** usecase.
- Respons via `respond.OK` / `respond.Created` / `respond.SuccessWithMeta`; error via `httperror.Handle(c, err)`.
- Tidak berisi logika bisnis dan tidak akses DB.

### Usecase
- Orkestrator proses bisnis; jalankan rule lewat method domain atau domain service, bukan if/else panjang.
- **Usecase dikelompokkan per domain**, bukan satu file per usecase. Satu domain = satu struct `<Ctx>UseCase` dengan satu method per operasi:

  ```go
  // internal/app/usecase/tenant/tenant_usecase.go
  type TenantDeps struct {
      Tx         port.Transactor
      TenantRepo tenantrepo.TenantRepository
      MemberRepo tenantrepo.TenantMemberRepository
      Query      port.TenantQueryReadModel
      Outbox     port.EventOutbox
  }

  type TenantUseCase struct{ deps TenantDeps }

  func NewTenantUseCase(deps TenantDeps) *TenantUseCase { return &TenantUseCase{deps: deps} }

  // Required — auth: user | scope: -
  func (uc *TenantUseCase) Create(ctx context.Context, actor *principal.Principal, req dto.CreateTenantRequest) (*dto.TenantResponse, error)
  // Required — auth: user (member / platform admin) | scope: -
  func (uc *TenantUseCase) List(ctx context.Context, actor *principal.Principal, q dto.ListTenantQuery) ([]dto.TenantItem, dto.Meta, error)
  ```
- Bila satu file domain terlalu besar (> ~500 baris) atau domain punya beberapa aggregate, boleh dipecah per sub-domain dalam paket yang sama, mis. `tenant_usecase.go` + `member_usecase.go` (`TenantUseCase`, `MemberUseCase`). Jangan kembali ke satu file per operasi.
- Helper bersama paket (`mapXxxDomainError`, mapper domain → DTO) di `helpers.go`.
- Tidak berisi SQL atau detail driver DB.
- **Wajib** memetakan domain object ke DTO sebelum return. Handler tidak boleh menerima domain object.
- Beri komentar kebutuhan akses di atas tiap method usecase, contoh: `// Required — auth: user (platform admin) | scope: -`.
- Perubahan data yang memicu event: tulis ke outbox **di dalam transaksi yang sama** (`Transactor.WithTx`).

### Domain
- Entity, value object, domain service, error code, interface repository.
- Tidak boleh import `infrastructure` atau `interfaces`.

### Domain Service
- Untuk rule yang melibatkan beberapa entity/aggregate atau butuh data repository.
- Boleh bergantung pada interface repository domain, tidak pada implementasinya.
- Contoh di proyek ini: `SnapshotBuilder`, `EntitlementCalculator`, `ScopeValidator`, `PermissionEvaluator`, `LicenseVerifier`.

### Persistence
- Implementasi interface repository; mapping row ↔ domain object.
- Semua query lewat `execFromContext(ctx, r.db)` agar ikut transaksi di context.

---

## 4. Dependency Rule (Wajib)

```
interfaces -> app/usecase -> domain
infrastructure -> domain
```

Dilarang:
- Domain import infrastructure.
- Usecase import implementasi repository langsung.
- Handler akses DB tanpa usecase.

---

## 5. Repository & Query Model

- Interface repository: `internal/domain/<ctx>/repository`. Port aplikasi (query model, Transactor, EventOutbox, KeyStore, dst): `internal/app/port`.
- Implementasi: `internal/infrastructure/persistence`.
  - CRUD aggregate (Save/Update/FindByID/Delete/list sederhana) → `postgres_<ctx>_repository.go`.
  - Listing dengan pagination/filter/sort/join kompleks → `postgres_<ctx>_query.go`.
- Query model hanya dipakai bila benar-benar butuh pagination atau listing kompleks. Get by ID / unique field tetap lewat repository.

---

## 6. Error Mapping

Tiga tingkat:

1. **Domain** — `domainerr.New(code)` / `domainerr.Wrap(code, err)`. Code berformat `DOMAIN_<CTX>_<THING>`, didefinisikan di `domain/<ctx>/constant`, sekaligus key terjemahan di `locales/*.json`.
2. **Usecase** — `mapXxxDomainError(err)` → `apperror.BadRequest/Unauthorized/Forbidden/NotFound/Conflict/Unprocessable/Internal`.
3. **HTTP** — `httperror.Handle(c, err)` memetakan AppError ke status. Error validator → 422.

Persistence **wajib**:
- `sql.ErrNoRows` → `domainerr.New(constant.CodeXxxNotFound)`
- `pgconn.PgError` code `23505` → `domainerr.New(constant.CodeXxxDuplicateKey)`
- `23503` (FK, mis. composite FK lintas solution) → code domain yang sesuai (mis. `CodeXxxSolutionMismatch`), bukan error mentah
- Gagal INSERT/UPDATE/DELETE → `domainerr.Wrap(constant.CodeXxxPersistenceFailed, err)`
- Gagal SELECT → `domainerr.Wrap(constant.CodeXxxQueryFailed, err)`

Pakai helper di `persistence/pg_errors.go` (`isUniqueViolation`, `isForeignKeyViolation`, `isCheckViolation`, `constraintName`):

```go
if isUniqueViolation(err) {
    return domainerr.New(constant.CodeXxxDuplicateKey)
}
return domainerr.Wrap(constant.CodeXxxPersistenceFailed, err)
```

Persistence tidak pernah mengembalikan AppError.

Envelope respons:
- Sukses: `{status: "success", status_code, message, data, meta}`
  - Pengecualian: dokumen standar yang dibaca library pihak ketiga dikirim tanpa envelope — JWKS, openid-configuration, dan `POST /oauth/token` (respons & error RFC 6749 via `dto.OAuthError`).
- Error: `{status: "error", status_code, error_code, errors}` — `errors` berisi pesan terjemahan dari message key (string) atau map field → pesan untuk validasi.

Message key lintas domain (auth, dsb.) ada di `apperror` (`MsgTokenMissing`, `MsgInsufficientScope`, ...). Setiap key baru wajib ditambahkan ke **kedua** `locales/en.json` dan `locales/id.json`.

---

## 7. Pagination

```go
// app/dto/pagination_dto.go
type PaginationParams struct {
    Page     *int
    Limit    *int
    SortBy   *string
    SortType *string
}

type Meta struct {
    CurrentPage int64 `json:"current_page"`
    PerPage     int64 `json:"per_page"`
    Total       int64 `json:"total"`
    TotalPages  int64 `json:"total_pages"`
}
```

- DTO query list: field pagination pointer dengan tag `form:"page"`, `form:"limit"`, `form:"sort_by"`, `form:"sort_type"`.
- Usecase list **wajib** mengembalikan tiga nilai: `([]dto.XxxItem, dto.Meta, error)`. Jangan bungkus data + meta dalam satu struct.
- Persistence pakai `resolvePaginationParams(...)` untuk normalisasi limit/offset/sort (whitelist kolom sort).
- Handler list pakai `respond.SuccessWithMeta`.

---

## 8. Aturan Khusus Platform

### 8.1 Konsistensi solution (composite FK)
Tabel `plans`, `addons`, `benefits`, `subscriptions`, `permissions`, `roles` punya `UNIQUE (id, solution_id)`. Tabel relasi menyimpan `solution_id` dan FK ke `(xxx_id, solution_id)`. Jangan hilangkan kolom/constraint ini demi "menyederhanakan" — DB yang menolak relasi lintas solution. Usecase tetap memvalidasi lebih awal agar pesan error jelas.

### 8.2 Soft delete
- `companies`, `branches`, `plans`, `addons`, `roles`: set `deleted_at`; semua query default `WHERE deleted_at IS NULL`.
- `tenant_members`: `status = 'inactive'` + hapus role assignment user di tenant itu.
- `subscription_addons`: `status = 'canceled'` + hapus snapshot benefit add-on.

### 8.3 Snapshot benefit
Entitlement **selalu** dihitung dari `subscription_benefit_snapshots`, bukan dari `plan_benefits` / `addon_benefits` langsung. Perubahan plan/add-on master tidak memengaruhi subscription berjalan.

Perhitungan memakai `entitlement/service.Calculate` + `Resolver` — jangan menghitung ulang di usecase lain. Nilai benefit angka `-1` = unlimited; aritmetika angka pakai `big.Rat` (tanpa float).

### 8.4 Outbox
Setiap perubahan yang ada di katalog event (spec §2.9) menulis baris `events` di transaksi yang sama dengan perubahan data, lewat `port.EventOutbox.Append`. Implementasinya menolak (`ErrOutboxRequiresTx`) bila dipanggil di luar `Transactor.WithTx`. Jangan kirim webhook langsung dari usecase.

### 8.5 Autentikasi & otorisasi
- Token user: `typ=user`, `aud = solutions.code`, `tid` boleh kosong. Token service: `typ=service`, `aud=platform`, `sol`, `scope`.
- Endpoint per-tenant (path `/tenants/{tenantId}/...`) wajib lewat domain service `tenant/service.AccessPolicy.EnsureCanAccess` di usecase: platform admin bebas; token service hanya untuk tenant yang subscribe (`trialing`/`active`) solution client-nya; user lain wajib `tid` = tenantId dan member `active`. Route-nya memakai `RequireUserOrService(scope)` (users:*, org:*, entitlements:read). Endpoint identitas & pemilihan tenant (`/auth/*`, `GET/POST /tenants`) boleh tanpa `tid`.
- Resource anak tenant yang diakses lewat ID-nya sendiri (mis. `/companies/{companyId}`): muat resource, cek `AccessPolicy` terhadap tenant pemiliknya, dan laporkan **404** bila akses ditolak (jangan bocorkan keberadaan data tenant lain). `tid` kosong tetap 403.
- Administrasi tenant (member, company/branch, role custom, role assignment) terbuka untuk semua member aktif — platform tidak menegakkan RBAC di sini; RBAC dievaluasi lewat `authz/check` (`authz/service.Evaluate`, hierarki scope tenant ⊇ company ⊇ branch).
- Service client hanya boleh akses data `solution_id`-nya dan tenant yang subscribe solution tersebut. Data per-solution (subscription, entitlement, usage) wajib difilter ke `principal.SolutionID` untuk token service.
- Subscription: maksimal satu yang live per tenant per solution; "usable" = `trialing`/`active` (`valueobject.UsableStatuses`).
- Endpoint idempotent (mis. usage): insert record ber-unique key di dalam transaksi; unique violation → baca record asli di luar transaksi, payload sama = replay (200), beda = 409. Penghitung memakai upsert atomik (`used = used + EXCLUDED.used`), bukan read-modify-write.
- Platform admin = `users.is_platform_admin` → `RequirePlatformAdmin`.
- Middleware (`interfaces/http/middleware/auth.go`): `Authenticate(port.TokenAuthenticator)` menyimpan `*principal.Principal` di gin context (ambil dengan `middleware.PrincipalFrom(c)`), lalu guard `RequireUser`, `RequireService(scope)`, `RequireUserOrService(scope)`, `RequirePlatformAdmin`, `RequireActiveTenant`. Handler meneruskan principal ke usecase; usecase tidak membaca gin context.
- Route terproteksi didaftarkan di `Routes.Authenticated` (sudah memasang `Authenticate`); route publik di `Routes.V1`.
- Token JWT RS256 diterbitkan lewat `port.TokenIssuer` dan diverifikasi `jwttoken.Authenticator` (key dari `signing_keys` + `port.KeyStore`). Private key tidak pernah disimpan di DB maupun di-log.
- Token user selalu divalidasi ulang ke DB oleh `AuthUseCase.ResolveUserPrincipal` (dirangkai di `bootstrap`): sesi aktif, user & solution `active`, `tid` sama dengan sesi. Jangan mempercayai `IsPlatformAdmin` dari sumber lain.
- Token service juga divalidasi ulang ke DB oleh `SolutionClientUseCase.ResolveServicePrincipal`: client & solution `active`, `sol` cocok, scope di-intersect dengan scope client terkini. Guard `RequireService(scope)` karenanya selalu memakai scope terbaru.
- Pesan error login / token endpoint tidak boleh membedakan "akun/client tidak ada" dan "password/secret salah".
- Secret (client secret, refresh token) hanya disimpan sebagai hash; webhook secret dienkripsi. Secret mentah hanya ditampilkan sekali di respons create/rotate.

### 8.6 Deployment mode
- `DEPLOYMENT_MODE` dibaca dari config (`cloud` / `on_prem`).
- Endpoint yang dikelola license (spec §2.14) dibungkus `routes.Deployment.LicenseManaged()` → 403 `LICENSE_MANAGED` di `on_prem`.
- Endpoint license/installation dibungkus `routes.Deployment.OnPremOnly()` → 404 di `cloud`.
- Saat `license_state` `read_only` / `invalid`, operasi tulis data bisnis ditolak.

---

## 9. Handler Test: Wajib Ada, Wajib Dicek

### Aturan utama
1. Menambah/mengubah handler → cek `<ctx>_handler_test.go` di direktori yang sama. Belum ada → buat. Sudah ada → tambah/perbarui test yang relevan.
2. Endpoint baru belum dianggap selesai sebelum handler test-nya hijau.
3. Saat review/debug handler → jalankan test sebelum dan sesudah perubahan.

### Gaya
- **Black-box integration test, tanpa mock usecase/repository.** Server dibangun lengkap (repo Postgres nyata via testcontainers); hanya adapter eksternal yang diganti fake di `testhelper/noop.go`.
- Package `api_test` di `internal/interfaces/http/handler/api/`.
- Satu fungsi per skenario: `Test<Domain>_<Endpoint>_<Scenario>`, contoh `TestTenant_CreateMember_Forbidden`. Pisahkan grup endpoint dengan banner komentar `// ── Members ──────`.
- `require` untuk setup/prasyarat, `assert` untuk pengecekan. Decode respons ke anonymous struct `struct{ Data struct{...} \`json:"data"\` }`.
- Data unik per test (`uniqueEmail`, `uniqueCode`, `uniqueSlug`) — DB dipakai bersama tanpa truncate.
- Efek samping (outbox event, snapshot, soft delete) dicek lewat SQL langsung: `testSrv.DB.QueryRowContext(...)`.
- Helper setup via API diberi prefix `must` dan memanggil `t.Helper()`.

```go
func TestTenant_Create_Unauthenticated(t *testing.T) {
	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": "Acme", "slug": uniqueSlug("acme")})
	assert.Equal(t, 401, w.Code)
}

func TestTenant_Create_Success(t *testing.T) {
	token := mustLoginNewUser(t)
	slug := uniqueSlug("acme")

	w := testSrv.POST("/api/v1/tenants", map[string]any{"name": "Acme", "slug": slug},
		testhelper.BearerHeader(token))
	require.Equal(t, 201, w.Code, "create tenant: %s", w.Body.String())

	var resp struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, slug, resp.Data.Slug)
}
```

### Skenario wajib per endpoint
- `Unauthenticated` → 401 untuk semua endpoint terproteksi
- `Forbidden` → 403 (bukan platform admin / bukan member / scope service kurang / tenant tidak subscribe / `LICENSE_MANAGED`)
- `InvalidPayload` → 400/422 untuk endpoint ber-body (missing required dari Gin = 422; boleh `assert.True(t, w.Code == 400 || w.Code == 422)`)
- `NotFound` → 404 untuk endpoint by ID
- `Duplicate` / `Conflict` → 409 bila ada unique constraint
- `Success` → 2xx + assert isi `data` (dan `meta` untuk list)

### Infrastruktur test
```
internal/testhelper/
  testdb.go       → MustStartTestDB (postgres:16-alpine + semua migrasi)
  testserver.go   → MustStartTestServer(opts...) [WithDeploymentMode, WithAuthenticator], TestConfig,
                    GET/POST/PUT/PATCH/DELETE/DELETEBody, POSTForm, BearerHeader, DecodeBody, DecodeError
                    memakai internal/bootstrap + router.Setup yang sama dengan cmd/app (jangan duplikasi route)
                    TestServer.MustDerive(opts...) → engine lain di DB yang sama (mis. mode on_prem)
                    TestServer.Issuer → menerbitkan JWT asli; TestServer.KeyStore (memory)
                    Authenticator test = FakeAuthenticator dulu, lalu verifier JWT
  fake_auth.go    → FakeAuthenticator: UserToken, PlatformAdminToken, ServiceToken, Issue(principal)
  seed.go         → UniqueCode, UniqueEmail, MustSeedSolution, MustSeedSigningKey, SetUserStatus, ... (insert langsung ke DB)
  TestPasswordHasher() → bcrypt cost minimum (dipakai TestServer & seeder di test)
  txhelper.go     → WithTestTx (selalu rollback)
  assert.go       → AssertDomainError
  (ditambah saat dibutuhkan fase domain)
  fixtures.go     → data statis (mis. license test yang ditandatangani test key)
  noop.go         → fake KeyStore, fake clock, fake webhook sender

internal/interfaces/http/handler/api/
  main_test.go      → TestMain, var testSrv, testOnPremSrv, testAdminTok (login admin sungguhan via seeder)
  mustTenantOwner   → token ASLI ber-tid lewat alur login → tenant → subscribe → switch-tenant
  testutil_test.go  → adminHeader, userHeader (fake), mustCreateSolution, mustCreateUser, mustLogin,
                      mustLoginNewUser, mustCreateClient, mustServiceToken, mustCreateTenant,
                      mustTenantOwner, tenantToken (fake, kasus khusus), principalUser, mustCreateCompany,
                      mustCreateBenefit, mustCreatePlan, mustCreateAddon, mustSubscribe, mustSwitchTenant,
                      countEvents, ...
                      (helper lewat API, prefix must)
  Catatan: FakeAuthenticator.ServiceToken tidak melewati validasi client di DB — untuk skenario
  yang bergantung pada status/scope client, pakai mustServiceToken (token asli).
```

### Test lain
- Domain service dengan logika murni (`EntitlementCalculator`, `PermissionEvaluator`, state license) → unit test table-driven di `internal/domain/<ctx>/service/*_test.go`, fake ditulis di file test.
- Repository/transactor → `internal/infrastructure/persistence/*_test.go` (package `persistence_test`, `TestMain` dengan `MustStartTestDB`, global `testDB`). Helper unexported diuji di `*_internal_test.go` (package `persistence`).
- Middleware & httperror → unit test dengan `gin.New()` + `httptest` (lihat `middleware/auth_test.go`).

### Menjalankan test
```bash
go test ./internal/interfaces/http/handler/api/... -count=1 -timeout 300s
go test ./internal/interfaces/http/handler/api/ -run "TestTenant" -v -timeout 120s
go test ./... -count=1
# atau via Makefile
make test-handler RUN=TestTenant
make test            # semua
make lint            # golangci-lint v2 via Docker
```
Docker harus jalan (testcontainers).

### Nuance
- Jangan menamai kolom output sama dengan kolom yang dipakai `ORDER BY` bila tipenya berubah (mis. `price::text`): PostgreSQL mengutamakan nama kolom output → urutan jadi teks. Qualify kolom sort dengan nama/alias tabel (`plans.price`).
- Test repository di dalam `WithTestTx` yang sengaja memicu error SQL (mis. unique violation) wajib dibungkus `SAVEPOINT` / `ROLLBACK TO SAVEPOINT` bila masih ada statement setelahnya — Postgres membatalkan seluruh transaksi setelah satu error.
- `http.NewRequest` di testserver membuat `RemoteAddr` kosong → `c.ClientIP()` = `""`. Untuk kolom `INET` gunakan `NULLIF($n, '')::inet`.

---

## 10. Penamaan File Migrasi (Wajib Timestamp)

```bash
make migrate-create NAME=create_tenants
# internal/migrations/20261014120000_create_tenants.up.sql
# internal/migrations/20261014120000_create_tenants.down.sql
```

- Jangan buat file migrasi manual dengan nomor urut; selalu pakai `make migrate-create`.
- `NAME` pakai `snake_case` deskriptif.
- Setiap `up` wajib punya `down` yang benar-benar membalikkan perubahan.
- Seluruh kolom enum diberi `CHECK (kolom IN (...))` sesuai spec §2.0; index sesuai spec §2.11.

Alasan: nomor urut dari dua branch berbeda bisa bertabrakan, dan golang-migrate bisa diam-diam melewatkan migrasi yang versinya dianggap sudah diterapkan.
