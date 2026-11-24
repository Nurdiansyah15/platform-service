package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeAssignmentNotFound        domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_NOT_FOUND"
	CodeAssignmentDuplicate       domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_DUPLICATE"
	CodeAssignmentNotMember       domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_NOT_MEMBER"
	CodeAssignmentRoleUnavailable domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_ROLE_UNAVAILABLE"
	CodeAssignmentNotSubscribed   domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_NOT_SUBSCRIBED"
	CodeScopeTypeInvalid          domainerr.Code = "DOMAIN_AUTHZ_SCOPE_TYPE_INVALID"
	CodeScopeInvalid              domainerr.Code = "DOMAIN_AUTHZ_SCOPE_INVALID"
	CodeCheckTargetRequired       domainerr.Code = "DOMAIN_AUTHZ_CHECK_TARGET_REQUIRED"
	CodeCheckSolutionUnavailable  domainerr.Code = "DOMAIN_AUTHZ_CHECK_SOLUTION_UNAVAILABLE"
	// CodeSolutionRequired — token user wajib memilih solution_id (token service memakai solution client).
	CodeSolutionRequired domainerr.Code = "DOMAIN_AUTHZ_SOLUTION_REQUIRED"
	// CodeAdminPermissionDenied — token user tidak punya permission platform untuk
	// operasi administrasi company (mis. branch.manage).
	CodeAdminPermissionDenied  domainerr.Code = "DOMAIN_AUTHZ_ADMIN_PERMISSION_DENIED"
	CodeAuthzPersistenceFailed domainerr.Code = "DOMAIN_AUTHZ_PERSISTENCE_FAILED"
	CodeAuthzQueryFailed       domainerr.Code = "DOMAIN_AUTHZ_QUERY_FAILED"
)

// MaxBatchChecks — batas jumlah cek per POST /authz/check-batch.
const MaxBatchChecks = 100

// ScopeIDMaxLength — panjang maksimum scope_id (scope custom memakai id milik solution).
const ScopeIDMaxLength = 150

// Permission platform: administrasi company. Permission ini milik katalog solution console
// (CONSOLE_SOLUTION_CODE) sehingga memakai mesin RBAC yang sama (role, assignment, scope
// company ⊇ branch). Token user wajib memilikinya untuk operasi tulis data company;
// token service dan platform admin dibebaskan.
const (
	PermCompanyManage = "company.manage" // ubah info company
	PermMemberManage  = "member.manage"  // buat user, tambah / ubah / keluarkan member
	PermBranchManage  = "branch.manage"  // tambah / ubah / hapus branch
	PermRoleManage    = "role.manage"    // buat / ubah / hapus role custom & permission-nya
	PermRoleAssign    = "role.assign"    // beri / cabut role

	// CompanyAdminRoleCode — role global sistem di solution console berisi semua permission
	// platform. Diberikan otomatis ke pembuat company.
	CompanyAdminRoleCode = "company_admin"
	// PlatformCatalogVersion — versi katalog permission platform di solution console.
	PlatformCatalogVersion = "platform-2026.11.24"
)

// PlatformPermissionGroup — satu group tampilan katalog permission platform.
type PlatformPermissionGroup struct {
	Code, Name, Description string
}

// Group permission platform (urut tampilan).
const (
	PlatformGroupOrganization = "organization"
	PlatformGroupAccess       = "access"
)

var PlatformPermissionGroups = []PlatformPermissionGroup{
	{Code: PlatformGroupOrganization, Name: "Organisasi", Description: "Info company, anggota, dan cabang."},
	{Code: PlatformGroupAccess, Name: "Role & akses", Description: "Role custom dan penugasan role."},
}

// PlatformPermission — satu entri katalog permission platform. Scope tidak dibatasi
// (berlaku di company & branch).
type PlatformPermission struct {
	Code, Name, Description, Group string
}

// PlatformPermissions — katalog permission platform (urut tampilan).
var PlatformPermissions = []PlatformPermission{
	{PermCompanyManage, "Kelola info company", "Mengubah nama dan slug company.", PlatformGroupOrganization},
	{PermMemberManage, "Kelola anggota", "Membuat akun user, menambah, mengubah status, dan mengeluarkan anggota.", PlatformGroupOrganization},
	{PermBranchManage, "Kelola cabang", "Menambah, mengubah, dan menghapus cabang.", PlatformGroupOrganization},
	{PermRoleManage, "Kelola role custom", "Membuat, mengubah, dan menghapus role custom beserta permission-nya.", PlatformGroupAccess},
	{PermRoleAssign, "Beri & cabut role", "Memberi dan mencabut role anggota pada scope yang dimiliki.", PlatformGroupAccess},
}
