package valueobject

import (
	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
)

type ScopeType string

const (
	ScopeCompany ScopeType = "company"
	ScopeBranch  ScopeType = "branch"
)

// NewScopeType — kosong → company.
func NewScopeType(raw string) (ScopeType, error) {
	if raw == "" {
		return ScopeCompany, nil
	}
	switch s := ScopeType(raw); s {
	case ScopeCompany, ScopeBranch:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeScopeTypeInvalid)
	}
}

func (s ScopeType) String() string { return string(s) }

// Scope adalah lingkup berlakunya role / permission.
type Scope struct {
	Type ScopeType
	ID   string
}
