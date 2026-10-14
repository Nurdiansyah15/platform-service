package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
)

// permissionCodePattern — segmen huruf kecil/angka/_/- dipisah "." atau ":" (mis. "employee.read",
// "payroll:approve"), diawali huruf.
var permissionCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*([.:][a-z0-9_-]+)*$`)

type PermissionCode struct {
	value string
}

func NewPermissionCode(raw string) (PermissionCode, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.CodeMaxLength || !permissionCodePattern.MatchString(v) {
		return PermissionCode{}, domainerr.New(constant.CodePermissionCodeInvalid)
	}
	return PermissionCode{value: v}, nil
}

func (c PermissionCode) Value() string { return c.value }

type PermissionStatus string

const (
	StatusActive     PermissionStatus = "active"
	StatusDeprecated PermissionStatus = "deprecated"
)

func (s PermissionStatus) String() string { return string(s) }

// CatalogVersion — versi katalog yang dikirim solution saat deploy (maks 50 karakter).
type CatalogVersion struct {
	value string
}

func NewCatalogVersion(raw string) (CatalogVersion, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.CatalogVersionMaxLength {
		return CatalogVersion{}, domainerr.New(constant.CodeCatalogVersionInvalid)
	}
	return CatalogVersion{value: v}, nil
}

func (v CatalogVersion) Value() string { return v.value }
