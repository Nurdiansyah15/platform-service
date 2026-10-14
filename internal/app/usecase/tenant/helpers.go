package tenant

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/service"
)

func mapTenantDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeTenantNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeTenantNotSelected:
		return apperror.Forbidden(apperror.MsgTenantNotSelected)
	case constant.CodeTenantAccessDenied:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeTenantSlugDuplicate, constant.CodeMemberAlreadyExists:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeTenantNameInvalid, constant.CodeTenantSlugInvalid, constant.CodeTenantStatusInvalid,
		constant.CodeMemberStatusInvalid, constant.CodeMemberUserNotFound, constant.CodeMemberIdentityRequired:
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
	a := service.Accessor{UserID: p.UserID, ActiveTenantID: p.TenantID, IsPlatformAdmin: p.IsUser() && p.IsPlatformAdmin}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}

func toTenantResponse(t *entity.Tenant) dto.TenantResponse {
	return dto.TenantResponse{
		ID: t.ID, Name: t.Name, Slug: t.Slug.Value(), Status: t.Status.String(), CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}
