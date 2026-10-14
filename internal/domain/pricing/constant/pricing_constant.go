package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodePriceInvalid        domainerr.Code = "DOMAIN_PRICING_PRICE_INVALID"
	CodeCurrencyInvalid     domainerr.Code = "DOMAIN_PRICING_CURRENCY_INVALID"
	CodeBillingCycleInvalid domainerr.Code = "DOMAIN_PRICING_BILLING_CYCLE_INVALID"
)

// DefaultCurrency sesuai spec (Default Currency: IDR).
const DefaultCurrency = "IDR"
