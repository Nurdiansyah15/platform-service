// Tipe request/response API platform-service (ditulis manual, mengikuti DTO Go).

export interface ApiEnvelope<T> {
  status: 'success' | 'error'
  status_code: number
  message?: string
  data: T
  meta?: Meta | null
}

export interface ApiErrorBody {
  status: 'error'
  status_code: number
  error_code: string
  errors: string | Record<string, string> | null
}

export interface Meta {
  current_page: number
  per_page: number
  total: number
  total_pages: number
}

export interface Page<T> {
  data: T
  meta: Meta | null
}

export interface PaginationParams {
  page?: number
  limit?: number
  sort_by?: string
  sort_type?: 'asc' | 'desc'
}

// ── Auth ────────────────────────────────────────────────────────────────────
export interface LoginRequest {
  email: string
  password: string
  solution: string
}

export interface AuthTokenResponse {
  access_token: string
  token_type: string
  expires_in: number
  access_token_expires_at: string
  refresh_token: string
  refresh_token_expires_at: string
  session_id: string
  company_id: string | null
}

export interface UserResponse {
  id: string
  email: string
  name: string
  status: string
  is_platform_admin: boolean
  has_password: boolean
  created_at: string
  updated_at: string
}

export interface SolutionSummary {
  id: string
  code: string
  name: string
}

export interface SessionResponse {
  id: string
  solution: SolutionSummary
  company_id: string | null
  expires_at: string
  created_at: string
}

export interface MeResponse {
  user: UserResponse
  session: SessionResponse
}

export interface SelectableCompany {
  id: string
  name: string
  slug: string
  status: string
  is_active: boolean
}

// ── SSO & apps ──────────────────────────────────────────────────────────────
/** Shortcut solution (GET /auth/apps). */
export interface AppResponse {
  id: string
  code: string
  name: string
  description: string | null
  icon_url: string | null
  launch_url: string
  is_current: boolean
}

export interface AuthorizeRequest {
  solution: string
  redirect_uri: string
  code_challenge: string
  code_challenge_method: 'S256'
  state?: string
  company_id?: string
}

export interface AuthorizeResponse {
  redirect_to: string
  code: string
  expires_at: string
}

// ── Solution ────────────────────────────────────────────────────────────────
export interface SolutionResponse {
  id: string
  code: string
  name: string
  description: string | null
  status: 'active' | 'inactive'
  permission_catalog_version: string | null
  /** Halaman awal solution di launcher / menu apps. */
  launch_url: string | null
  icon_url: string | null
  /** Allowlist redirect_uri SSO (dicocokkan persis). */
  redirect_uris: string[]
  created_at: string
  updated_at: string
}

export interface CreateSolutionRequest {
  code: string
  name: string
  description?: string | null
  status?: 'active' | 'inactive'
  launch_url?: string
  icon_url?: string
  redirect_uris?: string[]
}

/** "" / [] mengosongkan tautan aplikasi. */
export interface UpdateSolutionRequest {
  name?: string
  description?: string
  status?: 'active' | 'inactive'
  launch_url?: string
  icon_url?: string
  redirect_uris?: string[]
}

export interface AdminSummary {
  users: { active: number; inactive: number; suspended: number; admins: number }
  companies: { active: number; suspended: number; closed: number }
  solutions: { active: number; inactive: number }
  active_clients: number
}

export interface ListSolutionQuery extends PaginationParams {
  status?: string
  search?: string
}

// ── Solution Client ─────────────────────────────────────────────────────────
export interface SolutionClientResponse {
  id: string
  solution_id: string
  name: string
  client_id: string
  scopes: string[]
  status: 'active' | 'revoked'
  last_used_at: string | null
  created_at: string
  updated_at: string
  revoked_at: string | null
}

/** Dikembalikan saat create / rotate-secret; secret hanya terlihat sekali. */
export interface SolutionClientWithSecretResponse extends SolutionClientResponse {
  client_secret: string
}

export interface CreateSolutionClientRequest {
  name: string
  scopes: string[]
}

export interface UpdateSolutionClientRequest {
  name?: string
  scopes?: string[]
}

/** Scope service yang dikenal platform (spec §2.8). */
export const SERVICE_SCOPES = [
  'users:read',
  'users:write',
  'org:read',
  'org:write',
  'authz:check',
  'roles:write',
  'entitlements:read',
  'usage:write',
  'catalog:sync',
  'events:read',
  'license:read',
] as const

