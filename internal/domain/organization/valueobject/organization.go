package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
)

var orgCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]*$`)

// OrgCode — code branch, dinormalisasi UPPERCASE (mis. "HQ-01").
type OrgCode struct {
	value string
}

func NewOrgCode(raw string) (OrgCode, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.CodeMaxLength || !orgCodePattern.MatchString(v) {
		return OrgCode{}, domainerr.New(constant.CodeOrgCodeInvalid)
	}
	return OrgCode{value: v}, nil
}

func (c OrgCode) Value() string { return c.value }

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Slug — huruf kecil, angka, tanda hubung di tengah (mis. "pt-contoh").
type Slug struct {
	value string
}

func NewSlug(raw string) (Slug, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.SlugMaxLength || !slugPattern.MatchString(v) {
		return Slug{}, domainerr.New(constant.CodeCompanySlugInvalid)
	}
	return Slug{value: v}, nil
}

func (s Slug) Value() string { return s.value }

type OrgStatus string

const (
	OrgStatusActive   OrgStatus = "active"
	OrgStatusInactive OrgStatus = "inactive"
)

// NewOrgStatus — kosong → active.
func NewOrgStatus(raw string) (OrgStatus, error) {
	if raw == "" {
		return OrgStatusActive, nil
	}
	switch s := OrgStatus(raw); s {
	case OrgStatusActive, OrgStatusInactive:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeOrgStatusInvalid)
	}
}

func (s OrgStatus) String() string { return string(s) }

type CompanyStatus string

const (
	CompanyStatusActive    CompanyStatus = "active"
	CompanyStatusSuspended CompanyStatus = "suspended"
	CompanyStatusClosed    CompanyStatus = "closed"
)

func NewCompanyStatus(raw string) (CompanyStatus, error) {
	switch s := CompanyStatus(raw); s {
	case CompanyStatusActive, CompanyStatusSuspended, CompanyStatusClosed:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeCompanyStatusInvalid)
	}
}

func (s CompanyStatus) String() string { return string(s) }

type MemberStatus string

const (
	MemberStatusInvited  MemberStatus = "invited"
	MemberStatusActive   MemberStatus = "active"
	MemberStatusInactive MemberStatus = "inactive"
)

func NewMemberStatus(raw string) (MemberStatus, error) {
	switch s := MemberStatus(raw); s {
	case MemberStatusInvited, MemberStatusActive, MemberStatusInactive:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeMemberStatusInvalid)
	}
}

func (s MemberStatus) String() string { return string(s) }
