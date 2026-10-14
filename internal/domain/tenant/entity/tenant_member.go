package entity

import (
	"time"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/valueobject"
)

type TenantMember struct {
	TenantID string
	UserID   string
	Status   valueobject.MemberStatus
	JoinedAt time.Time
}

// NewTenantMember membuat keanggotaan baru. status kosong → active; inactive
// tidak boleh dipakai saat menambahkan member.
func NewTenantMember(tenantID, userID, status string, now time.Time) (*TenantMember, error) {
	st, err := initialStatus(status)
	if err != nil {
		return nil, err
	}
	return &TenantMember{TenantID: tenantID, UserID: userID, Status: st, JoinedAt: now.UTC()}, nil
}

// Rejoin mengaktifkan kembali member yang sebelumnya dikeluarkan (inactive).
// Member yang masih active / invited → CodeMemberAlreadyExists.
func (m *TenantMember) Rejoin(status string, now time.Time) error {
	if m.Status != valueobject.MemberStatusInactive {
		return domainerr.New(constant.CodeMemberAlreadyExists)
	}
	st, err := initialStatus(status)
	if err != nil {
		return err
	}
	m.Status = st
	m.JoinedAt = now.UTC()
	return nil
}

func (m *TenantMember) IsActive() bool {
	return m.Status == valueobject.MemberStatusActive
}

func initialStatus(raw string) (valueobject.MemberStatus, error) {
	if raw == "" {
		return valueobject.MemberStatusActive, nil
	}
	st, err := valueobject.NewMemberStatus(raw)
	if err != nil || st == valueobject.MemberStatusInactive {
		return "", domainerr.New(constant.CodeMemberStatusInvalid)
	}
	return st, nil
}
