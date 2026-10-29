package entity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/valueobject"
)

// Company adalah entitas tertinggi (menggantikan tenant).
type Company struct {
	ID        string
	Name      string
	Slug      valueobject.Slug
	Status    valueobject.CompanyStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Branch struct {
	ID        string
	CompanyID string
	Name      string
	Code      valueobject.OrgCode
	Status    valueobject.OrgStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type OrgUnitParams struct {
	Name   string
	Code   string
	Status string
}

func NewCompany(name, slug string, now time.Time) (*Company, error) {
	n := strings.TrimSpace(name)
	if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeCompanyNameInvalid)
	}
	s, err := valueobject.NewSlug(slug)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	return &Company{
		ID: uuid.NewString(), Name: n, Slug: s, Status: valueobject.CompanyStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (c *Company) IsActive() bool {
	return c.Status == valueobject.CompanyStatusActive
}

// Update mengganti nama dan/atau slug company; nil = tidak diubah.
func (c *Company) Update(name, slug *string, now time.Time) error {
	if name != nil {
		n := strings.TrimSpace(*name)
		if n == "" || utf8.RuneCountInString(n) > constant.NameMaxLength {
			return domainerr.New(constant.CodeCompanyNameInvalid)
		}
		c.Name = n
	}
	if slug != nil {
		s, err := valueobject.NewSlug(*slug)
		if err != nil {
			return err
		}
		c.Slug = s
	}
	c.UpdatedAt = now.UTC()
	return nil
}

// ChangeStatus mengganti status company (active / suspended / closed).
// Mengembalikan true bila status benar-benar berubah.
func (c *Company) ChangeStatus(raw string, now time.Time) (bool, error) {
	st, err := valueobject.NewCompanyStatus(raw)
	if err != nil {
		return false, err
	}
	if st == c.Status {
		return false, nil
	}
	c.Status = st
	c.UpdatedAt = now.UTC()
	return true, nil
}

func NewBranch(companyID string, p OrgUnitParams, now time.Time) (*Branch, error) {
	name, code, status, err := validateOrgUnit(p)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	return &Branch{
		ID: uuid.NewString(), CompanyID: companyID, Name: name, Code: code, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func validateOrgUnit(p OrgUnitParams) (string, valueobject.OrgCode, valueobject.OrgStatus, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return "", valueobject.OrgCode{}, "", domainerr.New(constant.CodeOrgNameInvalid)
	}
	code, err := valueobject.NewOrgCode(p.Code)
	if err != nil {
		return "", valueobject.OrgCode{}, "", err
	}
	status, err := valueobject.NewOrgStatus(p.Status)
	if err != nil {
		return "", valueobject.OrgCode{}, "", err
	}
	return name, code, status, nil
}
