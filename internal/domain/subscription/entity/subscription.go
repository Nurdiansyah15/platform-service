package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
	"platform-service/internal/domain/subscription/constant"
	"platform-service/internal/domain/subscription/valueobject"
)

// PlanTerms adalah data plan yang dibekukan ke subscription saat pembelian / ganti plan.
type PlanTerms struct {
	PlanID       string
	SolutionID   string
	BillingCycle pricingvo.BillingCycle
	Price        pricingvo.Price
	Currency     pricingvo.Currency
}

type Subscription struct {
	ID                 string
	TenantID           string
	SolutionID         string
	PlanID             string
	Status             valueobject.SubscriptionStatus
	BillingCycle       pricingvo.BillingCycle
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	PriceAtPurchase    pricingvo.Price
	Currency           pricingvo.Currency
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewSubscription membuat subscription baru. status kosong → active; hanya
// trialing / active yang boleh dipakai saat membuat.
func NewSubscription(tenantID string, plan PlanTerms, status string, periodStart, now time.Time) (*Subscription, error) {
	st := valueobject.StatusActive
	if status != "" {
		var err error
		if st, err = valueobject.NewSubscriptionStatus(status); err != nil {
			return nil, err
		}
		if !st.IsUsable() {
			return nil, domainerr.New(constant.CodeSubscriptionStatusInvalid)
		}
	}
	now = now.UTC()
	s := &Subscription{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Status:    st,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.applyPlan(plan, periodStart.UTC())
	return s, nil
}

// ChangePlan mengganti plan subscription live; periode baru dimulai sekarang.
func (s *Subscription) ChangePlan(plan PlanTerms, now time.Time) error {
	if !s.Status.IsLive() {
		return domainerr.New(constant.CodeSubscriptionNotLive)
	}
	if plan.PlanID == s.PlanID {
		return domainerr.New(constant.CodeSubscriptionSamePlan)
	}
	if plan.SolutionID != s.SolutionID {
		return domainerr.New(constant.CodeSubscriptionPlanUnavailable)
	}
	s.applyPlan(plan, now.UTC())
	s.UpdatedAt = now.UTC()
	return nil
}

// Cancel menghentikan subscription. Snapshot tetap disimpan sebagai histori.
func (s *Subscription) Cancel(now time.Time) error {
	if !s.Status.IsLive() {
		return domainerr.New(constant.CodeSubscriptionNotLive)
	}
	s.Status = valueobject.StatusCanceled
	s.UpdatedAt = now.UTC()
	return nil
}

func (s *Subscription) applyPlan(plan PlanTerms, periodStart time.Time) {
	s.PlanID = plan.PlanID
	s.SolutionID = plan.SolutionID
	s.BillingCycle = plan.BillingCycle
	s.PriceAtPurchase = plan.Price
	s.Currency = plan.Currency
	end := periodEnd(periodStart, plan.BillingCycle)
	s.CurrentPeriodStart = &periodStart
	s.CurrentPeriodEnd = &end
}

func periodEnd(start time.Time, cycle pricingvo.BillingCycle) time.Time {
	if cycle == pricingvo.BillingCycleYearly {
		return addMonthsClamped(start, 12)
	}
	return addMonthsClamped(start, 1)
}

// addMonthsClamped menambah bulan dan menjepit tanggal ke akhir bulan tujuan
// (31 Jan + 1 bulan = 28/29 Feb), berbeda dengan time.AddDate yang melimpah ke
// bulan berikutnya (31 Jan + 1 bulan = 3 Mar) dan membuat periode tagihan bergeser.
func addMonthsClamped(t time.Time, months int) time.Time {
	firstOfTarget := time.Date(t.Year(), t.Month()+time.Month(months), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	lastDay := firstOfTarget.AddDate(0, 1, -1).Day()
	day := t.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// ── Subscription add-on ──────────────────────────────────────────────────────

type SubscriptionAddonStatus string

const (
	AddonStatusActive   SubscriptionAddonStatus = "active"
	AddonStatusCanceled SubscriptionAddonStatus = "canceled"
)

// AddonTerms adalah data add-on yang dibekukan saat dipasang.
type AddonTerms struct {
	AddonID    string
	SolutionID string
	Price      pricingvo.Price
	Currency   pricingvo.Currency
}

type SubscriptionAddon struct {
	ID                  string
	SubscriptionID      string
	AddonID             string
	SolutionID          string
	Quantity            int
	UnitPriceAtPurchase pricingvo.Price
	Currency            pricingvo.Currency
	Status              SubscriptionAddonStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// AttachAddon memasang add-on ke subscription live.
func (s *Subscription) AttachAddon(addon AddonTerms, quantity int, now time.Time) (*SubscriptionAddon, error) {
	if !s.Status.IsLive() {
		return nil, domainerr.New(constant.CodeSubscriptionNotLive)
	}
	if addon.SolutionID != s.SolutionID {
		return nil, domainerr.New(constant.CodeSubscriptionAddonUnavailable)
	}
	if quantity < 1 {
		return nil, domainerr.New(constant.CodeSubscriptionAddonQuantity)
	}
	now = now.UTC()
	return &SubscriptionAddon{
		ID:                  uuid.NewString(),
		SubscriptionID:      s.ID,
		AddonID:             addon.AddonID,
		SolutionID:          s.SolutionID,
		Quantity:            quantity,
		UnitPriceAtPurchase: addon.Price,
		Currency:            addon.Currency,
		Status:              AddonStatusActive,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// Reactivate memasang ulang add-on yang pernah dibatalkan dengan harga & quantity baru.
func (a *SubscriptionAddon) Reactivate(addon AddonTerms, quantity int, now time.Time) error {
	if a.Status == AddonStatusActive {
		return domainerr.New(constant.CodeSubscriptionAddonAlreadyActive)
	}
	if quantity < 1 {
		return domainerr.New(constant.CodeSubscriptionAddonQuantity)
	}
	a.Quantity = quantity
	a.UnitPriceAtPurchase = addon.Price
	a.Currency = addon.Currency
	a.Status = AddonStatusActive
	a.UpdatedAt = now.UTC()
	return nil
}

// ── Snapshot ─────────────────────────────────────────────────────────────────

// BenefitSnapshot adalah nilai benefit yang dibekukan untuk subscription.
// SubscriptionAddonID nil = berasal dari plan.
type BenefitSnapshot struct {
	ID                  string
	SubscriptionID      string
	SubscriptionAddonID *string
	BenefitID           string
	Value               json.RawMessage
	CreatedAt           time.Time
}
