package usage

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	benefitentity "platform-service/internal/domain/benefit/entity"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/usage/constant"
	"platform-service/internal/domain/usage/entity"
)

func mapUsageDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case benefitconstant.CodeBenefitNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeUsageSolutionMismatch, constant.CodeUsageNoSubscription:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeUsageIdempotencyConflict:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeUsageQuantityInvalid, constant.CodeUsageIdempotencyKeyInvalid, constant.CodeUsageNotMetered, constant.CodeUsageWrongMode:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

func toResponse(b *benefitentity.Benefit, r *entity.UsageRecord, tu *entity.TenantUsage, replayed bool) *dto.UsageReportResponse {
	return &dto.UsageReportResponse{
		BenefitCode: b.Code.Value(),
		Mode:        string(r.Mode),
		PeriodStart: tu.Period.Start,
		PeriodEnd:   tu.Period.End,
		Used:        tu.Used,
		Replayed:    replayed,
		OccurredAt:  r.OccurredAt,
	}
}
