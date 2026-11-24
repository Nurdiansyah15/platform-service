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

// NewPermissionStatus — active | deprecated.
func NewPermissionStatus(raw string) (PermissionStatus, error) {
	switch s := PermissionStatus(raw); s {
	case StatusActive, StatusDeprecated:
		return s, nil
	default:
		return "", domainerr.New(constant.CodePermissionStatusInvalid)
	}
}

// PermissionSource — asal permission: sync (katalog dari backend solution) atau manual
// (dibuat platform admin). Sync hanya mengelola permission ber-source sync.
type PermissionSource string

const (
	SourceSync   PermissionSource = "sync"
	SourceManual PermissionSource = "manual"
)

func (s PermissionSource) String() string { return string(s) }

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

// groupCodePattern — code permission group (mis. "leave", "payroll-report").
var groupCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type GroupCode struct {
	value string
}

func NewGroupCode(raw string) (GroupCode, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.GroupCodeMaxLength || !groupCodePattern.MatchString(v) {
		return GroupCode{}, domainerr.New(constant.CodeGroupCodeInvalid)
	}
	return GroupCode{value: v}, nil
}

func (c GroupCode) Value() string { return c.value }

// scopeTypeCodePattern — code scope type custom (mis. "department", "work_unit").
var scopeTypeCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ScopeTypeCode — code scope type custom milik solution. "company" dan "branch" reserved
// (built-in, selalu tersedia).
type ScopeTypeCode struct {
	value string
}

func NewScopeTypeCode(raw string) (ScopeTypeCode, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.ScopeTypeCodeMaxLength || !scopeTypeCodePattern.MatchString(v) {
		return ScopeTypeCode{}, domainerr.New(constant.CodeScopeTypeCodeInvalid)
	}
	if IsBuiltinScopeType(v) {
		return ScopeTypeCode{}, domainerr.New(constant.CodeScopeTypeReserved)
	}
	return ScopeTypeCode{value: v}, nil
}

func (c ScopeTypeCode) Value() string { return c.value }

// IsBuiltinScopeType — company / branch.
func IsBuiltinScopeType(code string) bool {
	return code == constant.ScopeTypeCompany || code == constant.ScopeTypeBranch
}

// ScopeParent — scope induk scope type custom: company (default) atau branch.
type ScopeParent string

const (
	ParentCompany ScopeParent = constant.ScopeTypeCompany
	ParentBranch  ScopeParent = constant.ScopeTypeBranch
)

// NewScopeParent — kosong → company.
func NewScopeParent(raw string) (ScopeParent, error) {
	switch v := ScopeParent(strings.TrimSpace(raw)); v {
	case "":
		return ParentCompany, nil
	case ParentCompany, ParentBranch:
		return v, nil
	default:
		return "", domainerr.New(constant.CodeScopeTypeParentInvalid)
	}
}

func (p ScopeParent) String() string { return string(p) }
