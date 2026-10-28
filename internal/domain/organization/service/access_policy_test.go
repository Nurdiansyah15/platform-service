package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/service"
	"platform-service/internal/domain/organization/valueobject"
	"platform-service/internal/testhelper"
)

type fakeMembers map[string]*entity.CompanyMember

func (f fakeMembers) Save(context.Context, *entity.CompanyMember) error { return nil }
func (f fakeMembers) Find(_ context.Context, companyID, userID string) (*entity.CompanyMember, error) {
	if m, ok := f[companyID+"/"+userID]; ok {
		return m, nil
	}
	return nil, domainerr.New(constant.CodeMemberNotFound)
}

type fakeSubs map[string]bool

func (f fakeSubs) HasUsable(_ context.Context, companyID, solutionID string) (bool, error) {
	return f[companyID+"/"+solutionID], nil
}

func TestAccessPolicy(t *testing.T) {
	members := fakeMembers{
		"c1/active":   {CompanyID: "c1", UserID: "active", Status: valueobject.MemberStatusActive, JoinedAt: time.Now()},
		"c1/invited":  {CompanyID: "c1", UserID: "invited", Status: valueobject.MemberStatusInvited, JoinedAt: time.Now()},
		"c1/inactive": {CompanyID: "c1", UserID: "inactive", Status: valueobject.MemberStatusInactive, JoinedAt: time.Now()},
	}
	policy := service.NewAccessPolicy(members, fakeSubs{"c1/sol-a": true})

	tests := []struct {
		name string
		a    service.Accessor
		want domainerr.Code // kosong = boleh
	}{
		{"platform admin tanpa cid", service.Accessor{UserID: "x", IsPlatformAdmin: true}, ""},
		{"member active dengan cid cocok", service.Accessor{UserID: "active", ActiveCompanyID: "c1"}, ""},
		{"cid kosong", service.Accessor{UserID: "active"}, constant.CodeCompanyNotSelected},
		{"cid company lain", service.Accessor{UserID: "active", ActiveCompanyID: "c2"}, constant.CodeCompanyAccessDenied},
		{"member invited", service.Accessor{UserID: "invited", ActiveCompanyID: "c1"}, constant.CodeCompanyAccessDenied},
		{"member inactive", service.Accessor{UserID: "inactive", ActiveCompanyID: "c1"}, constant.CodeCompanyAccessDenied},
		{"bukan member", service.Accessor{UserID: "stranger", ActiveCompanyID: "c1"}, constant.CodeCompanyAccessDenied},
		{"service, company subscribe solution", service.Accessor{ServiceSolutionID: "sol-a"}, ""},
		{"service, company tidak subscribe", service.Accessor{ServiceSolutionID: "sol-b"}, constant.CodeCompanyAccessDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.EnsureCanAccess(context.Background(), tt.a, "c1")
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}
