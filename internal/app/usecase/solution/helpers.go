package solution

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solution/entity"
)

func mapSolutionDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeSolutionNotFound:
		return errSolutionNotFound()
	case constant.CodeSolutionCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeSolutionCodeInvalid, constant.CodeSolutionNameInvalid, constant.CodeSolutionStatusInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func errSolutionNotFound() error {
	return apperror.NotFound(string(constant.CodeSolutionNotFound))
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}

func toSolutionResponse(s *entity.Solution) dto.SolutionResponse {
	return dto.SolutionResponse{
		ID:                       s.ID,
		Code:                     s.Code.Value(),
		Name:                     s.Name,
		Description:              s.Description,
		Status:                   s.Status.String(),
		PermissionCatalogVersion: s.PermissionCatalogVersion,
		CreatedAt:                s.CreatedAt,
		UpdatedAt:                s.UpdatedAt,
	}
}
