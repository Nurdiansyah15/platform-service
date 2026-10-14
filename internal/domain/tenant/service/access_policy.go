// Package service berisi domain service tenant.
package service

import (
	"context"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/repository"
)

// Accessor menggambarkan pemanggil yang ingin mengakses data satu tenant.
type Accessor struct {
	UserID          string
	ActiveTenantID  string // claim tid token
	IsPlatformAdmin bool
	// ServiceSolutionID terisi bila pemanggil adalah token service (claim sol).
	ServiceSolutionID string
}

// SubscriptionChecker memeriksa apakah tenant berlangganan solution (trialing / active).
type SubscriptionChecker interface {
	HasUsable(ctx context.Context, tenantID, solutionID string) (bool, error)
}

// AccessPolicy menentukan apakah user boleh mengakses data tenant tertentu.
//
// Aturan (spec §2.7, §2.8 + keputusan Fase 5 & 10):
//   - platform admin: selalu boleh
//   - token service: tenant harus berlangganan (trialing / active) solution milik client
//   - user lain: tenant harus menjadi tenant aktif di token (tid) DAN user member active
type AccessPolicy struct {
	members       repository.TenantMemberRepository
	subscriptions SubscriptionChecker
}

func NewAccessPolicy(members repository.TenantMemberRepository, subscriptions SubscriptionChecker) *AccessPolicy {
	return &AccessPolicy{members: members, subscriptions: subscriptions}
}

func (p *AccessPolicy) EnsureCanAccess(ctx context.Context, a Accessor, tenantID string) error {
	if a.IsPlatformAdmin {
		return nil
	}
	if a.ServiceSolutionID != "" {
		ok, err := p.subscriptions.HasUsable(ctx, tenantID, a.ServiceSolutionID)
		if err != nil {
			return err
		}
		if !ok {
			return domainerr.New(constant.CodeTenantAccessDenied)
		}
		return nil
	}
	if a.ActiveTenantID == "" {
		return domainerr.New(constant.CodeTenantNotSelected)
	}
	if a.ActiveTenantID != tenantID {
		return domainerr.New(constant.CodeTenantAccessDenied)
	}
	m, err := p.members.Find(ctx, tenantID, a.UserID)
	if err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == constant.CodeMemberNotFound {
			return domainerr.New(constant.CodeTenantAccessDenied)
		}
		return err
	}
	if !m.IsActive() {
		return domainerr.New(constant.CodeTenantAccessDenied)
	}
	return nil
}
