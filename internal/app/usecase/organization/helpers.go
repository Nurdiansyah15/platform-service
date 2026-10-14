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
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	"platform-service/internal/domain/tenant/service"
)

// ensureTenant memeriksa hak akses ke tenant lalu memastikan tenant ada.
func ensureTenant(ctx context.Context, access *service.AccessPolicy, tenants tenantrepo.TenantRepository, actor *principal.Principal, tenantID string) error {
	if err := access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return mapOrgDomainError(err)
	}
	if _, err := tenants.FindByID(ctx, tenantID); err != nil {
		return mapOrgDomainError(err)
	}
	return nil
}

// accessibleCompany memuat company lalu memeriksa akses ke tenant pemiliknya.
// Company milik tenant yang tidak boleh diakses dilaporkan 404 agar keberadaannya
// tidak bocor lintas tenant.
func accessibleCompany(ctx context.Context, access *service.AccessPolicy, companies orgrepo.CompanyRepository, actor *principal.Principal, companyID string) (*entity.Company, error) {
	c, err := companies.FindByID(ctx, companyID)
	if err != nil {
		return nil, mapOrgDomainError(err)
	}
	if err := access.EnsureCanAccess(ctx, accessorOf(actor), c.TenantID); err != nil {
		if isDomainCode(err, tenantconstant.CodeTenantAccessDenied) {
			return nil, mapOrgDomainError(domainerr.New(constant.CodeCompanyNotFound))
		}
		return nil, mapOrgDomainError(err)
	}
	return c, nil
}

func mapOrgDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeCompanyNotFound, constant.CodeBranchNotFound, tenantconstant.CodeTenantNotFound:
		return apperror.NotFound(string(de.Code))
	case tenantconstant.CodeTenantNotSelected:
		return apperror.Forbidden(apperror.MsgTenantNotSelected)
	case tenantconstant.CodeTenantAccessDenied:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeCompanyCodeDuplicate, constant.CodeBranchCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeOrgCodeInvalid, constant.CodeOrgNameInvalid, constant.CodeOrgStatusInvalid:
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
	return service.Accessor{UserID: p.UserID, ActiveTenantID: p.TenantID, IsPlatformAdmin: p.IsPlatformAdmin}
}

func toCompanyResponse(c *entity.Company) dto.CompanyResponse {
	return dto.CompanyResponse{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Code: c.Code.Value(), Status: c.Status.String(),
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func toBranchResponse(b *entity.Branch) dto.BranchResponse {
	return dto.BranchResponse{
		ID: b.ID, CompanyID: b.CompanyID, Name: b.Name, Code: b.Code.Value(), Status: b.Status.String(),
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}
