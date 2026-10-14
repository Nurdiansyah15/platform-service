package dto

import "time"

type SyncPermissionsRequest struct {
	// Version — versi katalog; sama dengan versi tersimpan → tidak ada perubahan.
	Version     string                 `json:"version" binding:"required,max=50"`
	Permissions []PermissionDefinition `json:"permissions" binding:"required,dive"`
}

type PermissionDefinition struct {
	Code        string  `json:"code" binding:"required,max=150"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
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
}

type ListPermissionQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=active deprecated"`
	Search *string `form:"search"`
	PaginationParams
}

type PermissionResponse struct {
	ID          string    `json:"id"`
	SolutionID  string    `json:"solution_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
