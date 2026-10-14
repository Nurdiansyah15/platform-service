package valueobject

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/usage/constant"
)

// quantityPattern — NUMERIC(18,4): maks 14 digit bulat + 4 desimal, tidak negatif.
var quantityPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,13})(\.[0-9]{1,4})?$`)

// Quantity adalah jumlah pemakaian non-negatif (string ter-normalisasi 4 desimal).
type Quantity struct {
	value string
}

func NewQuantity(raw string) (Quantity, error) {
	v := strings.TrimSpace(raw)
	if !quantityPattern.MatchString(v) {
		return Quantity{}, domainerr.New(constant.CodeUsageQuantityInvalid)
	}
	whole, frac, _ := strings.Cut(v, ".")
	frac = (frac + "0000")[:4]
	return Quantity{value: whole + "." + frac}, nil
}

// ParseQuantityJSON menerima angka JSON atau string angka.
func ParseQuantityJSON(raw json.RawMessage) (Quantity, error) {
	v := bytes.TrimSpace(raw)
	if len(v) > 0 && v[0] == '"' {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return Quantity{}, domainerr.New(constant.CodeUsageQuantityInvalid)
		}
		return NewQuantity(s)
	}
	return NewQuantity(string(v))
}

func (q Quantity) String() string { return q.value }

func (q Quantity) IsZero() bool { return strings.Trim(strings.ReplaceAll(q.value, ".", ""), "0") == "" }

// Mode laporan usage (spec §2.10): delta untuk counter, absolute untuk gauge.
type Mode string

const (
	ModeDelta    Mode = "delta"
	ModeAbsolute Mode = "absolute"
)

// IdempotencyKey — wajib, maks 150 karakter.
type IdempotencyKey struct {
	value string
}

func NewIdempotencyKey(raw string) (IdempotencyKey, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.IdempotencyKeyMaxLength {
		return IdempotencyKey{}, domainerr.New(constant.CodeUsageIdempotencyKeyInvalid)
	}
	return IdempotencyKey{value: v}, nil
}

func (k IdempotencyKey) Value() string { return k.value }
