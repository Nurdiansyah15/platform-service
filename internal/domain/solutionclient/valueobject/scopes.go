package valueobject

import (
	"slices"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solutionclient/constant"
)

// Scopes adalah himpunan scope service yang valid, terurut dan tanpa duplikat.
type Scopes struct {
	values []string
}

func NewScopes(raw []string) (Scopes, error) {
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if !slices.Contains(constant.AllScopes, s) {
			return Scopes{}, domainerr.New(constant.CodeClientScopeInvalid)
		}
		out = append(out, s)
	}
	slices.Sort(out)
	return Scopes{values: slices.Compact(out)}, nil
}

// ParseScopeParam mem-parse parameter OAuth2 "scope" (dipisah spasi).
func ParseScopeParam(param string) (Scopes, error) {
	return NewScopes(strings.Fields(param))
}

func (s Scopes) Values() []string { return slices.Clone(s.values) }

func (s Scopes) Contains(scope string) bool { return slices.Contains(s.values, scope) }

func (s Scopes) IsEmpty() bool { return len(s.values) == 0 }

// IsSubsetOf — semua scope s dimiliki other.
func (s Scopes) IsSubsetOf(other Scopes) bool {
	for _, v := range s.values {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// Intersect mengembalikan scope yang ada di kedua himpunan.
func (s Scopes) Intersect(other Scopes) Scopes {
	out := make([]string, 0, len(s.values))
	for _, v := range s.values {
		if other.Contains(v) {
			out = append(out, v)
		}
	}
	return Scopes{values: out}
}
