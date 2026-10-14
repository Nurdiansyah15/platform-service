// Package service berisi domain service entitlement.
package service

import (
	"bytes"
	"encoding/json"
	"math/big"
	"sort"
	"strings"

	benefitentity "platform-service/internal/domain/benefit/entity"
	benefitvo "platform-service/internal/domain/benefit/valueobject"
	"platform-service/internal/domain/entitlement/entity"
)

var unlimitedRat = big.NewRat(benefitentity.UnlimitedValue, 1)

// Calculate menghitung entitlement satu benefit dari snapshot subscription
// (spec §2.5 & keputusan Fase 12):
//   - sum: plan + Σ(add-on × quantity)
//   - max / min: nilai tertinggi / terendah (quantity tidak dikalikan)
//   - any_true: OR semua nilai
//   - override: add-on aktif terbaru, bila tidak ada → plan
//
// Benefit angka bernilai -1 = unlimited: menang di sum/max, diabaikan di min
// kecuali semua nilai unlimited.
func Calculate(b *benefitentity.Benefit, values []entity.SnapshotValue) entity.Entitlement {
	out := entity.Entitlement{Benefit: b, Value: json.RawMessage("null")}
	if len(values) == 0 {
		return out
	}
	out.Granted = true

	if b.ValueType.IsNumeric() {
		return calculateNumeric(b, values, out)
	}
	switch b.AggregationStrategy {
	case benefitvo.AggregationAnyTrue:
		result := false
		for _, v := range values {
			var x bool
			if json.Unmarshal(v.Value, &x) == nil && x {
				result = true
			}
		}
		out.Value = boolJSON(result)
	default: // override (boolean / string / json)
		out.Value = latest(values).Value
	}
	return out
}

func calculateNumeric(b *benefitentity.Benefit, values []entity.SnapshotValue, out entity.Entitlement) entity.Entitlement {
	var result *big.Rat
	unlimited := false

	switch b.AggregationStrategy {
	case benefitvo.AggregationSum:
		result = new(big.Rat)
		for _, v := range values {
			n := parseRat(v.Value)
			if n.Cmp(unlimitedRat) == 0 {
				unlimited = true
				continue
			}
			if v.FromAddon && v.Quantity > 1 {
				n.Mul(n, big.NewRat(int64(v.Quantity), 1))
			}
			result.Add(result, n)
		}
	case benefitvo.AggregationMax:
		for _, v := range values {
			n := parseRat(v.Value)
			if n.Cmp(unlimitedRat) == 0 {
				unlimited = true
				continue
			}
			if result == nil || n.Cmp(result) > 0 {
				result = n
			}
		}
	case benefitvo.AggregationMin:
		for _, v := range values {
			n := parseRat(v.Value)
			if n.Cmp(unlimitedRat) == 0 {
				continue
			}
			if result == nil || n.Cmp(result) < 0 {
				result = n
			}
		}
		unlimited = result == nil
	default: // override
		result = parseRat(latest(values).Value)
		unlimited = result.Cmp(unlimitedRat) == 0
	}

	if unlimited {
		out.Unlimited = true
		out.Value = json.RawMessage("-1")
		return out
	}
	out.Limit = result
	out.Value = json.RawMessage(FormatRat(result))
	return out
}

// latest — add-on terbaru menang; tanpa add-on → nilai plan.
func latest(values []entity.SnapshotValue) entity.SnapshotValue {
	sorted := append([]entity.SnapshotValue(nil), values...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].FromAddon != sorted[j].FromAddon {
			return !sorted[i].FromAddon // plan dulu
		}
		return sorted[i].AttachedAt.Before(sorted[j].AttachedAt)
	})
	return sorted[len(sorted)-1]
}

func parseRat(raw json.RawMessage) *big.Rat {
	r, ok := new(big.Rat).SetString(string(bytes.TrimSpace(raw)))
	if !ok {
		return new(big.Rat)
	}
	return r
}

// FormatRat menulis bilangan rasional sebagai desimal tanpa nol berlebih (maks 4 desimal).
func FormatRat(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	s := strings.TrimRight(r.FloatString(4), "0")
	return strings.TrimSuffix(s, ".")
}

func boolJSON(b bool) json.RawMessage {
	if b {
		return json.RawMessage("true")
	}
	return json.RawMessage("false")
}

// Remaining menghitung sisa kuota (limit − used, minimal 0) dan apakah limit tercapai.
func Remaining(limit, used *big.Rat) (*big.Rat, bool) {
	rem := new(big.Rat).Sub(limit, used)
	reached := rem.Sign() <= 0
	if rem.Sign() < 0 {
		rem = new(big.Rat)
	}
	return rem, reached
}
