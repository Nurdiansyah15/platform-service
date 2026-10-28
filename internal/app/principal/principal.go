// Package principal mendeskripsikan pemanggil API yang sudah terautentikasi:
// user (token user, spec §2.8) atau service (token client credentials).
package principal

import "slices"

type Kind string

const (
	KindUser    Kind = "user"
	KindService Kind = "service"
)

type Principal struct {
	Kind Kind

	// Token user (typ=user)
	UserID          string
	SessionID       string
	CompanyID       string // company aktif; kosong sebelum pilih company
	SolutionCode    string // aud
	IsPlatformAdmin bool

	// Token service (typ=service)
	ClientID   string
	SolutionID string
	Scopes     []string
}

func (p *Principal) IsUser() bool {
	return p != nil && p.Kind == KindUser
}

func (p *Principal) IsService() bool {
	return p != nil && p.Kind == KindService
}

func (p *Principal) HasScope(scope string) bool {
	return p.IsService() && slices.Contains(p.Scopes, scope)
}

func (p *Principal) HasActiveCompany() bool {
	return p.IsUser() && p.CompanyID != ""
}
