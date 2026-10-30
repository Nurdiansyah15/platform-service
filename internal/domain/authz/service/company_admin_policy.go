package service

import (
	"context"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/repository"
	"platform-service/internal/domain/authz/valueobject"
	domainerr "platform-service/internal/domain/errors"
	orgservice "platform-service/internal/domain/organization/service"
)

// ConsoleSolution memberi ID solution console, pemilik katalog permission platform.
type ConsoleSolution interface {
	ConsoleSolutionID(ctx context.Context) (string, error)
}

// CompanyAdminPolicy menegakkan permission platform (constant.Perm*) untuk operasi tulis
// data company. Dipanggil SETELAH AccessPolicy.EnsureCanAccess:
//   - platform admin: selalu boleh
//   - token service: boleh — scope client digate route, izin bisnis ditegakkan backend solution
//   - token user: wajib punya permission di solution console pada scope yang diminta
//     (assignment level company berlaku juga untuk branch-branch-nya)
type CompanyAdminPolicy struct {
	grants  repository.GrantReader
	console ConsoleSolution
}

func NewCompanyAdminPolicy(grants repository.GrantReader, console ConsoleSolution) *CompanyAdminPolicy {
	return &CompanyAdminPolicy{grants: grants, console: console}
}

// EnsurePermission → CodeAdminPermissionDenied bila token user tidak punya permission.
func (p *CompanyAdminPolicy) EnsurePermission(ctx context.Context, a orgservice.Accessor, companyID, permission string, scope entity.CheckScope) error {
	if a.IsPlatformAdmin || a.ServiceSolutionID != "" {
		return nil
	}
	consoleID, err := p.console.ConsoleSolutionID(ctx)
	if err != nil {
		return err
	}
	grants, err := p.grants.Grants(ctx, companyID, a.UserID, consoleID)
	if err != nil {
		return err
	}
	if ok, _ := Evaluate(grants, permission, scope); !ok {
		return domainerr.New(constant.CodeAdminPermissionDenied)
	}
	return nil
}

// CompanyScope — scope cek level company.
func CompanyScope(companyID string) entity.CheckScope {
	return entity.CheckScope{Scope: valueobject.Scope{Type: valueobject.ScopeCompany, ID: companyID}}
}

// BranchScope — scope cek level branch (assignment level company induknya ikut berlaku).
func BranchScope(companyID, branchID string) entity.CheckScope {
	return entity.CheckScope{Scope: valueobject.Scope{Type: valueobject.ScopeBranch, ID: branchID}, BranchCompanyID: companyID}
}

// EnsurePlatformRoleControl mencegah eskalasi hak: role milik solution console (berisi
// permission platform) hanya boleh dibuat / diubah / di-assign / dicabut oleh pemegang
// SELURUH permission platform pada scope tersebut. Role solution lain tidak dibatasi.
func (p *CompanyAdminPolicy) EnsurePlatformRoleControl(ctx context.Context, a orgservice.Accessor, companyID, roleSolutionID string, scope entity.CheckScope) error {
	if a.IsPlatformAdmin || a.ServiceSolutionID != "" {
		return nil
	}
	consoleID, err := p.console.ConsoleSolutionID(ctx)
	if err != nil {
		return err
	}
	if roleSolutionID != consoleID {
		return nil
	}
	grants, err := p.grants.Grants(ctx, companyID, a.UserID, consoleID)
	if err != nil {
		return err
	}
	for _, perm := range constant.PlatformPermissions {
		if ok, _ := Evaluate(grants, perm.Code, scope); !ok {
			return domainerr.New(constant.CodeAdminPermissionDenied)
		}
	}
	return nil
}
