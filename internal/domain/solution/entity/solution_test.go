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

func TestOriginOf(t *testing.T) {
	tests := []struct {
		raw    string
		want   string
		wantOK bool
	}{
		{"https://Kasir.Example.com/auth/callback?x=1", "https://kasir.example.com", true},
		{"https://kasir.example.com:443/", "https://kasir.example.com", true},
		{"http://localhost:80", "http://localhost", true},
		{"http://localhost:3000", "http://localhost:3000", true},
		{"https://kasir.example.com:8443/", "https://kasir.example.com:8443", true},
		{"HTTPS://kasir.example.com", "https://kasir.example.com", true},
		{"null", "", false},
		{"", "", false},
		{"/relative", "", false},
		{"ftp://kasir.example.com", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, ok := entity.OriginOf(tt.raw)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSolution_Origins(t *testing.T) {
	launch := "https://kasir.example.com/"
	redirects := []string{
		"https://kasir.example.com/auth/callback",
		"https://auth.example.com/cb",
		"http://localhost:4000/cb",
	}
	s, err := entity.NewSolution(entity.NewSolutionParams{
		Code: "kasir", Name: "Kasir",
		Links: entity.AppLinks{LaunchURL: &launch, RedirectURIs: &redirects},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"https://kasir.example.com", "https://auth.example.com", "http://localhost:4000"}, s.Origins())
	assert.Equal(t, redirects, s.RedirectURIs, "RedirectURIs tidak ikut berubah")

	empty, err := entity.NewSolution(entity.NewSolutionParams{Code: "kosong", Name: "Kosong"})
	require.NoError(t, err)
	assert.Empty(t, empty.Origins())
}
