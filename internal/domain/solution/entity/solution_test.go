package entity_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solution/entity"
	"platform-service/internal/domain/solution/valueobject"
	"platform-service/internal/testhelper"
)

func TestNewSolution_Valid(t *testing.T) {
	desc := "  Aplikasi HR  "
	s, err := entity.NewSolution(entity.NewSolutionParams{Code: " easy-hr ", Name: " Easy HR ", Description: &desc})

	require.NoError(t, err)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, "easy-hr", s.Code.Value())
	assert.Equal(t, "Easy HR", s.Name)
	require.NotNil(t, s.Description)
	assert.Equal(t, "Aplikasi HR", *s.Description)
	assert.Equal(t, valueobject.SolutionStatusActive, s.Status)
	assert.True(t, s.IsActive())
}

func TestNewSolution_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		params entity.NewSolutionParams
		want   domainerr.Code
	}{
		{"code kosong", entity.NewSolutionParams{Code: "", Name: "X"}, constant.CodeSolutionCodeInvalid},
		{"code huruf besar", entity.NewSolutionParams{Code: "Easy-HR", Name: "X"}, constant.CodeSolutionCodeInvalid},
		{"code spasi", entity.NewSolutionParams{Code: "easy hr", Name: "X"}, constant.CodeSolutionCodeInvalid},
		{"code dash di ujung", entity.NewSolutionParams{Code: "easy-", Name: "X"}, constant.CodeSolutionCodeInvalid},
		{"code terlalu panjang", entity.NewSolutionParams{Code: strings.Repeat("a", 101), Name: "X"}, constant.CodeSolutionCodeInvalid},
		{"name kosong", entity.NewSolutionParams{Code: "easy-hr", Name: "   "}, constant.CodeSolutionNameInvalid},
		{"name terlalu panjang", entity.NewSolutionParams{Code: "easy-hr", Name: strings.Repeat("a", 256)}, constant.CodeSolutionNameInvalid},
		{"status tidak dikenal", entity.NewSolutionParams{Code: "easy-hr", Name: "X", Status: "deleted"}, constant.CodeSolutionStatusInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.NewSolution(tt.params)
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}
