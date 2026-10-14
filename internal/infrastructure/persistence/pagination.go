package persistence

import (
	"strings"

	"platform-service/internal/app/dto"
)

// resolvePaginationParams menormalisasi parameter pagination:
//   - page < 1 → 1, limit < 1 → defaultLimit, limit > maxLimit → maxLimit
//   - sort_by hanya diterima bila ada di sortable (key publik → kolom SQL), mencegah SQL injection
//   - sort_type hanya ASC/DESC
func resolvePaginationParams(params dto.PaginationParams, defaultLimit, maxLimit int, sortable map[string]string, defaultSortColumn, defaultSortType string) (limit int, offset int, currentPage int64, sortColumn string, sortType string) {
	page := 1
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}

	limit = defaultLimit
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset = (page - 1) * limit
	currentPage = int64(page)

	sortColumn = defaultSortColumn
	if params.SortBy != nil {
		if mapped, ok := sortable[strings.ToLower(strings.TrimSpace(*params.SortBy))]; ok {
			sortColumn = mapped
		}
	}

	sortType = strings.ToUpper(strings.TrimSpace(defaultSortType))
	if sortType != "ASC" && sortType != "DESC" {
		sortType = "DESC"
	}
	if params.SortType != nil {
		switch strings.ToLower(strings.TrimSpace(*params.SortType)) {
		case "asc":
			sortType = "ASC"
		case "desc":
			sortType = "DESC"
		}
	}

	return limit, offset, currentPage, sortColumn, sortType
}