// ── Company & Organization ──────────────────────────────────────────────────
export interface CompanyResponse {
  id: string
  name: string
  slug: string
  status: string
  created_at: string
  updated_at: string
  /** Jumlah member active (hanya diisi di list). */
  member_count?: number
}

export interface CompanySummary {
  members: { active: number; invited: number; inactive: number }
  branches: number
  custom_roles: number
  role_assignments: number
}

export interface MemberCandidate {
  id: string
  name: string
  email: string
}

export interface CreateCompanyRequest {
  name: string
  slug: string
}

export type CompanyStatus = 'active' | 'suspended' | 'closed'

/** Field yang tidak dikirim tidak diubah. `status` hanya boleh diubah platform admin. */
export interface UpdateCompanyRequest {
  name?: string
  slug?: string
  status?: CompanyStatus
}

export interface ListCompanyQuery extends PaginationParams {
  status?: string
  search?: string
}

export interface CompanyMemberResponse {
  company_id: string
  user_id: string
  email: string
  name: string
  status: string
  joined_at: string
  /** Jumlah role assignment di company (hanya diisi di list). */
  role_count?: number
}

export interface AddCompanyMemberRequest {
  user_id?: string
  email?: string
  status?: 'active' | 'invited'
}

export type MemberStatus = 'active' | 'invited' | 'inactive'

/** `inactive` = member dikeluarkan (role assignment di company ikut dihapus). */
export interface UpdateCompanyMemberRequest {
  status: MemberStatus
}

export interface BranchResponse {
  id: string
  company_id: string
  name: string
  code: string
  status: string
  created_at: string
  updated_at: string
}

export interface UpdateOrgUnitRequest {
  name?: string
  code?: string
  status?: 'active' | 'inactive'
}

export interface CreateOrgUnitRequest {
  name: string
  code: string
  status?: 'active' | 'inactive'
}

// ── Users ───────────────────────────────────────────────────────────────────
export type UserStatus = 'active' | 'inactive' | 'suspended'

export interface CreateUserRequest {
  email: string
  name: string
  password?: string
  status?: UserStatus
  is_platform_admin?: boolean
}

export interface ListUserQuery extends PaginationParams {
  status?: UserStatus
  is_platform_admin?: boolean
  search?: string
}

/** Field yang tidak dikirim tidak diubah. `password` = reset password oleh admin. */
export interface UpdateUserRequest {
  email?: string
  name?: string
  password?: string
  is_platform_admin?: boolean
}

export interface UserMembership {
  company_id: string
  company_name: string
  company_slug: string
  company_status: string
  member_status: MemberStatus
  joined_at: string
}

export interface UserDetailResponse extends UserResponse {
  memberships: UserMembership[]
}

/** Ganti password wajib menyertakan `current_password`. */
export interface UpdateMeRequest {
  name?: string
  password?: string
  current_password?: string
}

// ── Roles & Permissions ─────────────────────────────────────────────────────
export interface RoleResponse {
  id: string
  solution_id: string
  company_id: string | null
  code: string
  name: string
  description: string | null
  is_system: boolean
  scope: 'system' | 'global' | 'company'
  created_at: string
  updated_at: string
  permission_count: number
  member_count: number
}

export interface UpdateRoleRequest {
  name?: string
  description?: string
}

export interface RoleHolder {
  assignment_id: string
  user_id: string
  name: string
  email: string
  scope_values: Record<string, string>
  scope_labels: Record<string, string>
  scopes: AssignmentScope[]
  created_at: string
}

/** Satu nilai scope assignment beserta nama tampilannya. */
export interface AssignmentScope {
  type: string
  type_name: string
  value: string
  name: string
}

export interface CreateRoleRequest {
  code: string
  name: string
  description?: string | null
}

export interface PermissionGroupRef {
  code: string
  name: string
  sort_order: number
}

export interface RolePermissionResponse {
  id: string
  code: string
  name: string
  description: string | null
  status: string
  group: PermissionGroupRef | null
  /** Scope tempat permission berlaku untuk role ini: company | branch | scope type custom. */
  scope_type: string
}

