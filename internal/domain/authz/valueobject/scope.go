package valueobject

import (
	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
)

type ScopeType string

const (
	ScopeTenant  ScopeType = "tenant"
	ScopeCompany ScopeType = "company"
	ScopeBranch  ScopeType = "branch"
)

// NewScopeType — kosong → tenant.
func NewScopeType(raw string) (ScopeType, error) {
	if raw == "" {
		return ScopeTenant, nil
	}
	switch s := ScopeType(raw); s {
	case ScopeTenant, ScopeCompany, ScopeBranch:
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
