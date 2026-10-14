package dto

import (
	"encoding/json"
	"time"
)

type CreateAddonRequest struct {
	Code        string  `json:"code" binding:"required,max=100"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description *string `json:"description"`
	// Price — angka JSON atau string angka, maks 2 desimal (mis. 25000 atau "25000.00").
	Price        json.RawMessage `json:"price" swaggertype:"string" example:"25000.00"`
	Currency     string          `json:"currency" binding:"omitempty,len=3"`
	BillingCycle string          `json:"billing_cycle" binding:"omitempty,oneof=monthly yearly"`
	IsActive     *bool           `json:"is_active"`
}

type ListAddonQuery struct {
	// IsActive hanya berlaku untuk platform admin; user lain selalu melihat add-on aktif.
	IsActive     *bool   `form:"is_active"`
	BillingCycle *string `form:"billing_cycle" binding:"omitempty,oneof=monthly yearly"`
	Search       *string `form:"search"`
	PaginationParams
}

type AddonIDParam struct {
	AddonID string `uri:"addonId" binding:"required,uuid"`
}

type AddonResponse struct {
	ID           string    `json:"id"`
	SolutionID   string    `json:"solution_id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Price        string    `json:"price" example:"25000.00"`
	Currency     string    `json:"currency"`
	BillingCycle string    `json:"billing_cycle"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
