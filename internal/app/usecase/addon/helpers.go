package addon

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/addon/constant"
	"platform-service/internal/domain/addon/entity"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	domainerr "platform-service/internal/domain/errors"
	pricingconstant "platform-service/internal/domain/pricing/constant"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapAddonDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeAddonNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeAddonCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeAddonCodeInvalid, constant.CodeAddonNameInvalid,
		pricingconstant.CodePriceInvalid, pricingconstant.CodeCurrencyInvalid, pricingconstant.CodeBillingCycleInvalid,
		benefitconstant.CodeBenefitAssignmentDuplicate, benefitconstant.CodeBenefitAssignmentUnknown,
		benefitconstant.CodeBenefitSolutionMismatch, benefitconstant.CodeBenefitValueInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}

func toAddonResponse(a *entity.Addon) dto.AddonResponse {
	return dto.AddonResponse{
		ID: a.ID, SolutionID: a.SolutionID, Code: a.Code.Value(), Name: a.Name, Description: a.Description,
		Price: a.Price.String(), Currency: a.Currency.String(), BillingCycle: a.BillingCycle.String(),
		IsActive: a.IsActive, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func toAssignedBenefitResponses(items []port.AssignedBenefitReadItem) []dto.AssignedBenefitResponse {
	out := make([]dto.AssignedBenefitResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.AssignedBenefitResponse{
			BenefitID: it.BenefitID, Code: it.Code, Name: it.Name, ValueType: it.ValueType,
			AggregationStrategy: it.AggregationStrategy, Unit: it.Unit, Metering: it.Metering, Value: it.Value,
		})
	}
	return out
}
