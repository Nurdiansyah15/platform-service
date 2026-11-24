package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"platform-service/internal/domain/authz/constant"
	"platform-service/internal/domain/authz/entity"
	"platform-service/internal/domain/authz/valueobject"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgrepo "platform-service/internal/domain/organization/repository"
	permconstant "platform-service/internal/domain/permission/constant"
	permrepo "platform-service/internal/domain/permission/repository"
	permvo "platform-service/internal/domain/permission/valueobject"
)

// ScopeValidator memastikan scope milik company (spec §2.6):
//   - company → scope_id = company itu sendiri (boleh kosong, diisi companyID)
//   - branch  → branch (belum dihapus) dari company tersebut
//   - custom  → scope type terdaftar di katalog solution; scope_id string bebas milik solution.
//     Bila scope type ber-parent branch dan parent_id diisi, parent_id wajib branch company itu.
type ScopeValidator struct {
	branches   orgrepo.BranchRepository
	scopeTypes permrepo.ScopeTypeRepository
}

func NewScopeValidator(branches orgrepo.BranchRepository, scopeTypes permrepo.ScopeTypeRepository) *ScopeValidator {
	return &ScopeValidator{branches: branches, scopeTypes: scopeTypes}
}

// ScopeInput — scope yang akan divalidasi.
type ScopeInput struct {
	SolutionID string
	CompanyID  string
	Type       valueobject.ScopeType
	ID         string
	ParentID   string
	// ForAssignment — scope type custom yang deprecated ditolak (assignment baru); untuk
	// check tetap diterima agar assignment lama tetap bisa dievaluasi.
	ForAssignment bool
}

// Resolve memvalidasi scope dan mengembalikan CheckScope.
func (v *ScopeValidator) Resolve(ctx context.Context, in ScopeInput) (entity.CheckScope, error) {
	invalid := domainerr.New(constant.CodeScopeInvalid)
	base := entity.CheckScope{CompanyID: in.CompanyID}
	switch in.Type {
	case valueobject.ScopeCompany:
		if (in.ID != "" && in.ID != in.CompanyID) || in.ParentID != "" {
			return entity.CheckScope{}, invalid
		}
		base.Scope = valueobject.Scope{Type: in.Type, ID: in.CompanyID}
		return base, nil

	case valueobject.ScopeBranch:
		if in.ParentID != "" {
			return entity.CheckScope{}, invalid
		}
		id, err := v.ensureBranch(ctx, in.CompanyID, in.ID)
		if err != nil {
			return entity.CheckScope{}, err
		}
		base.Scope = valueobject.Scope{Type: in.Type, ID: id}
		return base, nil
	}

	st, err := v.scopeTypes.FindByCode(ctx, in.SolutionID, in.Type.String())
	if err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == permconstant.CodeScopeTypeNotFound {
			return entity.CheckScope{}, domainerr.New(constant.CodeScopeTypeInvalid)
		}
		return entity.CheckScope{}, err
	}
	if in.ForAssignment && !st.IsActive() {
		return entity.CheckScope{}, domainerr.New(constant.CodeScopeTypeInvalid)
	}
	if in.ID == "" || len(in.ID) > constant.ScopeIDMaxLength {
		return entity.CheckScope{}, invalid // scope custom wajib menyebut scope_id
	}
	base.Scope = valueobject.Scope{Type: in.Type, ID: in.ID}
	if in.ParentID != "" {
		if st.Parent != permvo.ParentBranch {
			return entity.CheckScope{}, invalid
		}
		if base.ParentBranchID, err = v.ensureBranch(ctx, in.CompanyID, in.ParentID); err != nil {
			return entity.CheckScope{}, err
		}
	}
	return base, nil
}

func (v *ScopeValidator) ensureBranch(ctx context.Context, companyID, branchID string) (string, error) {
	invalid := domainerr.New(constant.CodeScopeInvalid)
	if _, err := uuid.Parse(branchID); err != nil {
		return "", invalid
	}
	b, err := v.branches.FindByID(ctx, branchID)
	if err != nil {
		return "", notFoundAsInvalid(err, orgconstant.CodeBranchNotFound)
	}
	if b.CompanyID != companyID {
		return "", invalid
	}
	return b.ID, nil
}

func notFoundAsInvalid(err error, notFound domainerr.Code) error {
	var de *domainerr.DomainError
	if errors.As(err, &de) && de.Code == notFound {
		return domainerr.New(constant.CodeScopeInvalid)
	}
	return err
}
