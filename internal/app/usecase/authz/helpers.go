package authz

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/authz/constant"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgservice "platform-service/internal/domain/organization/service"
)

func mapAuthzDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case orgconstant.CodeCompanyNotFound:
		return apperror.NotFound(string(de.Code))
	case orgconstant.CodeCompanyNotSelected:
		return apperror.Forbidden(apperror.MsgCompanyNotSelected)
	case orgconstant.CodeCompanyAccessDenied, constant.CodeAssignmentNotSubscribed:
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

func accessorOf(p *principal.Principal) orgservice.Accessor {
	a := orgservice.Accessor{UserID: p.UserID, ActiveCompanyID: p.CompanyID, IsPlatformAdmin: p.IsUser() && p.IsPlatformAdmin}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}
