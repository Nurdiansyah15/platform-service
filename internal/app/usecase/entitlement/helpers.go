package entitlement

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	ententity "platform-service/internal/domain/entitlement/entity"
	entservice "platform-service/internal/domain/entitlement/service"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgservice "platform-service/internal/domain/organization/service"
	solutionconstant "platform-service/internal/domain/solution/constant"
)

func mapEntitlementDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case benefitconstant.CodeBenefitNotFound, solutionconstant.CodeSolutionNotFound, orgconstant.CodeCompanyNotFound:
		return apperror.NotFound(string(de.Code))
	case orgconstant.CodeCompanyNotSelected:
		return apperror.Forbidden(apperror.MsgCompanyNotSelected)
	case orgconstant.CodeCompanyAccessDenied:
		return apperror.Forbidden(string(de.Code))
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func accessorOf(p *principal.Principal) orgservice.Accessor {
	a := orgservice.Accessor{UserID: p.UserID, ActiveCompanyID: p.CompanyID, IsPlatformAdmin: p.IsUser() && p.IsPlatformAdmin}
	if p.IsService() {
		a.ServiceSolutionID = p.SolutionID
	}
	return a
}

func toEntitlementResponse(e ententity.Entitlement) dto.EntitlementResponse {
	b := e.Benefit
	out := dto.EntitlementResponse{
		BenefitCode:         b.Code.Value(),
		Name:                b.Name,
		ValueType:           string(b.ValueType),
		AggregationStrategy: string(b.AggregationStrategy),
		Unit:                b.Unit,
		Metering:            string(b.Metering),
		Granted:             e.Granted,
		Value:               e.Value,
		Unlimited:           e.Unlimited,
	}
	if e.Limit != nil && b.IsMetered() {
		limit := entservice.FormatRat(e.Limit)
		out.Limit = &limit
	}
	return out
}

func isPlatformAdmin(p *principal.Principal) bool {
	return p.IsUser() && p.IsPlatformAdmin
}
