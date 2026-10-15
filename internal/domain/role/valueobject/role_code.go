package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/role/constant"
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// RoleCode — huruf kecil, angka, "_" dan "-", diawali huruf (mis. "hr_admin").
type RoleCode struct {
	value string
}

func NewRoleCode(raw string) (RoleCode, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.CodeMaxLength || !roleCodePattern.MatchString(v) {
		return RoleCode{}, domainerr.New(constant.CodeRoleCodeInvalid)
	}
	return RoleCode{value: v}, nil
}

func (c RoleCode) Value() string { return c.value }
