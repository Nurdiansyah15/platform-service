package dto

import "time"

type CreateSolutionRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
	Status      string  `json:"status" binding:"omitempty,oneof=active inactive"`
}

type ListSolutionQuery struct {
	// Status hanya berlaku untuk platform admin; user lain selalu melihat solution active.
	Status *string `form:"status" binding:"omitempty,oneof=active inactive"`
	Search *string `form:"search"`
	PaginationParams
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
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
