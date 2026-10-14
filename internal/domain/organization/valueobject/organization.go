package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
)

var orgCodePattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]*$`)

// OrgCode — code company / branch, dinormalisasi UPPERCASE (mis. "HQ-01").
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
