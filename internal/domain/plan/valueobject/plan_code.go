package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/plan/constant"
)

var planCodePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// PlanCode — huruf kecil, angka, tanda hubung (mis. "pro", "enterprise-2026").
// Dipakai juga di payload license (spec §2.14: "plan": "pro").
type PlanCode struct {
	value string
}

func NewPlanCode(raw string) (PlanCode, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.CodeMaxLength || !planCodePattern.MatchString(v) {
		return PlanCode{}, domainerr.New(constant.CodePlanCodeInvalid)
	}
	return PlanCode{value: v}, nil
}

func (c PlanCode) Value() string { return c.value }
