# Central SaaS Platform — Domain Entity & API Specification

| Item | Nilai |
|---|---|
| Database | PostgreSQL (15+, untuk `UNIQUE NULLS NOT DISTINCT`) |
| API Style | REST |
| Base URL | `/api/v1` |
| Default Currency | `IDR` |
| Deployment Mode | `cloud` (SaaS) atau `on_prem` (dipasang di server client) — lihat bagian 2.14 |

## Daftar Isi

1. [Ringkasan Domain](#1-ringkasan-domain)
2. [Domain Entity Breakdown](#2-domain-entity-breakdown)
3. [API Specification](#3-api-specification)
4. [Endpoint Minimum untuk MVP](#4-endpoint-minimum-untuk-mvp)
5. [Prinsip Utama](#5-prinsip-utama)
6. [Changelog](#6-changelog)

---

## 1. Ringkasan Domain

| Domain | Entity |
|---|---|
| Identity & Organization | `users`, `tenants`, `tenant_members`, `companies`, `branches` |
| Solution Catalog | `solutions` |
| Plans & Add-ons | `plans`, `addons` |
| Benefits | `benefits`, `plan_benefits`, `addon_benefits` |
| Subscriptions | `subscriptions`, `subscription_addons`, `subscription_benefit_snapshots` |
| Authorization | `permissions`, `roles`, `role_permissions`, `user_role_assignments` |
| Auth & Session | `auth_sessions`, `signing_keys` |
| Solution Integration | `solution_clients`, `webhook_endpoints`, `events`, `webhook_deliveries` |
| Usage & Metering | `usage_records`, `tenant_usage` |
| Licensing (khusus `on_prem`) | `installations`, `licenses`, `license_checkins` |

### Relasi Utama

```text
tenants ──< tenant_members >── users
tenants ──< companies ──< branches

solutions ──< plans ──< plan_benefits >── benefits
solutions ──< addons ──< addon_benefits >── benefits
solutions ──< benefits

tenants ──< subscriptions >── solutions
subscriptions >── plans
subscriptions ──< subscription_addons >── addons
subscriptions ──< subscription_benefit_snapshots >── benefits
subscription_addons ──< subscription_benefit_snapshots

solutions ──< permissions
solutions ──< roles ──< role_permissions >── permissions
tenants ──< roles                         (custom role, tenant_id NOT NULL)
tenant_members ──< user_role_assignments >── roles
user_role_assignments ── scope (tenant | company | branch)

users ──< auth_sessions >── solutions
auth_sessions >── tenants                 (tenant aktif, boleh NULL sebelum pilih tenant)

solutions ──< solution_clients
solutions ──< webhook_endpoints ──< webhook_deliveries >── events

tenants ──< usage_records >── benefits
tenants ──< tenant_usage >── benefits

installations ──< licenses ──< license_checkins     (khusus on_prem)
```

### Pola Composite FK (Konsistensi Solution)

Tabel yang menjadi target memiliki `UNIQUE (id, solution_id)`. Tabel relasi menyimpan `solution_id` dan mereferensikan `(xxx_id, solution_id)`, sehingga DB menolak relasi lintas solution.

| Tabel Target | Constraint Tambahan |
|---|---|
| `plans` | `UNIQUE (id, solution_id)` |
| `addons` | `UNIQUE (id, solution_id)` |
| `benefits` | `UNIQUE (id, solution_id)` |
| `subscriptions` | `UNIQUE (id, solution_id)` |
| `permissions` | `UNIQUE (id, solution_id)` |
| `roles` | `UNIQUE (id, solution_id)` |

---

## 2. Domain Entity Breakdown

### 2.0 Nilai Enum (CHECK Constraint)

| Kolom | Nilai yang Diizinkan |
|---|---|
| `users.status` | `active`, `inactive`, `suspended` |
| `tenants.status` | `active`, `suspended`, `closed` |
| `tenant_members.status` | `invited`, `active`, `inactive` |
| `companies.status`, `branches.status` | `active`, `inactive` |
| `solutions.status` | `active`, `inactive` |
| `plans.billing_cycle`, `addons.billing_cycle`, `subscriptions.billing_cycle` | `monthly`, `yearly` |
| `benefits.value_type` | `boolean`, `integer`, `decimal`, `string`, `json` |
| `benefits.aggregation_strategy` | `sum`, `max`, `min`, `override`, `any_true` |
| `subscriptions.status` | `trialing`, `active`, `past_due`, `canceled`, `expired` |
| `subscription_addons.status` | `active`, `canceled` |
| `user_role_assignments.scope_type` | `tenant`, `company`, `branch` |
| `permissions.status` | `active`, `deprecated` |
| `benefits.metering` | `none`, `counter`, `gauge` |
| `benefits.usage_reset` | `never`, `billing_period`, `monthly` |
| `solution_clients.status` | `active`, `revoked` |
| `signing_keys.status` | `active`, `retiring`, `retired` |
| `webhook_endpoints.status` | `active`, `disabled` |
| `webhook_deliveries.status` | `pending`, `success`, `failed`, `dead` |
| `installations.deployment_mode` | `cloud`, `on_prem` |
| `installations.license_state` | `valid`, `grace`, `read_only`, `invalid` |
| `licenses.status` | `active`, `superseded`, `expired`, `invalid` |
| `license_checkins.channel` | `online`, `offline_file` |
| `license_checkins.status` | `success`, `failed` |

> Nilai di atas adalah usulan awal dan bisa disesuaikan. Seluruh kolom tersebut diberi `CHECK (kolom IN (...))`.

---

### 2.1 Identity & Organization

#### `users`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `email` | VARCHAR(255) | NOT NULL, UNIQUE |
| `name` | VARCHAR(255) | NOT NULL |
| `password_hash` | TEXT | — |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

#### `tenants`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `name` | VARCHAR(255) | NOT NULL |
| `slug` | VARCHAR(100) | NOT NULL, UNIQUE |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

#### `tenant_members`

| Kolom | Tipe | Constraint |
|---|---|---|
| `tenant_id` | UUID | NOT NULL, FK → `tenants(id)` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `joined_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **PK:** `(tenant_id, user_id)`

#### `companies`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL, FK → `tenants(id)` |
| `name` | VARCHAR(255) | NOT NULL |
| `code` | VARCHAR(100) | NOT NULL |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `deleted_at` | TIMESTAMPTZ | NULL (soft delete) |

- **UNIQUE (partial):** `(tenant_id, code) WHERE deleted_at IS NULL`

#### `branches`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `company_id` | UUID | NOT NULL, FK → `companies(id)` |
| `name` | VARCHAR(255) | NOT NULL |
| `code` | VARCHAR(100) | NOT NULL |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `deleted_at` | TIMESTAMPTZ | NULL (soft delete) |

- **UNIQUE (partial):** `(company_id, code) WHERE deleted_at IS NULL`

---

### 2.2 Solution Catalog

#### `solutions`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `code` | VARCHAR(100) | NOT NULL, UNIQUE |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | — |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `permission_catalog_version` | VARCHAR(50) | NULL — versi katalog permission terakhir yang di-sync |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

---

### 2.3 Plans & Add-ons

#### `plans`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `code` | VARCHAR(100) | NOT NULL |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | — |
| `price` | NUMERIC(18,2) | NOT NULL, default `0` |
| `currency` | VARCHAR(3) | NOT NULL, default `'IDR'` |
| `billing_cycle` | VARCHAR(30) | NOT NULL, default `'monthly'`, CHECK enum |
| `is_active` | BOOLEAN | NOT NULL, default `TRUE` |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `deleted_at` | TIMESTAMPTZ | NULL (soft delete) |

- **UNIQUE (partial):** `(solution_id, code) WHERE deleted_at IS NULL`
- **UNIQUE:** `(id, solution_id)` — target composite FK

#### `addons`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `code` | VARCHAR(100) | NOT NULL |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | — |
| `price` | NUMERIC(18,2) | NOT NULL, default `0` |
| `currency` | VARCHAR(3) | NOT NULL, default `'IDR'` |
| `billing_cycle` | VARCHAR(30) | NOT NULL, default `'monthly'`, CHECK enum |
| `is_active` | BOOLEAN | NOT NULL, default `TRUE` |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `deleted_at` | TIMESTAMPTZ | NULL (soft delete) |

- **UNIQUE (partial):** `(solution_id, code) WHERE deleted_at IS NULL`
- **UNIQUE:** `(id, solution_id)` — target composite FK

---

### 2.4 Benefits

#### `benefits`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `code` | VARCHAR(100) | NOT NULL |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | — |
| `value_type` | VARCHAR(30) | NOT NULL, CHECK enum |
| `aggregation_strategy` | VARCHAR(30) | NOT NULL, CHECK enum |
| `unit` | VARCHAR(50) | — |
| `metering` | VARCHAR(30) | NOT NULL, default `'none'`, CHECK enum |
| `usage_reset` | VARCHAR(30) | NOT NULL, default `'never'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **UNIQUE:** `(solution_id, code)`
- **UNIQUE:** `(id, solution_id)` — target composite FK

**Metering:**

| `metering` | Arti | Contoh |
|---|---|---|
| `none` | Benefit tidak diukur pemakaiannya (fitur on/off, nilai statis) | `export_pdf = true` |
| `counter` | Pemakaian bertambah lewat laporan delta | jumlah transaksi, API call |
| `gauge` | Pemakaian dilaporkan sebagai nilai absolut terkini | jumlah karyawan aktif, storage terpakai |

`usage_reset` menentukan kapan `counter` kembali ke 0. `gauge` tidak di-reset.

#### `plan_benefits`

| Kolom | Tipe | Constraint |
|---|---|---|
| `plan_id` | UUID | NOT NULL |
| `benefit_id` | UUID | NOT NULL |
| `solution_id` | UUID | NOT NULL |
| `value` | JSONB | NOT NULL |

- **PK:** `(plan_id, benefit_id)`
- **FK:** `(plan_id, solution_id)` → `plans(id, solution_id)`
- **FK:** `(benefit_id, solution_id)` → `benefits(id, solution_id)`

#### `addon_benefits`

| Kolom | Tipe | Constraint |
|---|---|---|
| `addon_id` | UUID | NOT NULL |
| `benefit_id` | UUID | NOT NULL |
| `solution_id` | UUID | NOT NULL |
| `value` | JSONB | NOT NULL |

- **PK:** `(addon_id, benefit_id)`
- **FK:** `(addon_id, solution_id)` → `addons(id, solution_id)`
- **FK:** `(benefit_id, solution_id)` → `benefits(id, solution_id)`

---

### 2.5 Subscriptions

#### `subscriptions`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL, FK → `tenants(id)` |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `plan_id` | UUID | NOT NULL |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `billing_cycle` | VARCHAR(30) | NOT NULL, CHECK enum |
| `current_period_start` | TIMESTAMPTZ | — |
| `current_period_end` | TIMESTAMPTZ | — |
| `price_at_purchase` | NUMERIC(18,2) | NOT NULL |
| `currency` | VARCHAR(3) | NOT NULL, default `'IDR'` |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **FK:** `(plan_id, solution_id)` → `plans(id, solution_id)`
- **UNIQUE:** `(id, solution_id)` — target composite FK

#### `subscription_addons`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `subscription_id` | UUID | NOT NULL |
| `addon_id` | UUID | NOT NULL |
| `solution_id` | UUID | NOT NULL |
| `quantity` | INTEGER | NOT NULL, default `1`, CHECK `quantity > 0` |
| `unit_price_at_purchase` | NUMERIC(18,2) | NOT NULL |
| `currency` | VARCHAR(3) | NOT NULL, default `'IDR'` |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **UNIQUE:** `(subscription_id, addon_id)`
- **FK:** `(subscription_id, solution_id)` → `subscriptions(id, solution_id)`
- **FK:** `(addon_id, solution_id)` → `addons(id, solution_id)`

#### `subscription_benefit_snapshots` (baru)

Menyimpan nilai benefit **pada saat pembelian**, sehingga perubahan `plan_benefits` / `addon_benefits` tidak memengaruhi subscription yang sudah berjalan.

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `subscription_id` | UUID | NOT NULL, FK → `subscriptions(id)` |
| `subscription_addon_id` | UUID | NULL, FK → `subscription_addons(id)` — NULL = berasal dari plan |
| `benefit_id` | UUID | NOT NULL, FK → `benefits(id)` |
| `value` | JSONB | NOT NULL |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **UNIQUE NULLS NOT DISTINCT:** `(subscription_id, subscription_addon_id, benefit_id)`

**Aturan pembentukan snapshot:**

| Event | Aksi |
|---|---|
| Create subscription | Salin seluruh `plan_benefits` milik plan ke snapshot (`subscription_addon_id = NULL`) |
| Change plan | Hapus snapshot plan lama, salin `plan_benefits` plan baru |
| Tambah add-on | Salin `addon_benefits` milik add-on ke snapshot (`subscription_addon_id` terisi) |
| Hapus / cancel add-on | Hapus snapshot milik `subscription_addon_id` tersebut |

Effective entitlement dihitung dari snapshot ini menggunakan `benefits.aggregation_strategy`, dengan nilai add-on dikalikan `subscription_addons.quantity` bila relevan (mis. strategi `sum`).

---

### 2.6 Authorization

#### `permissions`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `code` | VARCHAR(150) | NOT NULL |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | — |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **UNIQUE:** `(solution_id, code)`
- Permission `deprecated` tetap tersimpan (tidak dihapus) agar role lama tidak rusak, tetapi tidak bisa ditambahkan ke role baru.
- **UNIQUE:** `(id, solution_id)` — target composite FK

#### `roles` (diperbarui — mendukung custom role per tenant)

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `tenant_id` | UUID | NULL, FK → `tenants(id)` |
| `code` | VARCHAR(100) | NOT NULL |
| `name` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | — |
| `is_system` | BOOLEAN | NOT NULL, default `FALSE` |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `deleted_at` | TIMESTAMPTZ | NULL (soft delete) |

- **CHECK:** `NOT is_system OR tenant_id IS NULL` — role sistem tidak boleh milik tenant
- **UNIQUE (partial):** `(solution_id, code) WHERE tenant_id IS NULL AND deleted_at IS NULL` — role global
- **UNIQUE (partial):** `(solution_id, tenant_id, code) WHERE tenant_id IS NOT NULL AND deleted_at IS NULL` — custom role tenant
- **UNIQUE:** `(id, solution_id)` — target composite FK

**Jenis role:**

| Jenis | `tenant_id` | `is_system` | Dibuat oleh | Bisa diubah/dihapus tenant |
|---|---|---|---|---|
| System role | NULL | TRUE | Platform | Tidak |
| Global template role | NULL | FALSE | Platform | Tidak |
| Custom tenant role | terisi | FALSE | Tenant | Ya (hanya tenant pemilik) |

**Aturan custom role:**
- Tenant hanya boleh membuat role untuk solution yang sedang di-subscribe (status `trialing` / `active`).
- Permission yang dipasang harus milik solution yang sama (dijaga composite FK di `role_permissions`).
- Custom role hanya terlihat dan bisa di-assign di dalam tenant pemiliknya.

#### `role_permissions`

| Kolom | Tipe | Constraint |
|---|---|---|
| `role_id` | UUID | NOT NULL |
| `permission_id` | UUID | NOT NULL |
| `solution_id` | UUID | NOT NULL |

- **PK:** `(role_id, permission_id)`
- **FK:** `(role_id, solution_id)` → `roles(id, solution_id)`
- **FK:** `(permission_id, solution_id)` → `permissions(id, solution_id)`

#### `user_role_assignments`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `user_id` | UUID | NOT NULL |
| `role_id` | UUID | NOT NULL, FK → `roles(id)` |
| `scope_type` | VARCHAR(30) | NOT NULL, CHECK enum |
| `scope_id` | UUID | NOT NULL |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **FK:** `(tenant_id, user_id)` → `tenant_members(tenant_id, user_id)` — user wajib anggota tenant
- **UNIQUE:** `(user_id, role_id, scope_type, scope_id)`

**Validasi di service layer:**
- Role harus berasal dari solution yang sedang di-subscribe tenant.
- Jika `roles.tenant_id` terisi, nilainya harus sama dengan `user_role_assignments.tenant_id`.
- `scope_id` harus valid sesuai `scope_type` dan milik tenant tersebut (`tenant` = tenant itu sendiri, `company` = company milik tenant, `branch` = branch dari company milik tenant).

---

### 2.7 Auth & Session (baru)

#### `auth_sessions`

Satu baris per sesi login user pada satu solution. Menyimpan refresh token (hash) dan tenant aktif.

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK — dipakai sebagai claim `sid` |
| `user_id` | UUID | NOT NULL, FK → `users(id)` |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` — audience sesi |
| `tenant_id` | UUID | NULL, FK → `tenants(id)` — tenant aktif |
| `refresh_token_hash` | TEXT | NOT NULL, UNIQUE |
| `user_agent` | TEXT | — |
| `ip_address` | INET | — |
| `expires_at` | TIMESTAMPTZ | NOT NULL |
| `last_used_at` | TIMESTAMPTZ | — |
| `revoked_at` | TIMESTAMPTZ | NULL |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

**Aturan:**
- Refresh token bersifat opaque, di-rotate setiap kali dipakai (hash lama diganti).
- Refresh ditolak jika sesi revoked/expired, user tidak `active`, membership tenant aktif tidak `active`, atau tenant tidak `active`.
- Logout = set `revoked_at`. Access token yang sudah terbit tetap valid sampai `exp`, karena itu umur access token dibuat pendek.

**Alur pilih / switch tenant:**

1. Login → sesi dibuat. Jika user hanya punya 1 tenant yang subscribe solution tersebut, `tenant_id` langsung terisi. Jika lebih dari 1, `tenant_id = NULL`.
2. Token dengan `tid` kosong hanya boleh memanggil endpoint identitas & pemilihan tenant.
3. `POST /auth/switch-tenant` → validasi membership aktif + subscription aktif tenant untuk solution sesi → update `auth_sessions.tenant_id` → terbitkan access & refresh token baru.

#### `signing_keys`

Kunci penandatangan JWT, dipublikasikan lewat JWKS.

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `kid` | VARCHAR(100) | NOT NULL, UNIQUE — dipakai di header JWT |
| `algorithm` | VARCHAR(20) | NOT NULL, default `'RS256'` |
| `public_key` | TEXT | NOT NULL (PEM / JWK) |
| `private_key_ref` | TEXT | NOT NULL — referensi ke KMS / secret manager, bukan private key mentah |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `retired_at` | TIMESTAMPTZ | NULL |

**Rotasi:** hanya satu key `active` (dipakai menandatangani). Key `retiring` masih dipublikasikan di JWKS sampai semua token yang ditandatanganinya kedaluwarsa, lalu menjadi `retired` dan dikeluarkan dari JWKS.

- **UNIQUE (partial):** `(status) WHERE status = 'active'`

---

### 2.8 Solution Clients & Token Claims (baru)

#### `solution_clients`

Kredensial service-to-service untuk backend solution (OAuth2 client credentials).

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `name` | VARCHAR(255) | NOT NULL — mis. "Easy HR Backend Prod" |
| `client_id` | VARCHAR(100) | NOT NULL, UNIQUE |
| `client_secret_hash` | TEXT | NOT NULL |
| `scopes` | TEXT[] | NOT NULL, default `'{}'` |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `last_used_at` | TIMESTAMPTZ | — |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `revoked_at` | TIMESTAMPTZ | NULL |

- Secret hanya ditampilkan sekali saat create / rotate.
- Client hanya boleh mengakses data milik `solution_id`-nya dan tenant yang subscribe solution tersebut.

**Daftar scope service:**

| Scope | Akses |
|---|---|
| `users:read` | Baca user & membership tenant |
| `users:write` | Create / invite / update / nonaktifkan user & membership |
| `org:read` | Baca company & branch |
| `org:write` | Kelola company & branch |
| `authz:check` | `authz/check`, `check-batch`, effective permissions |
| `roles:write` | Kelola role & role assignment |
| `entitlements:read` | Baca entitlement & usage |
| `usage:write` | Laporkan usage |
| `catalog:sync` | Sync katalog permission & system role |
| `events:read` | Polling / replay event |
| `license:read` | Status & file license (khusus `on_prem`) |

#### Definisi Claim Token

**User access token** (JWT, umur pendek, mis. 15 menit):

| Claim | Isi |
|---|---|
| `iss` | URL platform |
| `sub` | `users.id` |
| `aud` | `solutions.code` |
| `tid` | `tenants.id` aktif (boleh kosong sebelum pilih tenant) |
| `sid` | `auth_sessions.id` |
| `typ` | `user` |
| `iat`, `exp`, `jti` | Standar JWT |

**Service access token** (JWT, hasil client credentials):

| Claim | Isi |
|---|---|
| `iss` | URL platform |
| `sub` | `solution_clients.client_id` |
| `aud` | `platform` |
| `sol` | `solutions.id` |
| `scope` | Daftar scope, dipisah spasi |
| `typ` | `service` |
| `iat`, `exp`, `jti` | Standar JWT |

Header JWT selalu memuat `kid` sesuai `signing_keys`. Permission dan entitlement **tidak** dimasukkan ke token.

---

### 2.9 Events & Webhooks (baru)

#### `events` (outbox)

Ditulis dalam transaksi yang sama dengan perubahan data, lalu dikirim oleh worker.

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK — dipakai sebagai idempotency key penerima |
| `event_type` | VARCHAR(100) | NOT NULL |
| `tenant_id` | UUID | NULL — NULL untuk event level user/global |
| `solution_id` | UUID | NULL — NULL jika relevan untuk semua solution |
| `payload` | JSONB | NOT NULL |
| `occurred_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

#### `webhook_endpoints`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `solution_id` | UUID | NOT NULL, FK → `solutions(id)` |
| `url` | TEXT | NOT NULL |
| `secret_encrypted` | TEXT | NOT NULL — untuk tanda tangan HMAC |
| `event_types` | TEXT[] | NOT NULL — langganan event |
| `status` | VARCHAR(30) | NOT NULL, default `'active'`, CHECK enum |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

#### `webhook_deliveries`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `event_id` | UUID | NOT NULL, FK → `events(id)` |
| `webhook_endpoint_id` | UUID | NOT NULL, FK → `webhook_endpoints(id)` |
| `status` | VARCHAR(30) | NOT NULL, default `'pending'`, CHECK enum |
| `attempt_count` | INTEGER | NOT NULL, default `0` |
| `next_attempt_at` | TIMESTAMPTZ | — |
| `last_response_code` | INTEGER | — |
| `last_error` | TEXT | — |
| `delivered_at` | TIMESTAMPTZ | — |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **UNIQUE:** `(event_id, webhook_endpoint_id)`

**Aturan pengiriman:**
- At-least-once; penerima wajib idempotent berdasarkan `event.id`.
- Header: `X-Platform-Event-Id`, `X-Platform-Event-Type`, `X-Platform-Timestamp`, `X-Platform-Signature` (HMAC-SHA256 dari `timestamp.body`).
- Retry dengan exponential backoff; setelah batas percobaan → `dead`, bisa di-retry manual.
- Event tenant-scoped hanya dikirim ke solution yang di-subscribe tenant tersebut. Event user dikirim ke solution dari tenant mana pun tempat user menjadi anggota.
- Urutan tidak dijamin; payload memuat `updated_at` / versi agar penerima bisa mengabaikan event yang lebih lama.
- Polling `GET /events` tersedia sebagai fallback & replay.

**Katalog event:**

| Grup | Event Type |
|---|---|
| User | `user.updated`, `user.status_changed` |
| Tenant | `tenant.updated`, `tenant.status_changed` |
| Membership | `tenant_member.added`, `tenant_member.updated`, `tenant_member.removed` |
| Organization | `company.created`, `company.updated`, `company.deleted`, `branch.created`, `branch.updated`, `branch.deleted` |
| Subscription | `subscription.created`, `subscription.plan_changed`, `subscription.status_changed`, `subscription_addon.added`, `subscription_addon.updated`, `subscription_addon.removed` |
| Entitlement | `entitlement.changed` (dipicu oleh perubahan snapshot) |
| Authorization | `role.created`, `role.updated`, `role.deleted`, `role.permissions_changed`, `role_assignment.created`, `role_assignment.deleted` |
| Usage | `usage.limit_reached` |
| License (`on_prem`) | `license.installed`, `license.state_changed`, `license.expiring` |

---

### 2.10 Usage & Metering (baru)

#### `usage_records`

Catatan laporan usage dari solution (append-only, idempotent).

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL, FK → `tenants(id)` |
| `solution_id` | UUID | NOT NULL |
| `benefit_id` | UUID | NOT NULL |
| `mode` | VARCHAR(30) | NOT NULL, CHECK IN (`'delta'`, `'absolute'`) |
| `quantity` | NUMERIC(18,4) | NOT NULL |
| `idempotency_key` | VARCHAR(150) | NOT NULL |
| `reported_by_client_id` | UUID | NOT NULL, FK → `solution_clients(id)` |
| `occurred_at` | TIMESTAMPTZ | NOT NULL |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **FK:** `(benefit_id, solution_id)` → `benefits(id, solution_id)`
- **UNIQUE:** `(solution_id, idempotency_key)`

#### `tenant_usage`

Nilai pemakaian terkini per tenant per benefit per periode.

| Kolom | Tipe | Constraint |
|---|---|---|
| `tenant_id` | UUID | NOT NULL, FK → `tenants(id)` |
| `solution_id` | UUID | NOT NULL |
| `benefit_id` | UUID | NOT NULL |
| `period_start` | TIMESTAMPTZ | NOT NULL — untuk `usage_reset = never` diisi tanggal awal tetap |
| `period_end` | TIMESTAMPTZ | NULL |
| `used` | NUMERIC(18,4) | NOT NULL, default `0` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **PK:** `(tenant_id, benefit_id, period_start)`
- **FK:** `(benefit_id, solution_id)` → `benefits(id, solution_id)`

**Aturan:**
- `counter` → laporan `delta`, `used = used + quantity`.
- `gauge` → laporan `absolute`, `used = quantity`.
- Benefit dengan `metering = none` menolak laporan usage.
- Entitlement response memuat `limit`, `used`, dan `remaining` untuk benefit yang di-meter.
- Platform tidak memblokir aksi di solution; solution yang menegakkan limit berdasarkan `remaining`. Event `usage.limit_reached` dikirim saat `used >= limit`.

---

### 2.11 Indexes

| Index | Tabel | Kolom | Keterangan |
|---|---|---|---|
| `idx_tenant_members_user_id` | `tenant_members` | `user_id` | baru |
| `idx_companies_tenant_id` | `companies` | `tenant_id` | |
| `idx_branches_company_id` | `branches` | `company_id` | |
| `idx_plans_solution_id` | `plans` | `solution_id` | |
| `idx_addons_solution_id` | `addons` | `solution_id` | |
| `idx_benefits_solution_id` | `benefits` | `solution_id` | |
| `idx_plan_benefits_benefit_id` | `plan_benefits` | `benefit_id` | baru |
| `idx_addon_benefits_benefit_id` | `addon_benefits` | `benefit_id` | baru |
| `idx_subscriptions_tenant_id` | `subscriptions` | `tenant_id` | |
| `idx_subscriptions_solution_id` | `subscriptions` | `solution_id` | |
| `idx_subscription_addons_subscription_id` | `subscription_addons` | `subscription_id` | |
| `idx_sub_benefit_snapshots_subscription_id` | `subscription_benefit_snapshots` | `subscription_id` | baru |
| `idx_roles_solution_id` | `roles` | `solution_id` | |
| `idx_roles_tenant_solution` | `roles` | `tenant_id, solution_id` | baru |
| `idx_role_permissions_permission_id` | `role_permissions` | `permission_id` | baru |
| `idx_user_role_assignments_user_tenant` | `user_role_assignments` | `user_id, tenant_id` | |
| `idx_user_role_assignments_scope` | `user_role_assignments` | `scope_type, scope_id` | |
| `idx_auth_sessions_user_id` | `auth_sessions` | `user_id` | baru |
| `idx_auth_sessions_active` | `auth_sessions` | `user_id, solution_id` WHERE `revoked_at IS NULL` | baru |
| `idx_solution_clients_solution_id` | `solution_clients` | `solution_id` | baru |
| `idx_webhook_endpoints_solution_id` | `webhook_endpoints` | `solution_id` | baru |
| `idx_events_occurred_at` | `events` | `occurred_at` | baru |
| `idx_events_tenant_id` | `events` | `tenant_id, occurred_at` | baru |
| `idx_webhook_deliveries_pending` | `webhook_deliveries` | `next_attempt_at` WHERE `status IN ('pending','failed')` | baru |
| `idx_usage_records_tenant_benefit` | `usage_records` | `tenant_id, benefit_id, occurred_at` | baru |

### 2.12 Soft Delete

| Tabel | Mekanisme |
|---|---|
| `companies`, `branches`, `plans`, `addons`, `roles` | Set `deleted_at = NOW()`; query default memfilter `deleted_at IS NULL` |
| `tenant_members` | Set `status = 'inactive'` dan hapus role assignment user di tenant tersebut |
| `subscription_addons` | Set `status = 'canceled'` dan hapus snapshot benefit add-on tersebut |

Data yang di-soft delete tetap utuh untuk keperluan histori dan relasi FK.

### 2.13 Catatan Integritas

Konsistensi solution antara plan, add-on, benefit, subscription, role, dan permission kini dijaga oleh composite FK di level DB. Validasi yang tetap di service layer: role sesuai subscription tenant, kepemilikan custom role, dan validitas `scope_id` terhadap tenant, company, dan branch.


### 2.14 Deployment Mode & Licensing (baru, khusus `on_prem`)

#### Prinsip

| Aspek | `cloud` | `on_prem` |
|---|---|---|
| Sumber entitlement | `subscription_benefit_snapshots` | **License file bertanda tangan** (diverifikasi saat load & berkala) |
| Kelola plan, add-on, benefit, subscription | Lewat API | **Dinonaktifkan** — hanya berubah lewat license baru |
| Public key verifikasi license | — | **Ditanam di binary**, bukan di DB / config yang bisa diedit |
| Limit usage | `tenant_usage` | Gauge dihitung dari data nyata (mis. `COUNT` user aktif); counter dari `usage_records` lokal |
| Laporan usage ke vendor | — | Lewat check-in online atau file offline |

Tabel `plans`, `plan_benefits`, `subscriptions`, dan snapshot di mode `on_prem` hanya cerminan informasi. Mengubahnya langsung di DB **tidak** memengaruhi entitlement.

`deployment_mode` dibaca dari konfigurasi build/runtime dan dicatat di `installations`. Mode `on_prem` tidak bisa diubah ke `cloud` lewat API.

#### `installations`

Satu baris per instalasi.

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK — installation ID, dikirim ke vendor untuk menerbitkan license |
| `deployment_mode` | VARCHAR(30) | NOT NULL, CHECK enum |
| `fingerprint` | TEXT | NOT NULL — hash identitas server (mis. machine ID, hostname, MAC) |
| `license_state` | VARCHAR(30) | NOT NULL, default `'invalid'`, CHECK enum |
| `last_seen_at` | TIMESTAMPTZ | NOT NULL — waktu tertinggi yang pernah tercatat, untuk deteksi mundur jam |
| `last_seen_hmac` | TEXT | NOT NULL — HMAC atas `last_seen_at` agar tidak bisa diedit diam-diam |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| `updated_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

#### `licenses`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `license_id` | VARCHAR(100) | NOT NULL, UNIQUE — ID dari vendor |
| `installation_id` | UUID | NOT NULL, FK → `installations(id)` |
| `customer_name` | VARCHAR(255) | NOT NULL |
| `payload` | JSONB | NOT NULL — isi license (lihat struktur di bawah) |
| `signature` | TEXT | NOT NULL — tanda tangan vendor atas `payload` |
| `key_id` | VARCHAR(100) | NOT NULL — kid public key vendor yang dipakai |
| `issued_at` | TIMESTAMPTZ | NOT NULL |
| `valid_from` | TIMESTAMPTZ | NOT NULL |
| `valid_until` | TIMESTAMPTZ | NOT NULL |
| `grace_until` | TIMESTAMPTZ | NOT NULL |
| `status` | VARCHAR(30) | NOT NULL, CHECK enum |
| `installed_by` | UUID | NULL, FK → `users(id)` |
| `installed_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

- **UNIQUE (partial):** `(installation_id) WHERE status = 'active'`
- Kolom selain `payload` dan `signature` hanya salinan untuk query; nilai yang dipakai selalu dari `payload` yang lolos verifikasi.

**Struktur `payload`:**

```json
{
  "license_id": "LIC-2026-0001",
  "installation_id": "uuid",
  "fingerprint": "sha256:...",
  "customer": { "name": "PT Contoh", "code": "contoh" },
  "issued_at": "2026-10-01T00:00:00Z",
  "valid_from": "2026-10-01T00:00:00Z",
  "valid_until": "2026-10-31T23:59:59Z",
  "grace_days": 14,
  "limits": { "max_tenants": 1 },
  "tenants": [
    {
      "slug": "contoh",
      "solutions": [
        {
          "code": "easy-hr",
          "plan": "pro",
          "addons": [{ "code": "payroll", "quantity": 1 }],
          "benefits": {
            "max_employees": 200,
            "payroll_module": true
          }
        }
      ]
    }
  ]
}
```

#### `license_checkins`

| Kolom | Tipe | Constraint |
|---|---|---|
| `id` | UUID | PK |
| `license_id` | UUID | NOT NULL, FK → `licenses(id)` |
| `channel` | VARCHAR(30) | NOT NULL, CHECK enum |
| `usage_payload` | JSONB | NOT NULL — ringkasan usage yang dilaporkan |
| `status` | VARCHAR(30) | NOT NULL, CHECK enum |
| `response_code` | INTEGER | — |
| `error` | TEXT | — |
| `server_time` | TIMESTAMPTZ | — — waktu dari vendor (pembanding jam lokal) |
| `created_at` | TIMESTAMPTZ | NOT NULL, default `NOW()` |

#### Siklus License

1. **Instalasi:** platform membuat `installations` dan menampilkan installation ID + fingerprint.
2. **Aktivasi:** online — platform mengirim ke license server vendor dan menerima license. Offline — admin mengirim installation ID ke vendor, lalu upload file license.
3. **Check-in berkala** (mis. harian): kirim usage, terima license perpanjangan (masa berlaku pendek, mis. 30 hari) + `server_time`.
4. **Offline / air-gapped:** ekspor laporan usage sebagai file, kirim ke vendor, upload license perpanjangan.

#### State & Penegakan

| `license_state` | Kondisi | Perilaku |
|---|---|---|
| `valid` | Signature valid, fingerprint cocok, `now ≤ valid_until` | Normal |
| `grace` | `valid_until < now ≤ grace_until` | Normal + peringatan di UI dan response |
| `read_only` | `now > grace_until`, atau limit `max_tenants` terlampaui | Baca & ekspor boleh; semua operasi tulis ke data bisnis ditolak |
| `invalid` | Signature tidak valid, fingerprint tidak cocok, license untuk instalasi lain, atau terdeteksi jam mundur | Sama dengan `read_only` + peringatan keras; login admin tetap bisa untuk upload license |

**Pemeriksaan:**
- Verifikasi signature saat startup, saat upload license, dan berkala (mis. tiap jam). Hasil disimpan di memori, bukan hanya di DB.
- **Deteksi jam mundur:** jika `now < installations.last_seen_at` melebihi toleransi (mis. 1 jam), atau HMAC `last_seen_at` tidak cocok → `invalid`.
- **Limit benefit:** benefit `gauge` dihitung dari data nyata saat dicek; benefit `counter` dari `usage_records` lokal.
- **Limit terlampaui** (mis. jumlah karyawan > `max_employees`): penambahan diblokir, data lama tetap bisa diakses dan diubah.
- **Cek berlapis:** solution SDK ikut memverifikasi license (via `GET /license/raw` + public key yang ditanam di SDK), tidak hanya mengandalkan respons platform.

**Endpoint yang dinonaktifkan di `on_prem`** (respons `403`, kode `LICENSE_MANAGED`):

| Grup | Endpoint |
|---|---|
| Solution Catalog | `POST /solutions`, `PATCH /solutions/{solutionId}` |
| Plans | `POST /solutions/{solutionId}/plans`, `PATCH`/`DELETE /plans/{planId}`, semua `PUT`/`DELETE /plans/{planId}/benefits...` |
| Add-ons | `POST /solutions/{solutionId}/addons`, `PATCH`/`DELETE /addons/{addonId}`, semua `PUT`/`DELETE /addons/{addonId}/benefits...` |
| Benefits | `POST /solutions/{solutionId}/benefits`, `PATCH /benefits/{benefitId}` |
| Subscription | `POST /tenants/{tenantId}/subscriptions`, `PATCH /subscriptions/{subscriptionId}`, `change-plan`, `cancel`, `reactivate` |
| Subscription Add-ons | `POST`, `PATCH`, `DELETE /subscriptions/{subscriptionId}/addons...` |
| Tenant | `POST /tenants` — hanya diizinkan untuk tenant yang tercantum di license dan dalam batas `max_tenants` |

**Catatan keamanan:** mode `on_prem` dikirim sebagai binary / container image (bukan source code); untuk bahasa interpreted gunakan obfuscation. Penegakan teknis dilengkapi kontrak lisensi (hak audit, kewajiban laporan usage, sanksi).

---

---

## 3. API Specification

Base URL: `/api/v1`

### 3.1 Authentication & Identity

| Method | Path | Catatan |
|---|---|---|
| POST | `/auth/login` | Wajib menyertakan solution (audience); membuat `auth_sessions` |
| POST | `/auth/refresh` | Rotasi refresh token |
| POST | `/auth/logout` | Revoke sesi saat ini |
| GET | `/auth/me` | User + tenant aktif |
| GET | `/auth/tenants` | Daftar tenant user yang subscribe solution sesi (baru) |
| POST | `/auth/switch-tenant` | Ganti tenant aktif, terbitkan token baru (baru) |
| GET | `/auth/sessions` | Daftar sesi aktif user (baru) |
| DELETE | `/auth/sessions/{sessionId}` | Revoke sesi tertentu (baru) |
| POST | `/auth/logout-all` | Revoke semua sesi user (baru) |

| Method | Path |
|---|---|
| GET | `/users/me` |
| PATCH | `/users/me` |
| GET | `/users` |
| POST | `/users` |
| GET | `/users/{userId}` |
| PATCH | `/users/{userId}` |
| PATCH | `/users/{userId}/status` |

### 3.2 Tenant & Membership

| Method | Path | Catatan |
|---|---|---|
| GET | `/tenants` | |
| POST | `/tenants` | |
| GET | `/tenants/{tenantId}` | |
| PATCH | `/tenants/{tenantId}` | |
| GET | `/tenants/{tenantId}/members` | |
| POST | `/tenants/{tenantId}/members` | |
| PATCH | `/tenants/{tenantId}/members/{userId}` | |
| DELETE | `/tenants/{tenantId}/members/{userId}` | Soft: status `inactive` + hapus role assignment |

### 3.3 Company & Branch

| Method | Path | Catatan |
|---|---|---|
| GET | `/tenants/{tenantId}/companies` | |
| POST | `/tenants/{tenantId}/companies` | |
| GET | `/companies/{companyId}` | |
| PATCH | `/companies/{companyId}` | |
| DELETE | `/companies/{companyId}` | Soft delete |
| GET | `/companies/{companyId}/branches` | |
| POST | `/companies/{companyId}/branches` | |
| GET | `/branches/{branchId}` | |
| PATCH | `/branches/{branchId}` | |
| DELETE | `/branches/{branchId}` | Soft delete |

### 3.4 Solution Catalog

| Method | Path |
|---|---|
| GET | `/solutions` |
| POST | `/solutions` |
| GET | `/solutions/{solutionId}` |
| PATCH | `/solutions/{solutionId}` |
| GET | `/solutions/{solutionId}/plans` |
| GET | `/solutions/{solutionId}/addons` |
| GET | `/solutions/{solutionId}/benefits` |
| GET | `/solutions/{solutionId}/permissions` |
| GET | `/solutions/{solutionId}/roles` |

### 3.5 Plans & Plan Benefits

| Method | Path | Catatan |
|---|---|---|
| POST | `/solutions/{solutionId}/plans` | |
| GET | `/plans/{planId}` | |
| PATCH | `/plans/{planId}` | |
| DELETE | `/plans/{planId}` | Soft delete |
| GET | `/plans/{planId}/benefits` | |
| PUT | `/plans/{planId}/benefits` | Tidak memengaruhi subscription berjalan (snapshot) |
| PUT | `/plans/{planId}/benefits/{benefitId}` | Tidak memengaruhi subscription berjalan (snapshot) |
| DELETE | `/plans/{planId}/benefits/{benefitId}` | |

### 3.6 Add-ons & Add-on Benefits

| Method | Path | Catatan |
|---|---|---|
| POST | `/solutions/{solutionId}/addons` | |
| GET | `/addons/{addonId}` | |
| PATCH | `/addons/{addonId}` | |
| DELETE | `/addons/{addonId}` | Soft delete |
| GET | `/addons/{addonId}/benefits` | |
| PUT | `/addons/{addonId}/benefits` | Tidak memengaruhi subscription berjalan (snapshot) |
| PUT | `/addons/{addonId}/benefits/{benefitId}` | Tidak memengaruhi subscription berjalan (snapshot) |
| DELETE | `/addons/{addonId}/benefits/{benefitId}` | |

### 3.7 Master Benefits

| Method | Path |
|---|---|
| POST | `/solutions/{solutionId}/benefits` |
| GET | `/benefits/{benefitId}` |
| PATCH | `/benefits/{benefitId}` |
| GET | `/benefits/{benefitId}/plans` |
| GET | `/benefits/{benefitId}/addons` |

### 3.8 Subscription

| Method | Path | Catatan |
|---|---|---|
| GET | `/tenants/{tenantId}/subscriptions` | |
| POST | `/tenants/{tenantId}/subscriptions` | Membuat snapshot benefit plan |
| GET | `/subscriptions/{subscriptionId}` | |
| PATCH | `/subscriptions/{subscriptionId}` | |
| POST | `/subscriptions/{subscriptionId}/change-plan` | Mengganti snapshot benefit plan |
| POST | `/subscriptions/{subscriptionId}/cancel` | |
| POST | `/subscriptions/{subscriptionId}/reactivate` | |

### 3.9 Subscription Add-ons

| Method | Path | Catatan |
|---|---|---|
| GET | `/subscriptions/{subscriptionId}/addons` | |
| POST | `/subscriptions/{subscriptionId}/addons` | Membuat snapshot benefit add-on |
| PATCH | `/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}` | |
| DELETE | `/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}` | Soft: status `canceled` + hapus snapshot add-on |

### 3.10 Effective Benefits / Entitlements

Dihitung dari `subscription_benefit_snapshots`. Untuk benefit yang di-meter, response memuat `limit`, `used`, dan `remaining` dari `tenant_usage`. Di mode `on_prem`, entitlement berasal dari license dan response juga memuat `license_state`.

| Method | Path |
|---|---|
| GET | `/tenants/{tenantId}/solutions/{solutionId}/entitlements` |
| GET | `/tenants/{tenantId}/solutions/{solutionId}/entitlements/{benefitCode}` |
| POST | `/tenants/{tenantId}/solutions/{solutionId}/entitlements/check` |

### 3.11 Roles — Platform (Global)

| Method | Path | Catatan |
|---|---|---|
| GET | `/solutions/{solutionId}/roles` | Hanya role global (`tenant_id IS NULL`) |
| POST | `/solutions/{solutionId}/roles` | Membuat role global |
| GET | `/roles/{roleId}` | |
| PATCH | `/roles/{roleId}` | Role `is_system` tidak bisa diubah |
| DELETE | `/roles/{roleId}` | Soft delete; role `is_system` tidak bisa dihapus |
| GET | `/roles/{roleId}/permissions` | |
| PUT | `/roles/{roleId}/permissions` | Role `is_system` tidak bisa diubah |

### 3.12 Roles — Tenant Custom (baru)

| Method | Path | Catatan |
|---|---|---|
| GET | `/tenants/{tenantId}/solutions/{solutionId}/roles` | Role global + custom role milik tenant |
| POST | `/tenants/{tenantId}/solutions/{solutionId}/roles` | Membuat custom role; tenant wajib subscribe solution |
| GET | `/tenants/{tenantId}/roles/{roleId}` | |
| PATCH | `/tenants/{tenantId}/roles/{roleId}` | Hanya custom role milik tenant |
| DELETE | `/tenants/{tenantId}/roles/{roleId}` | Soft delete; hanya custom role milik tenant |
| GET | `/tenants/{tenantId}/roles/{roleId}/permissions` | |
| PUT | `/tenants/{tenantId}/roles/{roleId}/permissions` | Permission harus milik solution yang sama |

### 3.13 Permissions

| Method | Path |
|---|---|
| GET | `/solutions/{solutionId}/permissions` |
| POST | `/solutions/{solutionId}/permissions` |
| GET | `/permissions/{permissionId}` |
| PATCH | `/permissions/{permissionId}` |

### 3.14 User Role Assignments

| Method | Path | Catatan |
|---|---|---|
| GET | `/tenants/{tenantId}/users/{userId}/role-assignments` | |
| POST | `/tenants/{tenantId}/users/{userId}/role-assignments` | User wajib anggota tenant; bisa role global atau custom milik tenant |
| PATCH | `/role-assignments/{assignmentId}` | |
| DELETE | `/role-assignments/{assignmentId}` | |

### 3.15 Authorization Check

| Method | Path |
|---|---|
| POST | `/authz/check` |
| POST | `/authz/check-batch` |
| GET | `/tenants/{tenantId}/users/{userId}/effective-permissions` |

### 3.16 Service Auth & Keys (baru)

| Method | Path | Catatan |
|---|---|---|
| POST | `/oauth/token` | `grant_type=client_credentials`; menghasilkan service token |
| GET | `/.well-known/jwks.json` | Public key aktif & retiring (publik, boleh di-cache) |
| GET | `/.well-known/openid-configuration` | Metadata issuer, JWKS URI, token endpoint |
| GET | `/admin/signing-keys` | Daftar signing key (admin platform) |
| POST | `/admin/signing-keys/rotate` | Buat key baru, key lama → `retiring` |

### 3.17 Solution Clients (baru)

| Method | Path | Catatan |
|---|---|---|
| GET | `/solutions/{solutionId}/clients` | |
| POST | `/solutions/{solutionId}/clients` | Secret ditampilkan sekali |
| GET | `/solution-clients/{clientId}` | |
| PATCH | `/solution-clients/{clientId}` | Ubah nama / scope |
| POST | `/solution-clients/{clientId}/rotate-secret` | Secret baru ditampilkan sekali |
| POST | `/solution-clients/{clientId}/revoke` | |

### 3.18 Webhooks & Events (baru)

| Method | Path | Catatan |
|---|---|---|
| GET | `/solutions/{solutionId}/webhooks` | |
| POST | `/solutions/{solutionId}/webhooks` | Secret ditampilkan sekali |
| GET | `/webhooks/{webhookId}` | |
| PATCH | `/webhooks/{webhookId}` | Ubah URL / event types / status |
| DELETE | `/webhooks/{webhookId}` | |
| POST | `/webhooks/{webhookId}/rotate-secret` | |
| POST | `/webhooks/{webhookId}/test` | Kirim event uji |
| GET | `/webhooks/{webhookId}/deliveries` | Riwayat pengiriman |
| POST | `/webhook-deliveries/{deliveryId}/retry` | Retry manual |
| GET | `/events` | Polling / replay; filter `since`, `event_type`, `tenant_id`; dibatasi ke solution pemanggil |
| GET | `/event-types` | Katalog event |

### 3.19 Usage Reporting (baru)

| Method | Path | Catatan |
|---|---|---|
| POST | `/tenants/{tenantId}/solutions/{solutionId}/usage` | Laporan `delta` (counter); wajib `idempotency_key` |
| PUT | `/tenants/{tenantId}/solutions/{solutionId}/usage/{benefitCode}` | Laporan `absolute` (gauge); wajib `idempotency_key` |
| GET | `/tenants/{tenantId}/solutions/{solutionId}/usage` | Pemakaian periode berjalan |
| GET | `/tenants/{tenantId}/solutions/{solutionId}/usage/{benefitCode}` | |

### 3.20 Catalog Sync (baru)

| Method | Path | Catatan |
|---|---|---|
| PUT | `/solutions/{solutionId}/permissions/sync` | Kirim katalog lengkap + `version`; upsert, yang tidak ada di daftar → `deprecated` |
| PUT | `/solutions/{solutionId}/roles/sync` | Sync system role default beserta permission-nya (`is_system = TRUE`) |
| GET | `/solutions/{solutionId}/catalog-version` | Versi katalog yang tersimpan |

Sync bersifat idempotent: jika `version` sama dengan `solutions.permission_catalog_version`, tidak ada perubahan. Dipanggil saat deploy solution.

### 3.21 License & Installation (baru, khusus `on_prem`)

| Method | Path | Catatan |
|---|---|---|
| GET | `/installation` | Installation ID, fingerprint, deployment mode |
| GET | `/license` | License aktif, `license_state`, sisa hari, limit vs pemakaian |
| POST | `/license` | Upload file license (admin); diverifikasi sebelum disimpan |
| GET | `/license/raw` | Payload + signature license aktif, untuk verifikasi oleh solution SDK |
| GET | `/license/history` | Riwayat license |
| POST | `/license/activate` | Aktivasi online ke license server vendor |
| POST | `/license/checkin` | Picu check-in online manual |
| GET | `/license/usage-report` | Ekspor laporan usage (mode offline) |
| GET | `/license/checkins` | Riwayat check-in |

Di mode `cloud`, seluruh endpoint di atas mengembalikan `404`.

---

## 4. Endpoint Minimum untuk MVP

| Grup | Method | Path |
|---|---|---|
| Authentication | POST | `/api/v1/auth/login` |
| Authentication | POST | `/api/v1/auth/refresh` |
| Authentication | POST | `/api/v1/auth/logout` |
| Authentication | GET | `/api/v1/auth/me` |
| Tenant | POST | `/api/v1/tenants` |
| Tenant | GET | `/api/v1/tenants` |
| Tenant | GET | `/api/v1/tenants/{tenantId}` |
| Tenant | GET | `/api/v1/tenants/{tenantId}/members` |
| Tenant | POST | `/api/v1/tenants/{tenantId}/members` |
| Organization | GET | `/api/v1/tenants/{tenantId}/companies` |
| Organization | POST | `/api/v1/tenants/{tenantId}/companies` |
| Organization | GET | `/api/v1/companies/{companyId}/branches` |
| Organization | POST | `/api/v1/companies/{companyId}/branches` |
| Solution Catalog | GET | `/api/v1/solutions` |
| Solution Catalog | GET | `/api/v1/solutions/{solutionId}/plans` |
| Solution Catalog | GET | `/api/v1/solutions/{solutionId}/addons` |
| Plans & Benefits | POST | `/api/v1/solutions/{solutionId}/plans` |
| Plans & Benefits | GET | `/api/v1/plans/{planId}/benefits` |
| Plans & Benefits | PUT | `/api/v1/plans/{planId}/benefits` |
| Add-ons | POST | `/api/v1/solutions/{solutionId}/addons` |
| Add-ons | GET | `/api/v1/addons/{addonId}/benefits` |
| Add-ons | PUT | `/api/v1/addons/{addonId}/benefits` |
| Subscription | POST | `/api/v1/tenants/{tenantId}/subscriptions` |
| Subscription | GET | `/api/v1/tenants/{tenantId}/subscriptions` |
| Subscription | POST | `/api/v1/subscriptions/{subscriptionId}/change-plan` |
| Subscription | POST | `/api/v1/subscriptions/{subscriptionId}/cancel` |
| Subscription | GET | `/api/v1/subscriptions/{subscriptionId}/addons` |
| Subscription | POST | `/api/v1/subscriptions/{subscriptionId}/addons` |
| Effective Entitlements | GET | `/api/v1/tenants/{tenantId}/solutions/{solutionId}/entitlements` |
| Effective Entitlements | GET | `/api/v1/tenants/{tenantId}/solutions/{solutionId}/entitlements/{benefitCode}` |
| Authorization | GET | `/api/v1/solutions/{solutionId}/roles` |
| Authorization | POST | `/api/v1/solutions/{solutionId}/roles` |
| Authorization | GET | `/api/v1/roles/{roleId}/permissions` |
| Authorization | PUT | `/api/v1/roles/{roleId}/permissions` |
| Tenant Custom Role | GET | `/api/v1/tenants/{tenantId}/solutions/{solutionId}/roles` |
| Tenant Custom Role | POST | `/api/v1/tenants/{tenantId}/solutions/{solutionId}/roles` |
| Tenant Custom Role | PUT | `/api/v1/tenants/{tenantId}/roles/{roleId}/permissions` |
| Authorization | POST | `/api/v1/tenants/{tenantId}/users/{userId}/role-assignments` |
| Authorization | POST | `/api/v1/authz/check` |
| Authorization | POST | `/api/v1/authz/check-batch` |
| Auth & Session | GET | `/api/v1/auth/tenants` |
| Auth & Session | POST | `/api/v1/auth/switch-tenant` |
| Service Auth | POST | `/api/v1/oauth/token` |
| Service Auth | GET | `/api/v1/.well-known/jwks.json` |
| Solution Clients | POST | `/api/v1/solutions/{solutionId}/clients` |
| Solution Clients | POST | `/api/v1/solution-clients/{clientId}/rotate-secret` |
| Webhooks | POST | `/api/v1/solutions/{solutionId}/webhooks` |
| Webhooks | GET | `/api/v1/events` |
| Usage | POST | `/api/v1/tenants/{tenantId}/solutions/{solutionId}/usage` |
| Usage | PUT | `/api/v1/tenants/{tenantId}/solutions/{solutionId}/usage/{benefitCode}` |
| Catalog Sync | PUT | `/api/v1/solutions/{solutionId}/permissions/sync` |
| License (`on_prem`) | GET | `/api/v1/installation` |
| License (`on_prem`) | GET | `/api/v1/license` |
| License (`on_prem`) | POST | `/api/v1/license` |
| License (`on_prem`) | GET | `/api/v1/license/raw` |
| License (`on_prem`) | GET | `/api/v1/license/usage-report` |

---

## 5. Prinsip Utama

Aplikasi produk mengambil **effective benefits** dari snapshot subscription tenant dan memeriksa **permission user** secara terpisah. Keduanya harus divalidasi di sisi server.

---

## 6. Changelog

| # | Perubahan |
|---|---|
| 1 | Composite FK `(id, solution_id)` untuk menjaga konsistensi solution di `plan_benefits`, `addon_benefits`, `subscriptions`, `subscription_addons`, `role_permissions` |
| 2 | Tabel baru `subscription_benefit_snapshots`; entitlement dihitung dari snapshot |
| 3 | `user_role_assignments`: FK ke `tenant_members`, unique `(user_id, role_id, scope_type, scope_id)`, aturan validasi role & scope |
| 4 | CHECK constraint untuk seluruh kolom status/enum |
| 5 | Index baru untuk FK yang belum ter-index |
| 6 | Soft delete (`deleted_at`) untuk `companies`, `branches`, `plans`, `addons`, `roles`; `updated_at` di `companies` & `branches` |
| 7 | Custom role per tenant per solution: `roles.tenant_id`, partial unique, endpoint `/tenants/{tenantId}/...roles` |
| 8 | `solution_clients` + scope service + `POST /oauth/token` (client credentials) |
| 9 | `signing_keys`, endpoint JWKS & openid-configuration, definisi claim user & service token |
| 10 | Events (outbox), `webhook_endpoints`, `webhook_deliveries`, katalog event, endpoint polling `/events` |
| 11 | `auth_sessions` (refresh token, tenant aktif), endpoint `/auth/tenants`, `/auth/switch-tenant`, manajemen sesi |
| 12 | Usage reporting: `benefits.metering` & `usage_reset`, `usage_records`, `tenant_usage`, entitlement memuat `used`/`remaining` |
| 13 | Catalog sync: `permissions.status`, `solutions.permission_catalog_version`, endpoint sync permission & system role |
| 14 | Deployment mode `cloud`/`on_prem`; licensing: `installations`, `licenses`, `license_checkins`, state `valid`/`grace`/`read_only`/`invalid`, endpoint license, daftar endpoint yang dinonaktifkan di `on_prem` |
