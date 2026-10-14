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

type Company struct {
	ID        string
	TenantID  string
	Name      string
	Code      valueobject.OrgCode
	Status    valueobject.OrgStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
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

func NewCompany(tenantID string, p OrgUnitParams, now time.Time) (*Company, error) {
	name, code, status, err := validateOrgUnit(p)
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	return &Company{
		ID: uuid.NewString(), TenantID: tenantID, Name: name, Code: code, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}, nil
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
