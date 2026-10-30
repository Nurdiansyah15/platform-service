package role

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	authzconstant "platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgservice "platform-service/internal/domain/organization/service"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/domain/role/event"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapRoleDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeRoleNotFound, solutionconstant.CodeSolutionNotFound, orgconstant.CodeCompanyNotFound:
		return apperror.NotFound(string(de.Code))
	case orgconstant.CodeCompanyNotSelected:
		return apperror.Forbidden(apperror.MsgCompanyNotSelected)
	case orgconstant.CodeCompanyAccessDenied, constant.CodeRoleSystemImmutable, constant.CodeRoleCompanyNotSubscribed,
		authzconstant.CodeAdminPermissionDenied:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeRoleCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeRoleCodeInvalid, constant.CodeRoleNameInvalid, constant.CodeRolePermissionUnknown,
		constant.CodeRolePermissionDuplicate, constant.CodeRolePermissionDeprecated:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}

func accessorOf(p *principal.Principal) orgservice.Accessor {
	a := orgservice.Accessor{UserID: p.UserID, ActiveCompanyID: p.CompanyID, IsPlatformAdmin: isPlatformAdmin(p)}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}

func roleScope(companyID *string, isSystem bool) string {
	switch {
	case isSystem:
		return "system"
	case companyID == nil:
		return "global"
	default:
		return "company"
	}
}

func toRoleResponse(r *entity.Role) dto.RoleResponse {
	return dto.RoleResponse{
		ID: r.ID, SolutionID: r.SolutionID, CompanyID: r.CompanyID, Code: r.Code.Value(), Name: r.Name,
		Description: r.Description, IsSystem: r.IsSystem, Scope: roleScope(r.CompanyID, r.IsSystem),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func roleEvent(eventType string, r *entity.Role, permissionCodes []string) port.OutboxEvent {
	return port.OutboxEvent{
		EventType:  eventType,
		CompanyID:  r.CompanyID,
		SolutionID: &r.SolutionID,
		Payload: event.RolePayload{
			ID: r.ID, SolutionID: r.SolutionID, CompanyID: r.CompanyID, Code: r.Code.Value(), Name: r.Name,
			IsSystem: r.IsSystem, PermissionCodes: permissionCodes, UpdatedAt: r.UpdatedAt,
		},
	}
}
