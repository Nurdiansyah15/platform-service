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

// ── Solution ────────────────────────────────────────────────────────────────
export interface SolutionResponse {
  id: string
  code: string
  name: string
  description: string | null
  status: 'active' | 'inactive'
  permission_catalog_version: string | null
  created_at: string
  updated_at: string
}

export interface CreateSolutionRequest {
  code: string
  name: string
  description?: string | null
  status?: 'active' | 'inactive'
}

export interface ListSolutionQuery extends PaginationParams {
  status?: string
  search?: string
}

// ── Company & Organization ──────────────────────────────────────────────────
export interface CompanyResponse {
  id: string
  name: string
  slug: string
  status: string
  created_at: string
  updated_at: string
}

export interface CreateCompanyRequest {
  name: string
  slug: string
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
}

export interface AddCompanyMemberRequest {
  user_id?: string
  email?: string
  status?: 'active' | 'invited'
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

export interface CreateOrgUnitRequest {
  name: string
  code: string
  status?: 'active' | 'inactive'
}

// ── Users (platform admin) ──────────────────────────────────────────────────
export interface CreateUserRequest {
  email: string
  name: string
  password?: string
  status?: 'active' | 'inactive' | 'suspended'
  is_platform_admin?: boolean
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
}

export interface CreateRoleRequest {
  code: string
  name: string
  description?: string | null
}

export interface RolePermissionResponse {
  id: string
  code: string
  name: string
  status: string
}

export interface PermissionResponse {
  id: string
  solution_id: string
  code: string
  name: string
  description: string | null
  status: 'active' | 'deprecated'
  created_at: string
  updated_at: string
}

export interface ListPermissionQuery extends PaginationParams {
  status?: string
  search?: string
}

export interface ReplaceRolePermissionsRequest {
  permission_codes: string[]
}

export interface SyncPermissionsRequest {
  version: string
  permissions: { code: string; name: string; description?: string | null }[]
}

export interface SyncPermissionsResponse {
  version: string
  changed: boolean
  created: number
  updated: number
  reactivated: number
  deprecated: number
  unchanged: number
}

// ── Authorization ───────────────────────────────────────────────────────────
export interface CreateRoleAssignmentRequest {
  role_id: string
  scope_type?: 'company' | 'branch'
  scope_id?: string
}

export interface RoleAssignmentResponse {
  id: string
  company_id: string
  user_id: string
  role_id: string
  role_code: string
  solution_id: string
  scope_type: string
  scope_id: string
  created_at: string
}

export interface AuthzCheckRequest {
  company_id?: string
  user_id?: string
  permission: string
  scope_type?: 'company' | 'branch'
  scope_id?: string
}

export interface AuthzCheckItem {
  permission: string
  scope_type?: 'company' | 'branch'
  scope_id?: string
}

export interface AuthzCheckBatchRequest {
  company_id?: string
  user_id?: string
  checks: AuthzCheckItem[]
}

export interface AuthzCheckResult {
  permission: string
  scope_type: string
  scope_id: string
  allowed: boolean
  granted_by: string[] | null
}

export interface AuthzCheckBatchResponse {
  company_id: string
  user_id: string
  results: AuthzCheckResult[]
}

// ── Error klien ─────────────────────────────────────────────────────────────
export interface AppError {
  status: number
  code: string
  message: string
  fields: Record<string, string>
}
