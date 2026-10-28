package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/valueobject"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresTenantRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		tenants := persistence.NewPostgresTenantRepository(testDB)
		members := persistence.NewPostgresTenantMemberRepository(testDB)
		query := persistence.NewPostgresTenantQuery(testDB)

		tn, err := entity.NewTenant("Repo Tenant", testhelper.UniqueCode("repo"), time.Now())
		require.NoError(t, err)
		require.NoError(t, tenants.Save(ctx, tn))

		got, err := tenants.FindByID(ctx, tn.ID)
		require.NoError(t, err)
		assert.Equal(t, tn.Slug.Value(), got.Slug.Value())

		_, err = tenants.FindByID(ctx, uuid.NewString())
		testhelper.AssertDomainError(t, err, constant.CodeTenantNotFound)

		u := mustSaveUser(t, ctx, testhelper.UniqueEmail("member"))
		m, err := entity.NewTenantMember(tn.ID, u.ID, "", time.Now())
		require.NoError(t, err)
		require.NoError(t, members.Save(ctx, m))

		// Upsert: status berubah tanpa duplikasi.
		m.Status = valueobject.MemberStatusInactive
		require.NoError(t, members.Save(ctx, m))
		found, err := members.Find(ctx, tn.ID, u.ID)
		require.NoError(t, err)
		assert.Equal(t, valueobject.MemberStatusInactive, found.Status)

		_, err = members.Find(ctx, tn.ID, uuid.NewString())
		testhelper.AssertDomainError(t, err, constant.CodeMemberNotFound)

		// Query: member inactive tidak terhitung sebagai tenant milik user.
		uid := u.ID
		items, meta, err := query.ListTenants(ctx, port.TenantListReadQuery{MemberUserID: &uid})
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.Equal(t, int64(0), meta.Total)

		search := "100%_" // karakter LIKE di-escape, tidak cocok dengan semua baris
		items, _, err = query.ListTenants(ctx, port.TenantListReadQuery{Search: &search})
		require.NoError(t, err)
		assert.Empty(t, items)

		// Slug duplikat (terakhir, karena error membatalkan transaksi).
		dup, err := entity.NewTenant("Dup", tn.Slug.Value(), time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, tenants.Save(ctx, dup), constant.CodeTenantSlugDuplicate)
	})
}
