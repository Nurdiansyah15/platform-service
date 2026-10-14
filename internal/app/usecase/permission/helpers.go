package permission

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapPermissionDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodePermissionNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeCatalogSyncForbidden:
		return apperror.Forbidden(string(de.Code))
	case constant.CodePermissionCodeInvalid, constant.CodePermissionNameInvalid, constant.CodePermissionCodeDuplicate,
		constant.CodeCatalogVersionInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}

// canSync — platform admin, atau token service milik solution tersebut
// (scope catalog:sync dicek di route).
func canSync(p *principal.Principal, solutionID string) bool {
	return isPlatformAdmin(p) || (p.IsService() && p.SolutionID == solutionID)
}
