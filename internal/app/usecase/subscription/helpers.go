package subscription

import (
	"errors"
	"time"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	orgconstant "platform-service/internal/domain/organization/constant"
	orgservice "platform-service/internal/domain/organization/service"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/entity"
	"platform-service/internal/domain/subscription/event"
)

func mapSubscriptionDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeSubscriptionNotFound, constant.CodeSubscriptionAddonNotFound, orgconstant.CodeCompanyNotFound:
		return apperror.NotFound(string(de.Code))
	case orgconstant.CodeCompanyNotSelected:
		return apperror.Forbidden(apperror.MsgCompanyNotSelected)
	case orgconstant.CodeCompanyAccessDenied:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeSubscriptionAlreadyLive, constant.CodeSubscriptionNotLive, constant.CodeSubscriptionAddonAlreadyActive:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeSubscriptionStatusInvalid, constant.CodeSubscriptionSamePlan, constant.CodeSubscriptionPlanUnavailable,
		constant.CodeSubscriptionCompanyInactive, constant.CodeSubscriptionAddonUnavailable, constant.CodeSubscriptionAddonQuantity:
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

func subscriptionEvent(eventType string, s *entity.Subscription, previousPlan, previousStatus string) port.OutboxEvent {
	return port.OutboxEvent{
		EventType:  eventType,
		CompanyID:  &s.CompanyID,
		SolutionID: &s.SolutionID,
		Payload: event.SubscriptionPayload{
			ID: s.ID, CompanyID: s.CompanyID, SolutionID: s.SolutionID, PlanID: s.PlanID, Status: s.Status.String(),
			PreviousPlanID: previousPlan, PreviousStatus: previousStatus, UpdatedAt: s.UpdatedAt,
		},
	}
}

func subscriptionAddonEvent(s *entity.Subscription, a *entity.SubscriptionAddon) port.OutboxEvent {
	return port.OutboxEvent{
		EventType:  event.TypeSubscriptionAddonAdded,
		CompanyID:  &s.CompanyID,
		SolutionID: &s.SolutionID,
		Payload: event.SubscriptionAddonPayload{
			ID: a.ID, SubscriptionID: s.ID, CompanyID: s.CompanyID, SolutionID: s.SolutionID, AddonID: a.AddonID,
			Quantity: a.Quantity, Status: string(a.Status), UpdatedAt: a.UpdatedAt,
		},
	}
}

func entitlementChanged(s *entity.Subscription, reason string, now time.Time) port.OutboxEvent {
	return port.OutboxEvent{
		EventType:  event.TypeEntitlementChanged,
		CompanyID:  &s.CompanyID,
		SolutionID: &s.SolutionID,
		Payload: event.EntitlementChangedPayload{
			CompanyID: s.CompanyID, SolutionID: s.SolutionID, SubscriptionID: s.ID, Reason: reason, OccurredAt: now.UTC(),
		},
	}
}

func toSubscriptionResponse(it port.SubscriptionReadItem) dto.SubscriptionResponse {
	return dto.SubscriptionResponse{
		ID: it.ID, CompanyID: it.CompanyID, SolutionID: it.SolutionID, SolutionCode: it.SolutionCode,
		PlanID: it.PlanID, PlanCode: it.PlanCode, PlanName: it.PlanName, Status: it.Status, BillingCycle: it.BillingCycle,
		CurrentPeriodStart: it.CurrentPeriodStart, CurrentPeriodEnd: it.CurrentPeriodEnd,
		PriceAtPurchase: it.PriceAtPurchase, Currency: it.Currency, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
	}
}

func toSubscriptionAddonResponse(it port.SubscriptionAddonReadItem) dto.SubscriptionAddonResponse {
	return dto.SubscriptionAddonResponse{
		ID: it.ID, SubscriptionID: it.SubscriptionID, AddonID: it.AddonID, AddonCode: it.AddonCode, AddonName: it.AddonName,
		Quantity: it.Quantity, UnitPriceAtPurchase: it.UnitPriceAtPurchase, Currency: it.Currency, Status: it.Status,
		CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt,
	}
}
