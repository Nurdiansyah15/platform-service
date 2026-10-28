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

// ScopeValidator memastikan scope milik company (spec §2.6):
//   - company → scope_id = company itu sendiri (boleh kosong, diisi companyID)
//   - branch  → branch (belum dihapus) dari company tersebut
type ScopeValidator struct {
	branches orgrepo.BranchRepository
}

func NewScopeValidator(branches orgrepo.BranchRepository) *ScopeValidator {
	return &ScopeValidator{branches: branches}
}

// Resolve memvalidasi scope dan mengembalikan CheckScope (dengan company induk untuk branch).
// scopeID kosong pada scope company diisi companyID.
func (v *ScopeValidator) Resolve(ctx context.Context, companyID string, scopeType valueobject.ScopeType, scopeID string) (entity.CheckScope, error) {
	invalid := domainerr.New(constant.CodeScopeInvalid)
	switch scopeType {
	case valueobject.ScopeCompany:
		if scopeID != "" && scopeID != companyID {
			return entity.CheckScope{}, invalid
		}
		return entity.CheckScope{Scope: valueobject.Scope{Type: scopeType, ID: companyID}}, nil

	case valueobject.ScopeBranch:
		if scopeID == "" {
			return entity.CheckScope{}, invalid // branch wajib menyebut scope_id
		}
		b, err := v.branches.FindByID(ctx, scopeID)
		if err != nil {
			return entity.CheckScope{}, notFoundAsInvalid(err, orgconstant.CodeBranchNotFound)
		}
		if b.CompanyID != companyID {
			return entity.CheckScope{}, invalid
		}
		return entity.CheckScope{Scope: valueobject.Scope{Type: scopeType, ID: b.ID}, BranchCompanyID: companyID}, nil
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
