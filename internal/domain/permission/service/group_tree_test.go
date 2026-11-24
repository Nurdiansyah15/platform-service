package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/permission/constant"
	"platform-service/internal/domain/permission/entity"
	"platform-service/internal/domain/permission/service"
	"platform-service/internal/domain/permission/valueobject"
	"platform-service/internal/testhelper"
)

func childDef(t *testing.T, code, parent string, order int) entity.GroupDefinition {
	t.Helper()
	d := groupDef(t, code, code, order)
	var p *string
	if parent != "" {
		p = &parent
	}
	d, err := d.WithParent(p)
	require.NoError(t, err)
	return d
}

func byCode(groups []*entity.Group) map[string]*entity.Group {
	m := map[string]*entity.Group{}
	for _, g := range groups {
		m[g.Code.Value()] = g
	}
	return m
}

func TestGroupDefinition_WithParent_Self(t *testing.T) {
	self := "hr"
	_, err := groupDef(t, "hr", "HR", 0).WithParent(&self)
	testhelper.AssertDomainError(t, err, constant.CodeGroupParentInvalid)
}

func TestPlanGroupSync_Hierarchy(t *testing.T) {
	now := time.Now()
	// Anak ditulis sebelum induknya di katalog — tetap ter-resolve.
	plan, err := service.PlanGroupSync("s", nil, []entity.GroupDefinition{
		childDef(t, "employee", "hr", 0),
		childDef(t, "hr", "", 1),
		childDef(t, "contract", "employee", 2),
	}, now)
	require.NoError(t, err)
	require.Len(t, plan.Changed, 3)
	g := byCode(plan.Changed)
	assert.Nil(t, g["hr"].ParentID)
	assert.Equal(t, g["hr"].ID, *g["employee"].ParentID)
	assert.Equal(t, g["employee"].ID, *g["contract"].ParentID)

	path := service.GroupPath(plan.Changed, g["contract"].ID)
	require.Len(t, path, 3)
	assert.Equal(t, []string{"hr", "employee", "contract"},
		[]string{path[0].Code.Value(), path[1].Code.Value(), path[2].Code.Value()})

	// Sync ulang: hanya induk yang berubah → dihitung updated, bukan unchanged.
	existing := plan.Changed
	plan, err = service.PlanGroupSync("s", existing, []entity.GroupDefinition{
		childDef(t, "employee", "hr", 0),
		childDef(t, "hr", "", 1),
		childDef(t, "contract", "hr", 2),
	}, now)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Updated)
	assert.Equal(t, 2, plan.Unchanged)
	assert.Equal(t, g["hr"].ID, *g["contract"].ParentID)
}

func TestPlanGroupSync_HierarchyInvalid(t *testing.T) {
	now := time.Now()
	tests := map[string]struct {
		catalog []entity.GroupDefinition
		code    domainerr.Code
	}{
		"induk tidak di katalog": {[]entity.GroupDefinition{childDef(t, "a", "x", 0)}, constant.CodeGroupParentUnknown},
		"siklus": {[]entity.GroupDefinition{childDef(t, "a", "b", 0), childDef(t, "b", "a", 1)},
			constant.CodeGroupParentInvalid},
		"4 tingkat": {[]entity.GroupDefinition{childDef(t, "a", "", 0), childDef(t, "b", "a", 1), childDef(t, "c", "b", 2),
			childDef(t, "d", "c", 3)}, constant.CodeGroupTooDeep},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := service.PlanGroupSync("s", nil, tc.catalog, now)
			testhelper.AssertDomainError(t, err, tc.code)
		})
	}
}

// Group manual yang menempel di group sync ikut dihitung kedalamannya saat sync.
func TestPlanGroupSync_ManualChildTooDeep(t *testing.T) {
	now := time.Now()
	plan, err := service.PlanGroupSync("s", nil, []entity.GroupDefinition{childDef(t, "a", "", 0), childDef(t, "b", "a", 1)}, now)
	require.NoError(t, err)
	existing := plan.Changed
	manual := entity.NewManualGroup("s", groupDef(t, "m", "M", 0), now)
	_, err = service.PlaceGroup(existing, manual, "b", now) // a > b > m = 3 tingkat
	require.NoError(t, err)
	existing = append(existing, manual)

	_, err = service.PlanGroupSync("s", existing, []entity.GroupDefinition{
		childDef(t, "root", "", 0), childDef(t, "a", "root", 1), childDef(t, "b", "a", 2),
	}, now)
	testhelper.AssertDomainError(t, err, constant.CodeGroupTooDeep)
}

func TestPlaceGroup(t *testing.T) {
	now := time.Now()
	mk := func(code string) *entity.Group { return entity.NewManualGroup("s", groupDef(t, code, code, 0), now) }
	a, b, c, old := mk("a"), mk("b"), mk("c"), mk("old")
	old.Status = valueobject.StatusDeprecated
	all := []*entity.Group{a, b, c, old}

	changed, err := service.PlaceGroup(all, b, "a", now)
	require.NoError(t, err)
	assert.True(t, changed)
	_, err = service.PlaceGroup(all, c, "b", now)
	require.NoError(t, err)

	changed, err = service.PlaceGroup(all, b, "a", now)
	require.NoError(t, err)
	assert.False(t, changed, "induk sama → tidak berubah")

	tests := map[string]struct {
		g      *entity.Group
		parent string
		code   domainerr.Code
	}{
		"induk tidak ada":    {a, "x", constant.CodeGroupParentUnknown},
		"induk diri sendiri": {a, "a", constant.CodeGroupParentInvalid},
		"induk = turunan":    {a, "c", constant.CodeGroupParentInvalid},
		"induk deprecated":   {a, "old", constant.CodeGroupParentInvalid},
		"4 tingkat":          {mk("d"), "c", constant.CodeGroupTooDeep},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := service.PlaceGroup(all, tc.g, tc.parent, now)
			testhelper.AssertDomainError(t, err, tc.code)
		})
	}

	// Memindahkan a (subtree a > b > c) ke bawah root lain → 4 tingkat.
	root := mk("root")
	all = append(all, root)
	_, err = service.PlaceGroup(all, a, "root", now)
	testhelper.AssertDomainError(t, err, constant.CodeGroupTooDeep)

	// Induk deprecated yang sudah terpasang boleh tetap.
	_, err = service.PlaceGroup(all, root, "", now)
	require.NoError(t, err)
	old.Status = valueobject.StatusActive
	_, err = service.PlaceGroup(all, root, "old", now)
	require.NoError(t, err)
	old.Status = valueobject.StatusDeprecated
	_, err = service.PlaceGroup(all, root, "old", now)
	assert.NoError(t, err)
}
