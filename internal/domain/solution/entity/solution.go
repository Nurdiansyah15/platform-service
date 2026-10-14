package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solution/valueobject"
)

type Solution struct {
	ID                       string
	Code                     valueobject.SolutionCode
	Name                     string
	Description              *string
	Status                   valueobject.SolutionStatus
	PermissionCatalogVersion *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type NewSolutionParams struct {
	Code        string
	Name        string
	Description *string
	Status      string // kosong → active
}

// NewSolution membuat solution baru yang tervalidasi.
func NewSolution(p NewSolutionParams) (*Solution, error) {
	code, err := valueobject.NewSolutionCode(p.Code)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeSolutionNameInvalid)
	}

	status := valueobject.SolutionStatusActive
	if p.Status != "" {
		if status, err = valueobject.NewSolutionStatus(p.Status); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	return &Solution{
		ID:          uuid.NewString(),
		Code:        code,
		Name:        name,
		Description: normalizeOptional(p.Description),
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *Solution) IsActive() bool {
	return s.Status == valueobject.SolutionStatusActive
}

// SetPermissionCatalogVersion mencatat versi katalog permission yang terakhir di-sync.
func (s *Solution) SetPermissionCatalogVersion(version string, now time.Time) {
	s.PermissionCatalogVersion = &version
	s.UpdatedAt = now.UTC()
}

// HasPermissionCatalogVersion — katalog versi ini sudah pernah di-sync (sync idempotent).
func (s *Solution) HasPermissionCatalogVersion(version string) bool {
	return s.PermissionCatalogVersion != nil && *s.PermissionCatalogVersion == version
}

func normalizeOptional(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}
