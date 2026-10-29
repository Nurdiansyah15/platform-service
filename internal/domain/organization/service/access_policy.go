// Package service berisi domain service organisasi.
package service

import (
	"context"
	"errors"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/repository"
)

// Accessor menggambarkan pemanggil yang ingin mengakses data satu company.
type Accessor struct {
	UserID          string
	ActiveCompanyID string // claim cid token
	IsPlatformAdmin bool
	// ServiceSolutionID terisi bila pemanggil adalah token service (claim sol).
	ServiceSolutionID string
}

// SubscriptionChecker memeriksa apakah company berlangganan solution (trialing / active).
type SubscriptionChecker interface {
	HasUsable(ctx context.Context, companyID, solutionID string) (bool, error)
}

// AccessPolicy menentukan apakah user boleh mengakses data company tertentu.
//
// Aturan (spec §2.7, §2.8 + keputusan Fase 5 & 10):
//   - platform admin: selalu boleh
//   - token service: company harus berlangganan (trialing / active) solution milik client
//     — GATE-SUBSCRIPTION (nonaktif sementara): saat ini semua company boleh diakses
//   - user lain: company harus menjadi company aktif di token (cid) DAN user member active
type AccessPolicy struct {
	members       repository.CompanyMemberRepository
	subscriptions SubscriptionChecker
}

func NewAccessPolicy(members repository.CompanyMemberRepository, subscriptions SubscriptionChecker) *AccessPolicy {
	return &AccessPolicy{members: members, subscriptions: subscriptions}
}

func (p *AccessPolicy) EnsureCanAccess(ctx context.Context, a Accessor, companyID string) error {
	if a.IsPlatformAdmin {
		return nil
	}
	if a.ServiceSolutionID != "" {
		// GATE-SUBSCRIPTION (nonaktif sementara): plan/entitlement belum dipakai sebagai
		// batasan, jadi token service boleh mengakses company mana pun.
		// ok, err := p.subscriptions.HasUsable(ctx, companyID, a.ServiceSolutionID)
		// if err != nil {
		// 	return err
		// }
		// if !ok {
		// 	return domainerr.New(constant.CodeCompanyAccessDenied)
		// }
		return nil
	}
	if a.ActiveCompanyID == "" {
		return domainerr.New(constant.CodeCompanyNotSelected)
	}
	if a.ActiveCompanyID != companyID {
		return domainerr.New(constant.CodeCompanyAccessDenied)
	}
	m, err := p.members.Find(ctx, companyID, a.UserID)
	if err != nil {
		var de *domainerr.DomainError
		if errors.As(err, &de) && de.Code == constant.CodeMemberNotFound {
			return domainerr.New(constant.CodeCompanyAccessDenied)
		}
		return err
	}
	if !m.IsActive() {
		return domainerr.New(constant.CodeCompanyAccessDenied)
	}
	return nil
}
