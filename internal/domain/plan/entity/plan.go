package entity

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/plan/constant"
	planvo "platform-service/internal/domain/plan/valueobject"
	pricingvo "platform-service/internal/domain/pricing/valueobject"
)

type Plan struct {
	ID           string
	SolutionID   string
	Code         planvo.PlanCode
	Name         string
	Description  *string
	Price        pricingvo.Price
	Currency     pricingvo.Currency
	BillingCycle pricingvo.BillingCycle
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type NewPlanParams struct {
	SolutionID   string
	Code         string
	Name         string
	Description  *string
	Price        string // kosong → 0
	Currency     string // kosong → IDR
	BillingCycle string // kosong → monthly
	IsActive     *bool  // nil → true
}

func NewPlan(p NewPlanParams, now time.Time) (*Plan, error) {
	code, err := planvo.NewPlanCode(p.Code)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodePlanNameInvalid)
	}
	price := pricingvo.ZeroPrice()
	if strings.TrimSpace(p.Price) != "" {
		if price, err = pricingvo.NewPrice(p.Price); err != nil {
			return nil, err
		}
	}
	currency, err := pricingvo.NewCurrency(p.Currency)
	if err != nil {
		return nil, err
	}
	cycle, err := pricingvo.NewBillingCycle(p.BillingCycle)
	if err != nil {
		return nil, err
	}
	active := true
	if p.IsActive != nil {
		active = *p.IsActive
	}

	now = now.UTC()
	return &Plan{
		ID:           uuid.NewString(),
		SolutionID:   p.SolutionID,
		Code:         code,
		Name:         name,
		Description:  trimOptional(p.Description),
		Price:        price,
		Currency:     currency,
		BillingCycle: cycle,
		IsActive:     active,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// PlanBenefit adalah nilai satu benefit pada plan.
type PlanBenefit struct {
	PlanID     string
	BenefitID  string
	SolutionID string
	Value      json.RawMessage
}

func trimOptional(v *string) *string {
	if v == nil {
		return nil
	}
	t := strings.TrimSpace(*v)
	if t == "" {
		return nil
	}
	return &t
}
