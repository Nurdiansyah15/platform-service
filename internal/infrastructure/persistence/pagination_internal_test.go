package persistence

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"platform-service/internal/app/dto"
)

func TestResolvePaginationParams(t *testing.T) {
	sortable := map[string]string{"name": "t.name", "created_at": "t.created_at"}
	intp := func(v int) *int { return &v }
	strp := func(v string) *string { return &v }

	tests := []struct {
		name                         string
		params                       dto.PaginationParams
		wantLimit, wantOffset        int
		wantPage                     int64
		wantSortColumn, wantSortType string
	}{
		{"default", dto.PaginationParams{}, 20, 0, 1, "t.created_at", "DESC"},
		{"page & limit", dto.PaginationParams{Page: intp(3), Limit: intp(10)}, 10, 20, 3, "t.created_at", "DESC"},
		{"limit dibatasi max", dto.PaginationParams{Limit: intp(1000)}, 100, 0, 1, "t.created_at", "DESC"},
		{"nilai negatif → default", dto.PaginationParams{Page: intp(-1), Limit: intp(0)}, 20, 0, 1, "t.created_at", "DESC"},
		{"sort valid", dto.PaginationParams{SortBy: strp(" Name "), SortType: strp("asc")}, 20, 0, 1, "t.name", "ASC"},
		{"sort tidak di whitelist diabaikan", dto.PaginationParams{SortBy: strp("password_hash; DROP TABLE x"), SortType: strp("sideways")}, 20, 0, 1, "t.created_at", "DESC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset, page, col, typ := resolvePaginationParams(tt.params, 20, 100, sortable, "t.created_at", "desc")
			assert.Equal(t, tt.wantLimit, limit)
			assert.Equal(t, tt.wantOffset, offset)
			assert.Equal(t, tt.wantPage, page)
			assert.Equal(t, tt.wantSortColumn, col)
			assert.Equal(t, tt.wantSortType, typ)
		})
	}
}
