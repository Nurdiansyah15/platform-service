package authz

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantservice "platform-service/internal/domain/tenant/service"
)

func mapAuthzDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case tenantconstant.CodeTenantNotFound:
		return apperror.NotFound(string(de.Code))
	case tenantconstant.CodeTenantNotSelected:
		return apperror.Forbidden(apperror.MsgTenantNotSelected)
	case tenantconstant.CodeTenantAccessDenied, constant.CodeAssignmentNotSubscribed:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeAssignmentDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeAssignmentNotMember, constant.CodeAssignmentRoleUnavailable, constant.CodeScopeTypeInvalid,
		constant.CodeScopeInvalid, constant.CodeCheckTargetRequired, constant.CodeCheckSolutionUnavailable:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

func accessorOf(p *principal.Principal) tenantservice.Accessor {
	a := tenantservice.Accessor{UserID: p.UserID, ActiveTenantID: p.TenantID, IsPlatformAdmin: p.IsUser() && p.IsPlatformAdmin}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}
