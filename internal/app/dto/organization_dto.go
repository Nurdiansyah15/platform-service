package dto

import "time"

type CreateOrgUnitRequest struct {
	Name   string `json:"name" binding:"required,max=255"`
	Code   string `json:"code" binding:"required,max=100"`
	Status string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type ListOrgUnitQuery struct {
	Status *string `form:"status" binding:"omitempty,oneof=active inactive"`
	Search *string `form:"search"`
	PaginationParams
}

type CompanyIDParam struct {
	CompanyID string `uri:"companyId" binding:"required,uuid"`
}

type BranchResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
