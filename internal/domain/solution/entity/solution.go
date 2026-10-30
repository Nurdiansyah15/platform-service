package entity

import (
	"net"
	"net/url"
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
	// LaunchURL — halaman awal solution yang dibuka dari launcher / menu apps console.
	LaunchURL *string
	IconURL   *string
	// RedirectURIs — allowlist redirect_uri alur authorize SSO (dicocokkan persis).
	RedirectURIs []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AppLinks — tautan aplikasi solution. Field nil tidak diubah; string kosong / slice kosong
// mengosongkan nilainya.
type AppLinks struct {
	LaunchURL    *string
	IconURL      *string
	RedirectURIs *[]string
}

type NewSolutionParams struct {
	Code        string
	Name        string
	Description *string
	Status      string // kosong → active
	Links       AppLinks
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
	s := &Solution{
		ID:           uuid.NewString(),
		Code:         code,
		Name:         name,
		Description:  normalizeOptional(p.Description),
		Status:       status,
		RedirectURIs: []string{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.SetLinks(p.Links, now); err != nil {
		return nil, err
	}
	return s, nil
}

// Update mengganti nama / deskripsi / status; nil = tidak diubah. Kode tidak bisa diubah
// karena menjadi audience token (aud) solution.
func (s *Solution) Update(name, description, status *string, now time.Time) error {
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
			return domainerr.New(constant.CodeSolutionNameInvalid)
		}
		s.Name = n
	}
	if description != nil {
		s.Description = normalizeOptional(description)
	}
	if status != nil {
		st, err := valueobject.NewSolutionStatus(*status)
		if err != nil {
			return err
		}
		s.Status = st
	}
	s.UpdatedAt = now.UTC()
	return nil
}

// SetLinks memvalidasi lalu mengganti tautan aplikasi solution (lihat AppLinks).
func (s *Solution) SetLinks(l AppLinks, now time.Time) error {
	launch, err := normalizeURL(l.LaunchURL)
	if err != nil {
		return err
	}
	icon, err := normalizeURL(l.IconURL)
	if err != nil {
		return err
	}
	var redirects []string
	if l.RedirectURIs != nil {
		if len(*l.RedirectURIs) > constant.RedirectURIsMax {
			return domainerr.New(constant.CodeSolutionURLInvalid)
		}
		redirects = make([]string, 0, len(*l.RedirectURIs))
		seen := make(map[string]bool, len(*l.RedirectURIs))
		for _, raw := range *l.RedirectURIs {
			u := strings.TrimSpace(raw)
			if err := validateAppURL(u); err != nil {
				return err
			}
			if !seen[u] {
				seen[u] = true
				redirects = append(redirects, u)
			}
		}
	}

	if l.LaunchURL != nil {
		s.LaunchURL = launch
	}
	if l.IconURL != nil {
		s.IconURL = icon
	}
	if l.RedirectURIs != nil {
		s.RedirectURIs = redirects
	}
	s.UpdatedAt = now.UTC()
	return nil
}

// AllowsRedirect — redirectURI terdaftar persis di allowlist solution.
func (s *Solution) AllowsRedirect(redirectURI string) bool {
	for _, u := range s.RedirectURIs {
		if u == redirectURI {
			return true
		}
	}
	return false
}

// Origins — origin (scheme://host[:port], huruf kecil, port default dibuang) dari LaunchURL dan
// RedirectURIs, unik. Dipakai allowlist CORS: FE solution di origin ini boleh memanggil API.
func (s *Solution) Origins() []string {
	urls := s.RedirectURIs
	if s.LaunchURL != nil {
		urls = append([]string{*s.LaunchURL}, urls...)
	}
	out := make([]string, 0, len(urls))
	seen := make(map[string]bool, len(urls))
	for _, raw := range urls {
		o, ok := OriginOf(raw)
		if ok && !seen[o] {
			seen[o] = true
			out = append(out, o)
		}
	}
	return out
}

// OriginOf menormalkan URL / header Origin menjadi scheme://host[:port] huruf kecil tanpa port
// default (80 untuk http, 443 untuk https). ok=false bila bukan URL http(s) absolut.
func OriginOf(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return scheme + "://" + host, true
}

// IsLaunchable — solution bisa dibuka dari launcher / menu apps.
func (s *Solution) IsLaunchable() bool {
	return s.IsActive() && s.LaunchURL != nil
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

// normalizeURL — nil tetap nil (tidak diubah); string kosong → pointer ke nil (dikosongkan).
func normalizeURL(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil, nil
	}
	if err := validateAppURL(t); err != nil {
		return nil, err
	}
	return &t, nil
}

// validateAppURL — URL absolut https tanpa fragment; http hanya untuk localhost / 127.0.0.1
// (pengembangan lokal).
func validateAppURL(raw string) error {
	invalid := domainerr.New(constant.CodeSolutionURLInvalid)
	if raw == "" || len(raw) > constant.URLMaxLength {
		return invalid
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Fragment != "" || u.User != nil {
		return invalid
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if h := u.Hostname(); h == "localhost" || h == "127.0.0.1" {
			return nil
		}
	}
	return invalid
}
