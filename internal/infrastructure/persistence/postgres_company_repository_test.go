package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/valueobject"
	"platform-service/internal/infrastructure/persistence"
	"platform-service/internal/testhelper"
)

func TestPostgresCompanyRepositories(t *testing.T) {
	testhelper.WithTestTx(t, testDB, func(ctx context.Context) {
		companies := persistence.NewPostgresCompanyRepository(testDB)
		members := persistence.NewPostgresCompanyMemberRepository(testDB)
		query := persistence.NewPostgresCompanyQuery(testDB)

		tn, err := entity.NewCompany("Repo Company", testhelper.UniqueCode("repo"), time.Now())
		require.NoError(t, err)
		require.NoError(t, companies.Save(ctx, tn))

		got, err := companies.FindByID(ctx, tn.ID)
		require.NoError(t, err)
		assert.Equal(t, tn.Slug.Value(), got.Slug.Value())

		_, err = companies.FindByID(ctx, uuid.NewString())
		testhelper.AssertDomainError(t, err, constant.CodeCompanyNotFound)

		u := mustSaveUser(t, ctx, testhelper.UniqueEmail("member"))
		m, err := entity.NewCompanyMember(tn.ID, u.ID, "", time.Now())
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

		// Query: member inactive tidak terhitung sebagai company milik user.
		uid := u.ID
		items, meta, err := query.ListCompanies(ctx, port.CompanyListReadQuery{MemberUserID: &uid})
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.Equal(t, int64(0), meta.Total)

		search := "100%_" // karakter LIKE di-escape, tidak cocok dengan semua baris
		items, _, err = query.ListCompanies(ctx, port.CompanyListReadQuery{Search: &search})
		require.NoError(t, err)
		assert.Empty(t, items)

		// Slug duplikat (terakhir, karena error membatalkan transaksi).
		dup, err := entity.NewCompany("Dup", tn.Slug.Value(), time.Now())
		require.NoError(t, err)
		testhelper.AssertDomainError(t, companies.Save(ctx, dup), constant.CodeCompanySlugDuplicate)
	})
}
