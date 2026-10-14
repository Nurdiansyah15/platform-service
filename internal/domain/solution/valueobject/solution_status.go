package valueobject

import (
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solution/constant"
)

type SolutionStatus string

const (
	SolutionStatusActive   SolutionStatus = "active"
	SolutionStatusInactive SolutionStatus = "inactive"
)

func NewSolutionStatus(raw string) (SolutionStatus, error) {
	switch s := SolutionStatus(raw); s {
	case SolutionStatusActive, SolutionStatusInactive:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeSolutionStatusInvalid)
	}
}

func (s SolutionStatus) String() string { return string(s) }
