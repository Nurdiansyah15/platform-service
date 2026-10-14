// Package entity berisi model entitlement (hak efektif tenant atas benefit).
package entity

import (
	"encoding/json"
	"math/big"
	"time"

	benefitentity "platform-service/internal/domain/benefit/entity"
)

// SnapshotValue adalah satu nilai benefit yang dibekukan untuk subscription.
type SnapshotValue struct {
	BenefitID string
	Value     json.RawMessage
	// FromAddon — false = berasal dari plan.
	FromAddon bool
	// Quantity add-on (1 untuk nilai plan).
	Quantity int
	// AttachedAt — waktu add-on (terakhir) dipasang; dipakai strategi override.
	AttachedAt time.Time
}

// Entitlement adalah hak efektif tenant atas satu benefit.
type Entitlement struct {
	Benefit *benefitentity.Benefit
	// Granted — benefit diberikan plan / add-on aktif.
	Granted bool
	// Value — nilai efektif (JSON); null bila tidak granted.
	Value json.RawMessage
	// Unlimited — benefit angka bernilai -1.
	Unlimited bool
	// Limit — nilai efektif angka (nil bila bukan angka, tidak granted, atau unlimited).
	Limit *big.Rat
}
