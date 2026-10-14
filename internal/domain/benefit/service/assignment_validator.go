// Package service berisi domain service benefit.
package service

import (
	"context"
	"encoding/json"

	"platform-service/internal/domain/benefit/constant"
	"platform-service/internal/domain/benefit/entity"
	"platform-service/internal/domain/benefit/repository"
	domainerr "platform-service/internal/domain/errors"
)

// Assignment adalah satu nilai benefit yang akan dipasang ke plan / add-on.
type Assignment struct {
	BenefitID string
	Value     json.RawMessage
}

// ValidatedAssignment adalah Assignment yang lolos validasi beserta benefit-nya.
type ValidatedAssignment struct {
	Benefit *entity.Benefit
	Value   json.RawMessage
}

// AssignmentValidator memvalidasi sekumpulan nilai benefit untuk satu solution:
//   - setiap benefit_id unik dalam satu permintaan
//   - benefit ada dan milik solution yang sama (composite FK di DB adalah pengaman terakhir)
//   - nilai cocok dengan value_type benefit
//
// Dipakai plan_benefits (Fase 8) dan addon_benefits (Fase 9).
type AssignmentValidator struct {
	benefits repository.BenefitRepository
}

func NewAssignmentValidator(benefits repository.BenefitRepository) *AssignmentValidator {
	return &AssignmentValidator{benefits: benefits}
}

func (v *AssignmentValidator) Validate(ctx context.Context, solutionID string, items []Assignment) ([]ValidatedAssignment, error) {
	if len(items) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		if seen[it.BenefitID] {
			return nil, domainerr.New(constant.CodeBenefitAssignmentDuplicate)
		}
		seen[it.BenefitID] = true
		ids = append(ids, it.BenefitID)
	}

	found, err := v.benefits.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*entity.Benefit, len(found))
	for _, b := range found {
		byID[b.ID] = b
	}

	out := make([]ValidatedAssignment, 0, len(items))
	for _, it := range items {
		b, ok := byID[it.BenefitID]
		if !ok {
			return nil, domainerr.New(constant.CodeBenefitAssignmentUnknown)
		}
		if b.SolutionID != solutionID {
			return nil, domainerr.New(constant.CodeBenefitSolutionMismatch)
		}
		if err := b.ValidateValue(it.Value); err != nil {
			return nil, err
		}
		out = append(out, ValidatedAssignment{Benefit: b, Value: it.Value})
	}
	return out, nil
}
