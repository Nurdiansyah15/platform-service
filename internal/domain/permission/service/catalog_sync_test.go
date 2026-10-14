package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/permission/service"
	"platform-service/internal/domain/permission/valueobject"
	"platform-service/internal/testhelper"
)

func def(t *testing.T, code, name string) entity.Definition {
	t.Helper()
	d, err := entity.NewDefinition(code, name, nil)
	require.NoError(t, err)
	return d
}

func TestPlanCatalogSync(t *testing.T) {
	now := time.Now()
	read := entity.NewPermission("s", def(t, "employee.read", "Read employee"), now)
	write := entity.NewPermission("s", def(t, "employee.write", "Write employee"), now)
	legacy := entity.NewPermission("s", def(t, "legacy.export", "Legacy"), now)
	legacy.Status = valueobject.StatusDeprecated
	old := entity.NewPermission("s", def(t, "report.old", "Old"), now)
	old.Status = valueobject.StatusDeprecated

	plan, err := service.PlanCatalogSync("s", []*entity.Permission{read, write, legacy, old}, []entity.Definition{
		def(t, "employee.read", "Read employee"),     // unchanged
		def(t, "employee.write", "Write employees"),  // updated
		def(t, "legacy.export", "Legacy export"),     // reactivated
		def(t, "payroll:approve", "Approve payroll"), // created
	}, now)
	require.NoError(t, err)

	assert.Equal(t, 1, plan.Created)
	assert.Equal(t, 1, plan.Updated)
	assert.Equal(t, 1, plan.Reactivated)
	assert.Equal(t, 0, plan.Deprecated)
	assert.Equal(t, 2, plan.Unchanged, "employee.read + report.old (sudah deprecated)")
	assert.Len(t, plan.Changed, 3)
	assert.True(t, legacy.IsActive())
	assert.Equal(t, "Write employees", write.Name)

	// Katalog kosong → semua yang aktif di-deprecate.
	plan, err = service.PlanCatalogSync("s", []*entity.Permission{read, write}, nil, now)
	require.NoError(t, err)
	assert.Equal(t, 2, plan.Deprecated)
	assert.False(t, read.IsActive())
}

func TestPlanCatalogSync_Duplicate(t *testing.T) {
	_, err := service.PlanCatalogSync("s", nil, []entity.Definition{def(t, "a.read", "A"), def(t, "a.read", "A lagi")}, time.Now())
	testhelper.AssertDomainError(t, err, constant.CodePermissionCodeDuplicate)
}

func TestNewDefinition_Invalid(t *testing.T) {
	for _, code := range []string{"", "Employee.Read", "employee read", ".read", "employee..read", "1st.perm"} {
		_, err := entity.NewDefinition(code, "X", nil)
		testhelper.AssertDomainError(t, err, constant.CodePermissionCodeInvalid)
	}
	_, err := entity.NewDefinition("a.read", " ", nil)
	testhelper.AssertDomainError(t, err, constant.CodePermissionNameInvalid)
}
