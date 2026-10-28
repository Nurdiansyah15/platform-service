package entity

import (
	"time"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/valueobject"
)

type CompanyMember struct {
	CompanyID string
	UserID    string
	Status    valueobject.MemberStatus
	JoinedAt  time.Time
}

// NewCompanyMember membuat keanggotaan baru. status kosong → active; inactive
// tidak boleh dipakai saat menambahkan member.
func NewCompanyMember(companyID, userID, status string, now time.Time) (*CompanyMember, error) {
	st, err := initialStatus(status)
	if err != nil {
		return nil, err
	}
	return &CompanyMember{CompanyID: companyID, UserID: userID, Status: st, JoinedAt: now.UTC()}, nil
}

// Rejoin mengaktifkan kembali member yang sebelumnya dikeluarkan (inactive).
// Member yang masih active / invited → CodeMemberAlreadyExists.
func (m *CompanyMember) Rejoin(status string, now time.Time) error {
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

func (m *CompanyMember) IsActive() bool {
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
