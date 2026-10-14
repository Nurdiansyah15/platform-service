package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/valueobject"
	"platform-service/internal/testhelper"
)

func TestNewTenant(t *testing.T) {
	tn, err := entity.NewTenant(" PT Contoh ", " PT-Contoh ", time.Now())
	require.NoError(t, err)
	assert.Equal(t, "PT Contoh", tn.Name)
	assert.Equal(t, "pt-contoh", tn.Slug.Value())
	assert.True(t, tn.IsActive())

	_, err = entity.NewTenant("", "x", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeTenantNameInvalid)
	_, err = entity.NewTenant("X", "pt contoh", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeTenantSlugInvalid)
}

func TestNewTenantMember(t *testing.T) {
	m, err := entity.NewTenantMember("t", "u", "", time.Now())
	require.NoError(t, err)
	assert.True(t, m.IsActive())

	m, err = entity.NewTenantMember("t", "u", "invited", time.Now())
	require.NoError(t, err)
	assert.Equal(t, valueobject.MemberStatusInvited, m.Status)

	_, err = entity.NewTenantMember("t", "u", "inactive", time.Now())
	testhelper.AssertDomainError(t, err, constant.CodeMemberStatusInvalid)
}

func TestTenantMember_Rejoin(t *testing.T) {
	m, err := entity.NewTenantMember("t", "u", "", time.Now())
	require.NoError(t, err)
	testhelper.AssertDomainError(t, m.Rejoin("", time.Now()), constant.CodeMemberAlreadyExists)

	m.Status = valueobject.MemberStatusInactive
	require.NoError(t, m.Rejoin("invited", time.Now()))
	assert.Equal(t, valueobject.MemberStatusInvited, m.Status)
}
