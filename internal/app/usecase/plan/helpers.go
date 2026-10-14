package plan

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/plan/constant"
	"platform-service/internal/domain/plan/entity"
	pricingconstant "platform-service/internal/domain/pricing/constant"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapPlanDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodePlanNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodePlanCodeDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodePlanCodeInvalid, constant.CodePlanNameInvalid,
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

func toPlanResponse(p *entity.Plan) dto.PlanResponse {
	return dto.PlanResponse{
		ID: p.ID, SolutionID: p.SolutionID, Code: p.Code.Value(), Name: p.Name, Description: p.Description,
		Price: p.Price.String(), Currency: p.Currency.String(), BillingCycle: p.BillingCycle.String(),
		IsActive: p.IsActive, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// toAssignedBenefitResponses memetakan read item benefit plan ke DTO.
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
