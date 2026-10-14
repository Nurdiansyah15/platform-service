package valueobject

import (
	"regexp"
	"strings"

	"platform-service/internal/domain/addon/constant"
	domainerr "platform-service/internal/domain/errors"
)

var addonCodePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// AddonCode — huruf kecil, angka, tanda hubung (mis. "payroll"). Dipakai juga di
// payload license (spec §2.14: "addons": [{"code": "payroll"}]).
type AddonCode struct {
	value string
}

func NewAddonCode(raw string) (AddonCode, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" || len(v) > constant.CodeMaxLength || !addonCodePattern.MatchString(v) {
		return AddonCode{}, domainerr.New(constant.CodeAddonCodeInvalid)
	}
	return AddonCode{value: v}, nil
}

func (c AddonCode) Value() string { return c.value }
