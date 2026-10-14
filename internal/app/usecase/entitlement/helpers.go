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
	solutionconstant "platform-service/internal/domain/solution/constant"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantservice "platform-service/internal/domain/tenant/service"
)

func mapEntitlementDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case benefitconstant.CodeBenefitNotFound, solutionconstant.CodeSolutionNotFound, tenantconstant.CodeTenantNotFound:
		return apperror.NotFound(string(de.Code))
	case tenantconstant.CodeTenantNotSelected:
		return apperror.Forbidden(apperror.MsgTenantNotSelected)
	case tenantconstant.CodeTenantAccessDenied:
		return apperror.Forbidden(string(de.Code))
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func accessorOf(p *principal.Principal) tenantservice.Accessor {
	a := tenantservice.Accessor{UserID: p.UserID, ActiveTenantID: p.TenantID, IsPlatformAdmin: p.IsUser() && p.IsPlatformAdmin}
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
