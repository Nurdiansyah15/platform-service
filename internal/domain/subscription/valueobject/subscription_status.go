package valueobject

import (
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/subscription/constant"
)

type SubscriptionStatus string

const (
	StatusTrialing SubscriptionStatus = "trialing"
	StatusActive   SubscriptionStatus = "active"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusExpired  SubscriptionStatus = "expired"
)

func NewSubscriptionStatus(raw string) (SubscriptionStatus, error) {
	switch s := SubscriptionStatus(raw); s {
	case StatusTrialing, StatusActive, StatusPastDue, StatusCanceled, StatusExpired:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeSubscriptionStatusInvalid)
	}
}

func (s SubscriptionStatus) String() string { return string(s) }

// IsLive — subscription masih berjalan (belum dibatalkan / kedaluwarsa). Maksimal
// satu subscription live per tenant per solution.
func (s SubscriptionStatus) IsLive() bool {
	return s == StatusTrialing || s == StatusActive || s == StatusPastDue
}

// IsUsable — subscription memberi akses (pilih tenant, entitlement, akses service).
func (s SubscriptionStatus) IsUsable() bool {
	return s == StatusTrialing || s == StatusActive
}

// UsableStatuses dipakai query yang memfilter subscription "aktif" (spec §2.6 / §2.7).
var UsableStatuses = []string{string(StatusTrialing), string(StatusActive)}
