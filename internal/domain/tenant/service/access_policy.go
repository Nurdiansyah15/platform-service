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
}

// AccessPolicy menentukan apakah user boleh mengakses data tenant tertentu.
//
// Aturan (spec §2.7 + keputusan Fase 5):
//   - platform admin: selalu boleh
//   - user lain: tenant harus menjadi tenant aktif di token (tid) DAN user member active
//
// Akses token service (subscription solution) ditambahkan di Fase 10.
type AccessPolicy struct {
	members repository.TenantMemberRepository
}

func NewAccessPolicy(members repository.TenantMemberRepository) *AccessPolicy {
	return &AccessPolicy{members: members}
}

func (p *AccessPolicy) EnsureCanAccess(ctx context.Context, a Accessor, tenantID string) error {
	if a.IsPlatformAdmin {
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
