package service

import (
	"context"
	"errors"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/valueobject"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgrepo "platform-service/internal/domain/organization/repository"
)

// ScopeValidator memastikan scope milik tenant (spec §2.6):
//   - tenant  → scope_id = tenant itu sendiri
//   - company → company (belum dihapus) milik tenant
//   - branch  → branch (belum dihapus) dari company milik tenant
type ScopeValidator struct {
	companies orgrepo.CompanyRepository
	branches  orgrepo.BranchRepository
}

func NewScopeValidator(companies orgrepo.CompanyRepository, branches orgrepo.BranchRepository) *ScopeValidator {
	return &ScopeValidator{companies: companies, branches: branches}
}

// Resolve memvalidasi scope dan mengembalikan CheckScope (dengan company induk untuk branch).
// scopeID kosong pada scope tenant diisi tenantID.
func (v *ScopeValidator) Resolve(ctx context.Context, tenantID string, scopeType valueobject.ScopeType, scopeID string) (entity.CheckScope, error) {
	invalid := domainerr.New(constant.CodeScopeInvalid)
	if scopeType != valueobject.ScopeTenant && scopeID == "" {
		return entity.CheckScope{}, invalid // company / branch wajib menyebut scope_id
	}
	switch scopeType {
	case valueobject.ScopeTenant:
		if scopeID != "" && scopeID != tenantID {
			return entity.CheckScope{}, invalid
		}
		return entity.CheckScope{Scope: valueobject.Scope{Type: scopeType, ID: tenantID}}, nil

	case valueobject.ScopeCompany:
		c, err := v.companies.FindByID(ctx, scopeID)
		if err != nil {
			return entity.CheckScope{}, notFoundAsInvalid(err, orgconstant.CodeCompanyNotFound)
		}
		if c.TenantID != tenantID {
			return entity.CheckScope{}, invalid
		}
		return entity.CheckScope{Scope: valueobject.Scope{Type: scopeType, ID: c.ID}}, nil

	case valueobject.ScopeBranch:
		b, err := v.branches.FindByID(ctx, scopeID)
		if err != nil {
			return entity.CheckScope{}, notFoundAsInvalid(err, orgconstant.CodeBranchNotFound)
		}
		c, err := v.companies.FindByID(ctx, b.CompanyID)
		if err != nil {
			return entity.CheckScope{}, notFoundAsInvalid(err, orgconstant.CodeCompanyNotFound)
		}
		if c.TenantID != tenantID {
			return entity.CheckScope{}, invalid
		}
		return entity.CheckScope{Scope: valueobject.Scope{Type: scopeType, ID: b.ID}, BranchCompanyID: c.ID}, nil
	}
	return entity.CheckScope{}, invalid
}

func notFoundAsInvalid(err error, notFound domainerr.Code) error {
	var de *domainerr.DomainError
	if errors.As(err, &de) && de.Code == notFound {
		return domainerr.New(constant.CodeScopeInvalid)
	}
	return err
}
