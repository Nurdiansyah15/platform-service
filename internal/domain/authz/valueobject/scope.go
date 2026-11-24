package valueobject

import (
	"regexp"

	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
)

// ScopeType — company & branch built-in; selain itu scope type custom yang didefinisikan
// solution di katalog permission (mis. "department"). Keberadaan scope type custom
// divalidasi ScopeValidator terhadap katalog solution.
type ScopeType string

const (
	ScopeCompany ScopeType = "company"
	ScopeBranch  ScopeType = "branch"
)

const scopeTypeMaxLength = 30

var scopeTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// NewScopeType — kosong → company; selain itu hanya validasi format.
func NewScopeType(raw string) (ScopeType, error) {
	if raw == "" {
		return ScopeCompany, nil
	}
	if len(raw) > scopeTypeMaxLength || !scopeTypePattern.MatchString(raw) {
		return "", domainerr.New(constant.CodeScopeTypeInvalid)
	}
	return ScopeType(raw), nil
}

func (s ScopeType) String() string { return string(s) }

// IsBuiltin — company / branch (dikelola platform).
func (s ScopeType) IsBuiltin() bool { return s == ScopeCompany || s == ScopeBranch }

// Scope adalah lingkup berlakunya role / permission.
type Scope struct {
	Type ScopeType
	ID   string
}
