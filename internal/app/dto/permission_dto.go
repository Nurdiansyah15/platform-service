package dto

import "time"

type SyncPermissionsRequest struct {
	// Version — versi katalog; sama dengan versi tersimpan → tidak ada perubahan.
	Version string `json:"version" binding:"required,max=50"`
	// ScopeTypes — scope type custom (opsional). company & branch selalu tersedia.
	ScopeTypes []ScopeTypeDefinition `json:"scope_types" binding:"omitempty,dive"`
	// Groups — permission group untuk tampilan (opsional). Urutan array = urutan tampil.
	Groups      []PermissionGroupDefinition `json:"groups" binding:"omitempty,dive"`
	Permissions []PermissionDefinition      `json:"permissions" binding:"required,dive"`
}

type PermissionDefinition struct {
	Code        string  `json:"code" binding:"required,max=150"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
	// Group — code group di `groups` (opsional).
	Group *string `json:"group" binding:"omitempty,max=100"`
}

type PermissionGroupDefinition struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
}

type ScopeTypeDefinition struct {
	Code        string  `json:"code" binding:"required,max=30"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
	// Parent — company (default) atau branch.
	Parent string `json:"parent" binding:"omitempty,oneof=company branch"`
}

// SyncCounts — ringkasan perubahan satu jenis entri katalog.
type SyncCounts struct {
	Created     int `json:"created"`
	Updated     int `json:"updated"`
	Reactivated int `json:"reactivated"`
	Deprecated  int `json:"deprecated"`
	Unchanged   int `json:"unchanged"`
}

type SyncPermissionsResponse struct {
	Version string `json:"version"`
	// Changed — false bila versi sama dengan versi tersimpan (sync dilewati).
	Changed     bool `json:"changed"`
	Created     int  `json:"created"`
	Updated     int  `json:"updated"`
	Reactivated int  `json:"reactivated"`
	Deprecated  int  `json:"deprecated"`
	Unchanged   int  `json:"unchanged"`
	// Groups & ScopeTypes — ringkasan sync group dan scope type custom.
	Groups     SyncCounts `json:"groups"`
	ScopeTypes SyncCounts `json:"scope_types"`
}

type ListPermissionQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=active deprecated"`
	Search *string `form:"search"`
	// Group — filter code group; "-" = permission tanpa grup.
	Group *string `form:"group" binding:"omitempty,max=100"`
	// Source — sync | manual.
	Source *string `form:"source" binding:"omitempty,oneof=sync manual"`
	PaginationParams
}

// PermissionGroupRef — ringkasan group pada permission.
type PermissionGroupRef struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type PermissionResponse struct {
	ID          string  `json:"id"`
	SolutionID  string  `json:"solution_id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
	// Source — sync (dikelola sync katalog, read-only di platform) | manual (dibuat platform admin).
	Source string `json:"source"`
	// Group — nil bila permission tanpa grup.
	Group     *PermissionGroupRef `json:"group"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type PermissionGroupResponse struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	Status      string  `json:"status"`
	// Source — sync (dikelola sync katalog, read-only di platform) | manual (dibuat platform admin).
	Source string `json:"source"`
}

type PermissionGroupIDParam struct {
	GroupID string `uri:"groupId" binding:"required,uuid"`
}

// CreatePermissionGroupRequest — group manual (dibuat platform admin, tidak disentuh sync).
type CreatePermissionGroupRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	// SortOrder — urutan tampil (opsional; default setelah group terakhir).
	SortOrder *int `json:"sort_order" binding:"omitempty,min=0,max=100000"`
}

// UpdatePermissionGroupRequest — hanya group manual. Field nil tidak diubah; description ""
// = dikosongkan. Code tidak bisa diubah.
type UpdatePermissionGroupRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	SortOrder   *int    `json:"sort_order" binding:"omitempty,min=0,max=100000"`
	// Status — deprecated = hapus (tidak bisa dipilih untuk permission baru); active = aktifkan lagi.
	Status *string `json:"status" binding:"omitempty,oneof=active deprecated"`
}

type ListCatalogQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=active deprecated"`
}

type ScopeTypeResponse struct {
	// ID — kosong untuk scope built-in (company / branch).
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	// Parent — scope induk: "" untuk company, "company" / "branch" untuk lainnya.
	Parent string `json:"parent"`
	// Builtin — company & branch (dikelola platform; scope_id divalidasi platform).
	Builtin   bool   `json:"builtin"`
	SortOrder int    `json:"sort_order"`
	Status    string `json:"status"`
	// Source — builtin | sync (dikelola sync katalog, read-only di platform) | manual (dibuat platform admin).
	Source string `json:"source"`
}

type ScopeTypeIDParam struct {
	ScopeTypeID string `uri:"scopeTypeId" binding:"required,uuid"`
}

// CreateScopeTypeRequest — scope type custom manual (dibuat platform admin, tidak disentuh sync).
type CreateScopeTypeRequest struct {
	Code        string  `json:"code" binding:"required,max=30"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	// Parent — company (default) atau branch.
	Parent string `json:"parent" binding:"omitempty,oneof=company branch"`
	// SortOrder — urutan tampil (opsional; default setelah scope type terakhir).
	SortOrder *int `json:"sort_order" binding:"omitempty,min=0,max=100000"`
}

// UpdateScopeTypeRequest — hanya scope type manual. Field nil tidak diubah; description "" =
// dikosongkan. Code tidak bisa diubah.
type UpdateScopeTypeRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Parent      *string `json:"parent" binding:"omitempty,oneof=company branch"`
	SortOrder   *int    `json:"sort_order" binding:"omitempty,min=0,max=100000"`
	// Status — deprecated = hapus (tidak bisa dipakai assignment baru; yang lama tetap berlaku).
	Status *string `json:"status" binding:"omitempty,oneof=active deprecated"`
}

type PermissionIDParam struct {
	PermissionID string `uri:"permissionId" binding:"required,uuid"`
}

// CreatePermissionRequest — permission manual (dibuat platform admin, tidak disentuh sync).
type CreatePermissionRequest struct {
	Code        string  `json:"code" binding:"required,max=150"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	// Group — code permission group solution (opsional).
	Group *string `json:"group" binding:"omitempty,max=100"`
}

// UpdatePermissionRequest — hanya permission manual. Field nil tidak diubah; description /
// group "" = dikosongkan. Code tidak bisa diubah.
type UpdatePermissionRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Group       *string `json:"group" binding:"omitempty,max=100"`
	// Status — deprecated = "hapus" (tidak bisa ditambahkan ke role baru); active = aktifkan lagi.
	Status *string `json:"status" binding:"omitempty,oneof=active deprecated"`
}
