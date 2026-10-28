package usage

import (
	"errors"
	"math/big"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	benefitconstant "platform-service/internal/domain/benefit/constant"
	benefitentity "platform-service/internal/domain/benefit/entity"
	entservice "platform-service/internal/domain/entitlement/service"
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

func toResponse(b *benefitentity.Benefit, r *entity.UsageRecord, tu *entity.CompanyUsage, limit *big.Rat, replayed bool) *dto.UsageReportResponse {
	resp := &dto.UsageReportResponse{
		BenefitCode: b.Code.Value(),
		Mode:        string(r.Mode),
		PeriodStart: tu.Period.Start,
		PeriodEnd:   tu.Period.End,
		Used:        tu.Used,
		Replayed:    replayed,
		OccurredAt:  r.OccurredAt,
	}
	if used, ok := new(big.Rat).SetString(tu.Used); ok && limit != nil {
		limitStr := entservice.FormatRat(limit)
		rem, reached := entservice.Remaining(limit, used)
		remStr := entservice.FormatRat(rem)
		resp.Limit, resp.Remaining, resp.LimitReached = &limitStr, &remStr, reached
	}
	return resp
}
