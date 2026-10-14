package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Slug — huruf kecil, angka, tanda hubung di tengah (mis. "pt-contoh").
type Slug struct {
	value string
}

func NewSlug(raw string) (Slug, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.SlugMaxLength || !slugPattern.MatchString(v) {
		return Slug{}, domainerr.New(constant.CodeTenantSlugInvalid)
	}
	return Slug{value: v}, nil
}

func (s Slug) Value() string { return s.value }

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusClosed    TenantStatus = "closed"
)

func NewTenantStatus(raw string) (TenantStatus, error) {
	switch s := TenantStatus(raw); s {
	case TenantStatusActive, TenantStatusSuspended, TenantStatusClosed:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeTenantStatusInvalid)
	}
}

func (s TenantStatus) String() string { return string(s) }

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
