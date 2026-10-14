package valueobject

import (
	"regexp"
	"strings"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
)

// solutionCodePattern: huruf kecil, angka, dan tanda hubung di tengah (mis. "easy-hr").
// Code dipakai sebagai claim aud token user, jadi harus stabil & aman di URL.
var solutionCodePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type SolutionCode struct {
	value string
}

func NewSolutionCode(raw string) (SolutionCode, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len(v) > constant.CodeMaxLength || !solutionCodePattern.MatchString(v) {
		return SolutionCode{}, domainerr.New(constant.CodeSolutionCodeInvalid)
	}
	return SolutionCode{value: v}, nil
}

func (c SolutionCode) Value() string { return c.value }
