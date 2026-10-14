package benefit

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	domainerr "platform-service/internal/domain/errors"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapBenefitDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeBenefitNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeBenefitCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeBenefitCodeInvalid, constant.CodeBenefitNameInvalid, constant.CodeBenefitUnitInvalid,
		constant.CodeBenefitValueTypeInvalid, constant.CodeBenefitAggregationInvalid, constant.CodeBenefitMeteringInvalid,
		constant.CodeBenefitUsageResetInvalid, constant.CodeBenefitValueInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func toBenefitResponse(b *entity.Benefit) dto.BenefitResponse {
	return dto.BenefitResponse{
		ID:                  b.ID,
		SolutionID:          b.SolutionID,
		Code:                b.Code.Value(),
		Name:                b.Name,
		Description:         b.Description,
		ValueType:           string(b.ValueType),
		AggregationStrategy: string(b.AggregationStrategy),
		Unit:                b.Unit,
		Metering:            string(b.Metering),
		UsageReset:          string(b.UsageReset),
		CreatedAt:           b.CreatedAt,
		UpdatedAt:           b.UpdatedAt,
	}
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}
