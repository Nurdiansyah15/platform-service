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

func groupDef(t *testing.T, code, name string, order int) entity.GroupDefinition {
	t.Helper()
	d, err := entity.NewGroupDefinition(code, name, nil, order)
	require.NoError(t, err)
	return d
}

func scopeDef(t *testing.T, code, parent string, order int) entity.ScopeTypeDefinition {
	t.Helper()
	d, err := entity.NewScopeTypeDefinition(code, code, nil, parent, order)
	require.NoError(t, err)
	return d
}

func TestPlanGroupSync(t *testing.T) {
	now := time.Now()
	leave := entity.NewGroup("s", groupDef(t, "leave", "Cuti", 0), now)
	old := entity.NewGroup("s", groupDef(t, "old", "Lama", 1), now)

	plan, err := service.PlanGroupSync("s", []*entity.Group{leave, old}, []entity.GroupDefinition{
		groupDef(t, "payroll", "Payroll", 0), // created
		groupDef(t, "leave", "Cuti", 1),      // updated (urutan berubah)
	}, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Created)
	assert.Equal(t, 1, plan.Updated)
	assert.Equal(t, 1, plan.Deprecated)
	assert.Equal(t, 1, leave.SortOrder)
	assert.False(t, old.IsActive())

	_, err = service.PlanGroupSync("s", nil, []entity.GroupDefinition{groupDef(t, "a", "A", 0), groupDef(t, "a", "B", 1)}, now)
	testhelper.AssertDomainError(t, err, constant.CodeGroupDuplicate)
}

func TestPlanScopeTypeSync(t *testing.T) {
	now := time.Now()
	dept := entity.NewScopeType("s", scopeDef(t, "department", "", 0), now)
	plan, err := service.PlanScopeTypeSync("s", []*entity.ScopeType{dept}, []entity.ScopeTypeDefinition{
		scopeDef(t, "department", "branch", 0),
	}, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Updated)
	assert.Equal(t, valueobject.ParentBranch, dept.Parent)
}

func TestScopeTypeDefinition_Invalid(t *testing.T) {
	_, err := entity.NewScopeTypeDefinition("branch", "Cabang", nil, "", 0)
	testhelper.AssertDomainError(t, err, constant.CodeScopeTypeReserved)
	_, err = entity.NewScopeTypeDefinition("Dept", "Dept", nil, "", 0)
	testhelper.AssertDomainError(t, err, constant.CodeScopeTypeCodeInvalid)
	_, err = entity.NewScopeTypeDefinition("dept", "Dept", nil, "region", 0)
	testhelper.AssertDomainError(t, err, constant.CodeScopeTypeParentInvalid)
}

func TestValidateCatalogRefs(t *testing.T) {
	withGroup := func(code string, group *string) entity.Definition {
		d, err := def(t, code, code).WithGroup(group)
		require.NoError(t, err)
		return d
	}
	leave, unknown := "leave", "unknown"
	groups := []entity.GroupDefinition{groupDef(t, "leave", "Cuti", 0)}

	assert.NoError(t, service.ValidateCatalogRefs([]entity.Definition{withGroup("leave.approve", &leave), withGroup("misc.read", nil)}, groups))
	err := service.ValidateCatalogRefs([]entity.Definition{withGroup("x.read", &unknown)}, groups)
	testhelper.AssertDomainError(t, err, constant.CodeGroupUnknown)
}

func TestPlanCatalogSync_GroupChange(t *testing.T) {
	now := time.Now()
	p := entity.NewPermission("s", def(t, "leave.approve", "Approve"), now)
	gid := "g1"
	d := def(t, "leave.approve", "Approve")
	d.GroupID = &gid

	plan, err := service.PlanCatalogSync("s", []*entity.Permission{p}, []entity.Definition{d}, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Updated)
	assert.Equal(t, &gid, p.GroupID)
}

func TestPlanCatalogSync_ManualUntouched(t *testing.T) {
	now := time.Now()
	manual := entity.NewManualPermission("s", def(t, "custom.export", "Export"), now)
	synced := entity.NewPermission("s", def(t, "a.read", "A"), now)

	plan, err := service.PlanCatalogSync("s", []*entity.Permission{manual, synced}, nil, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Deprecated, "hanya permission sync yang di-deprecate")
	assert.True(t, manual.IsActive())

	_, err = service.PlanCatalogSync("s", []*entity.Permission{manual}, []entity.Definition{def(t, "custom.export", "X")}, now)
	testhelper.AssertDomainError(t, err, constant.CodePermissionSourceConflict)
}

func TestResolveManualRefs(t *testing.T) {
	now := time.Now()
	g := entity.NewGroup("s", groupDef(t, "leave", "Cuti", 0), now)
	leave, unknown := "leave", "nope"

	d, err := def(t, "x.read", "X").WithGroup(&leave)
	require.NoError(t, err)
	d, err = service.ResolveManualRefs(d, []*entity.Group{g})
	require.NoError(t, err)
	require.NotNil(t, d.GroupID)
	assert.Equal(t, g.ID, *d.GroupID)

	d, _ = def(t, "x.read", "X").WithGroup(&unknown)
	_, err = service.ResolveManualRefs(d, []*entity.Group{g})
	testhelper.AssertDomainError(t, err, constant.CodeGroupUnknown)
}

func TestPlanGroupSync_ManualUntouched(t *testing.T) {
	now := time.Now()
	manual := entity.NewManualGroup("s", groupDef(t, "custom", "Custom", 5), now)
	synced := entity.NewGroup("s", groupDef(t, "leave", "Cuti", 0), now)

	plan, err := service.PlanGroupSync("s", []*entity.Group{manual, synced}, nil, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Deprecated)
	assert.True(t, manual.IsActive())

	_, err = service.PlanGroupSync("s", []*entity.Group{manual}, []entity.GroupDefinition{groupDef(t, "custom", "X", 0)}, now)
	testhelper.AssertDomainError(t, err, constant.CodeGroupSourceConflict)
}

func TestPlanScopeTypeSync_ManualUntouched(t *testing.T) {
	now := time.Now()
	manual := entity.NewManualScopeType("s", scopeDef(t, "project", "", 3), now)
	synced := entity.NewScopeType("s", scopeDef(t, "department", "", 0), now)

	plan, err := service.PlanScopeTypeSync("s", []*entity.ScopeType{manual, synced}, nil, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Deprecated)
	assert.True(t, manual.IsActive())

	_, err = service.PlanScopeTypeSync("s", []*entity.ScopeType{manual}, []entity.ScopeTypeDefinition{scopeDef(t, "project", "", 0)}, now)
	testhelper.AssertDomainError(t, err, constant.CodeScopeTypeSourceConflict)
}
