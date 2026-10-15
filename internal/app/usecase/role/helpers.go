package role

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/role/constant"
	"platform-service/internal/domain/role/entity"
	"platform-service/internal/domain/role/event"
	solutionconstant "platform-service/internal/domain/solution/constant"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantservice "platform-service/internal/domain/tenant/service"
)

func mapRoleDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeRoleNotFound, solutionconstant.CodeSolutionNotFound, tenantconstant.CodeTenantNotFound:
		return apperror.NotFound(string(de.Code))
	case tenantconstant.CodeTenantNotSelected:
		return apperror.Forbidden(apperror.MsgTenantNotSelected)
	case tenantconstant.CodeTenantAccessDenied, constant.CodeRoleSystemImmutable, constant.CodeRoleTenantNotSubscribed:
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

func accessorOf(p *principal.Principal) tenantservice.Accessor {
	a := tenantservice.Accessor{UserID: p.UserID, ActiveTenantID: p.TenantID, IsPlatformAdmin: isPlatformAdmin(p)}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}

func roleScope(tenantID *string, isSystem bool) string {
	switch {
	case isSystem:
		return "system"
	case tenantID == nil:
		return "global"
	default:
		return "tenant"
	}
}

func toRoleResponse(r *entity.Role) dto.RoleResponse {
	return dto.RoleResponse{
		ID: r.ID, SolutionID: r.SolutionID, TenantID: r.TenantID, Code: r.Code.Value(), Name: r.Name,
		Description: r.Description, IsSystem: r.IsSystem, Scope: roleScope(r.TenantID, r.IsSystem),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func roleEvent(eventType string, r *entity.Role, permissionCodes []string) port.OutboxEvent {
	return port.OutboxEvent{
		EventType:  eventType,
		TenantID:   r.TenantID,
		SolutionID: &r.SolutionID,
		Payload: event.RolePayload{
			ID: r.ID, SolutionID: r.SolutionID, TenantID: r.TenantID, Code: r.Code.Value(), Name: r.Name,
			IsSystem: r.IsSystem, PermissionCodes: permissionCodes, UpdatedAt: r.UpdatedAt,
		},
	}
}
