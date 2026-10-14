package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/service"
	"platform-service/internal/domain/tenant/valueobject"
	"platform-service/internal/testhelper"
)

type fakeMembers map[string]*entity.TenantMember

func (f fakeMembers) Save(context.Context, *entity.TenantMember) error { return nil }
func (f fakeMembers) Find(_ context.Context, tenantID, userID string) (*entity.TenantMember, error) {
	if m, ok := f[tenantID+"/"+userID]; ok {
		return m, nil
	}
	return nil, domainerr.New(constant.CodeMemberNotFound)
}

type fakeSubs map[string]bool

func (f fakeSubs) HasUsable(_ context.Context, tenantID, solutionID string) (bool, error) {
	return f[tenantID+"/"+solutionID], nil
}

func TestAccessPolicy(t *testing.T) {
	members := fakeMembers{
		"t1/active":   {TenantID: "t1", UserID: "active", Status: valueobject.MemberStatusActive, JoinedAt: time.Now()},
		"t1/invited":  {TenantID: "t1", UserID: "invited", Status: valueobject.MemberStatusInvited, JoinedAt: time.Now()},
		"t1/inactive": {TenantID: "t1", UserID: "inactive", Status: valueobject.MemberStatusInactive, JoinedAt: time.Now()},
	}
	policy := service.NewAccessPolicy(members, fakeSubs{"t1/sol-a": true})

	tests := []struct {
		name string
		a    service.Accessor
		want domainerr.Code // kosong = boleh
	}{
		{"platform admin tanpa tid", service.Accessor{UserID: "x", IsPlatformAdmin: true}, ""},
		{"member active dengan tid cocok", service.Accessor{UserID: "active", ActiveTenantID: "t1"}, ""},
		{"tid kosong", service.Accessor{UserID: "active"}, constant.CodeTenantNotSelected},
		{"tid tenant lain", service.Accessor{UserID: "active", ActiveTenantID: "t2"}, constant.CodeTenantAccessDenied},
		{"member invited", service.Accessor{UserID: "invited", ActiveTenantID: "t1"}, constant.CodeTenantAccessDenied},
		{"member inactive", service.Accessor{UserID: "inactive", ActiveTenantID: "t1"}, constant.CodeTenantAccessDenied},
		{"bukan member", service.Accessor{UserID: "stranger", ActiveTenantID: "t1"}, constant.CodeTenantAccessDenied},
		{"service, tenant subscribe solution", service.Accessor{ServiceSolutionID: "sol-a"}, ""},
		{"service, tenant tidak subscribe", service.Accessor{ServiceSolutionID: "sol-b"}, constant.CodeTenantAccessDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.EnsureCanAccess(context.Background(), tt.a, "t1")
			if tt.want == "" {
				assert.NoError(t, err)
				return
			}
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}
