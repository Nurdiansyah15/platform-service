// Package valueobject berisi VO harga yang dipakai plan, add-on, dan subscription.
package valueobject

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/pricing/constant"
)

// pricePattern — NUMERIC(18,2): maks 16 digit bulat + 2 desimal, tidak negatif.
var pricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,2})?$`)

// Price adalah nominal uang non-negatif dengan 2 desimal, disimpan sebagai string
// ter-normalisasi (mis. "150000.00") agar tidak ada pembulatan floating point.
type Price struct {
	value string
}

func NewPrice(raw string) (Price, error) {
	v := strings.TrimSpace(raw)
	if !pricePattern.MatchString(v) {
		return Price{}, domainerr.New(constant.CodePriceInvalid)
	}
	whole, frac, _ := strings.Cut(v, ".")
	frac = (frac + "00")[:2]
	return Price{value: whole + "." + frac}, nil
}

func ZeroPrice() Price { return Price{value: "0.00"} }

func (p Price) String() string { return p.value }

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Currency — kode ISO 4217 tiga huruf (dinormalisasi UPPERCASE). Kosong → IDR.
type Currency struct {
	value string
}

func NewCurrency(raw string) (Currency, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if v == "" {
		v = constant.DefaultCurrency
	}
	if !currencyPattern.MatchString(v) {
		return Currency{}, domainerr.New(constant.CodeCurrencyInvalid)
	}
	return Currency{value: v}, nil
}

func (c Currency) String() string { return c.value }

type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// NewBillingCycle — kosong → monthly.
func NewBillingCycle(raw string) (BillingCycle, error) {
	if raw == "" {
		return BillingCycleMonthly, nil
	}
	switch c := BillingCycle(raw); c {
	case BillingCycleMonthly, BillingCycleYearly:
		return c, nil
	default:
		return "", domainerr.New(constant.CodeBillingCycleInvalid)
	}
}

func (c BillingCycle) String() string { return string(c) }

// ParsePriceJSON mengubah harga dari JSON (angka atau string angka) menjadi teks
// untuk NewPrice. Kosong / null → "" (dianggap harga 0 oleh pemanggil).
func ParsePriceJSON(raw json.RawMessage) (string, error) {
	v := bytes.TrimSpace(raw)
	if len(v) == 0 || bytes.Equal(v, []byte("null")) {
		return "", nil
	}
	if v[0] == '"' {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return "", domainerr.New(constant.CodePriceInvalid)
		}
		return s, nil
	}
	return string(v), nil
}
