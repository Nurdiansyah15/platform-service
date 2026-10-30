package dto

import "time"

type CreateSolutionRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
	Status      string  `json:"status" binding:"omitempty,oneof=active inactive"`
	// LaunchURL — halaman awal solution untuk launcher / menu apps (https; http hanya localhost).
	LaunchURL *string `json:"launch_url" binding:"omitempty,max=2000"`
	IconURL   *string `json:"icon_url" binding:"omitempty,max=2000"`
	// RedirectURIs — allowlist redirect_uri SSO, dicocokkan persis (maks. 10).
	RedirectURIs *[]string `json:"redirect_uris" binding:"omitempty,max=10,dive,max=2000"`
}

type ListSolutionQuery struct {
	// Status hanya berlaku untuk platform admin; user lain selalu melihat solution active.
	Status *string `form:"status" binding:"omitempty,oneof=active inactive"`
	Search *string `form:"search"`
	PaginationParams
}

// UpdateSolutionRequest — field nil tidak diubah; launch_url / icon_url "" dan redirect_uris []
// mengosongkan nilainya. Kode solution tidak bisa diubah.
type UpdateSolutionRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Status      *string `json:"status" binding:"omitempty,oneof=active inactive"`
	// LaunchURL — halaman awal solution untuk launcher / menu apps (https; http hanya localhost).
	LaunchURL *string `json:"launch_url" binding:"omitempty,max=2000"`
	IconURL   *string `json:"icon_url" binding:"omitempty,max=2000"`
	// RedirectURIs — allowlist redirect_uri SSO, dicocokkan persis (maks. 10).
	RedirectURIs *[]string `json:"redirect_uris" binding:"omitempty,max=10,dive,max=2000"`
}

type SolutionIDParam struct {
	SolutionID string `uri:"solutionId" binding:"required,uuid"`
}

type SolutionResponse struct {
	ID                       string    `json:"id"`
	Code                     string    `json:"code"`
	Name                     string    `json:"name"`
	Description              *string   `json:"description"`
	Status                   string    `json:"status"`
	PermissionCatalogVersion *string   `json:"permission_catalog_version"`
	LaunchURL                *string   `json:"launch_url"`
	IconURL                  *string   `json:"icon_url"`
	RedirectURIs             []string  `json:"redirect_uris"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
