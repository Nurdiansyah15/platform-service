package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/valueobject"
	"platform-service/internal/testhelper"
)

func TestNewCompany(t *testing.T) {
	c, err := entity.NewCompany(" PT Contoh ", " PT-Contoh ", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "PT Contoh", c.Name)
	assert.Equal(t, "pt-contoh", c.Slug.Value())
	assert.True(t, c.IsActive())

	_, err = entity.NewCompany("", "x", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeCompanyNameInvalid)
	_, err = entity.NewCompany("X", "pt contoh", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeCompanySlugInvalid)
}

func TestNewCompanyMember(t *testing.T) {
	m, err := entity.NewCompanyMember("c", "u", "", time.Now())
	require.NoError(t, err)
	assert.True(t, m.IsActive())

	m, err = entity.NewCompanyMember("c", "u", "invited", time.Now())
	require.NoError(t, err)
	assert.Equal(t, valueobject.MemberStatusInvited, m.Status)

	_, err = entity.NewCompanyMember("c", "u", "inactive", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeMemberStatusInvalid)
}

func TestCompanyMember_Rejoin(t *testing.T) {
	m, err := entity.NewCompanyMember("c", "u", "", time.Now())
	require.NoError(t, err)
	testhelper.AssertDomainError(t, m.Rejoin("", time.Now()), constant.CodeMemberAlreadyExists)

	m.Status = valueobject.MemberStatusInactive
	require.NoError(t, m.Rejoin("invited", time.Now()))
	assert.Equal(t, valueobject.MemberStatusInvited, m.Status)
}

func TestNewBranch_Invalid(t *testing.T) {
	tests := []struct {
		name string
		p    entity.OrgUnitParams
		want domainerr.Code
	}{
		{"name kosong", entity.OrgUnitParams{Name: " ", Code: "B1"}, constant.CodeOrgNameInvalid},
		{"code kosong", entity.OrgUnitParams{Name: "X", Code: ""}, constant.CodeOrgCodeInvalid},
		{"code spasi", entity.OrgUnitParams{Name: "X", Code: "B 1"}, constant.CodeOrgCodeInvalid},
		{"code diawali tanda", entity.OrgUnitParams{Name: "X", Code: "-B1"}, constant.CodeOrgCodeInvalid},
		{"status tak dikenal", entity.OrgUnitParams{Name: "X", Code: "B1", Status: "closed"}, constant.CodeOrgStatusInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.NewBranch("c1", tt.p, time.Now())
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}
