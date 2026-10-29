package organization

import (
	"context"
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	orgrepo "platform-service/internal/domain/organization/repository"
	"platform-service/internal/domain/organization/service"
)

// accessibleCompany memeriksa akses ke company lalu memastikan company ada.
func accessibleCompany(ctx context.Context, access *service.AccessPolicy, companies orgrepo.CompanyRepository, actor *principal.Principal, companyID string) (*entity.Company, error) {
	if err := access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return nil, mapCompanyDomainError(err)
	}
	c, err := companies.FindByID(ctx, companyID)
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}
	return c, nil
}

func mapCompanyDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeCompanyNotFound, constant.CodeBranchNotFound, constant.CodeMemberNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeCompanyNotSelected:
		return apperror.Forbidden(apperror.MsgCompanyNotSelected)
	case constant.CodeCompanyAccessDenied, constant.CodeCompanyStatusForbidden:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeCompanySlugDuplicate, constant.CodeMemberAlreadyExists, constant.CodeBranchCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeCompanyNameInvalid, constant.CodeCompanySlugInvalid, constant.CodeCompanyStatusInvalid,
		constant.CodeMemberStatusInvalid, constant.CodeMemberUserNotFound, constant.CodeMemberIdentityRequired,
		constant.CodeOrgCodeInvalid, constant.CodeOrgNameInvalid, constant.CodeOrgStatusInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

func accessorOf(p *principal.Principal) service.Accessor {
	a := service.Accessor{UserID: p.UserID, ActiveCompanyID: p.CompanyID, IsPlatformAdmin: p.IsUser() && p.IsPlatformAdmin}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}

func toCompanyResponse(c *entity.Company) dto.CompanyResponse {
	return dto.CompanyResponse{
		ID: c.ID, Name: c.Name, Slug: c.Slug.Value(), Status: c.Status.String(),
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toBranchResponse(b *entity.Branch) dto.BranchResponse {
	return dto.BranchResponse{
		ID: b.ID, CompanyID: b.CompanyID, Name: b.Name, Code: b.Code.Value(), Status: b.Status.String(),
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}
