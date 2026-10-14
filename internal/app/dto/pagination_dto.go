package dto

// PaginationParams di-embed ke DTO query list. Field pointer agar bisa nil/opsional.
type PaginationParams struct {
	Page     *int    `form:"page"`
	Limit    *int    `form:"limit"`
	SortBy   *string `form:"sort_by"`
	SortType *string `form:"sort_type"`
}

type Meta struct {
	CurrentPage int64 `json:"current_page"`
	PerPage     int64 `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int64 `json:"total_pages"`
}

// NewMeta menghitung Meta dari hasil query ber-pagination.
func NewMeta(currentPage int64, perPage int, total int64) Meta {
	var totalPages int64
	if perPage > 0 {
		totalPages = (total + int64(perPage) - 1) / int64(perPage)
	}
	return Meta{CurrentPage: currentPage, PerPage: int64(perPage), Total: total, TotalPages: totalPages}
}