export interface PermissionResponse {
  id: string
  solution_id: string
  code: string
  name: string
  description: string | null
  status: 'active' | 'deprecated'
  /** sync = dikelola sync katalog (read-only di platform); manual = dibuat platform admin. */
  source: 'sync' | 'manual'
  /** null = tanpa grup (UI mengelompokkan by prefix code). */
  group: PermissionGroupRef | null
  created_at: string
  updated_at: string
}

export interface ListPermissionQuery extends PaginationParams {
  status?: string
  search?: string
  /** Code group; '-' = tanpa grup. */
  group?: string
  source?: 'sync' | 'manual'
}

export interface CreatePermissionRequest {
  code: string
  name: string
  description?: string | null
  group?: string | null
}

/** Hanya permission manual. description / group '' = dikosongkan; status deprecated = hapus. */
export interface UpdatePermissionRequest {
  name?: string
  description?: string
  group?: string
  status?: 'active' | 'deprecated'
}

export interface PermissionGroupResponse {
  id: string
  code: string
  name: string
  description: string | null
  sort_order: number
  status: 'active' | 'deprecated'
  source: 'sync' | 'manual'
}

export interface CreatePermissionGroupRequest {
  code: string
  name: string
  description?: string | null
  sort_order?: number
}

/** Hanya group manual. description '' = dikosongkan; status deprecated = hapus. */
export interface UpdatePermissionGroupRequest {
  name?: string
  description?: string
  sort_order?: number
  status?: 'active' | 'deprecated'
}

export interface ScopeTypeResponse {
  /** Kosong untuk built-in (company / branch). */
  id: string
  code: string
  name: string
  description: string | null
  /** '' untuk company; 'company' | 'branch' untuk lainnya. */
  parent: '' | 'company' | 'branch'
  builtin: boolean
  sort_order: number
  status: 'active' | 'deprecated'
  source: 'builtin' | 'sync' | 'manual'
}

export interface CreateScopeTypeRequest {
  code: string
  name: string
  description?: string | null
  parent?: 'company' | 'branch'
  sort_order?: number
}

/** Hanya scope type manual. description '' = dikosongkan; status deprecated = hapus. */
export interface UpdateScopeTypeRequest {
  name?: string
  description?: string
  parent?: 'company' | 'branch'
  sort_order?: number
  status?: 'active' | 'deprecated'
}

export interface ReplaceRolePermissionsRequest {
  permissions: { code: string; scope_type?: string }[]
}

export interface SyncPermissionsRequest {
  version: string
  scope_types?: { code: string; name: string; description?: string | null; parent?: 'company' | 'branch' }[]
  groups?: { code: string; name: string; description?: string | null }[]
  permissions: { code: string; name: string; description?: string | null; group?: string | null }[]
}

export interface SyncCounts {
  created: number
  updated: number
  reactivated: number
  deprecated: number
  unchanged: number
}

export interface SyncPermissionsResponse {
  version: string
  changed: boolean
  created: number
  updated: number
  reactivated: number
  deprecated: number
  unchanged: number
  groups: SyncCounts
  scope_types: SyncCounts
}

// ── Authorization ───────────────────────────────────────────────────────────
export interface CreateRoleAssignmentRequest {
  role_id: string
  /** Nilai untuk setiap scope_type non-company yang dipakai permission role. */
  scope_values?: Record<string, string>
  /** Nama tampilan opsional untuk nilai scope custom. */
  scope_labels?: Record<string, string>
}

export interface RoleAssignmentResponse {
  id: string
  company_id: string
  user_id: string
  role_id: string
  role_code: string
  solution_id: string
  scope_values: Record<string, string>
  scope_labels: Record<string, string>
  created_at: string
}

/** Assignment beserta info role & scope (GET .../role-assignments). */
export interface RoleAssignmentDetail {
  id: string
  company_id: string
  user_id: string
  role_id: string
  role_code: string
  role_name: string
  role_is_global: boolean
  solution_id: string
  solution_code: string
  scope_values: Record<string, string>
  scope_labels: Record<string, string>
  scopes: AssignmentScope[]
  /** scope_type unik permission role saat ini — dasar level role.assign untuk mencabut. */
  role_scope_types: string[]
  created_at: string
}

export interface EffectivePermission {
  permission: string
  name: string
  group: PermissionGroupRef | null
  grants: { role_code: string; scope_type: string; scope_id: string }[]
}

// ── Error klien ─────────────────────────────────────────────────────────────
export interface AppError {
  status: number
  code: string
  message: string
  fields: Record<string, string>
}
