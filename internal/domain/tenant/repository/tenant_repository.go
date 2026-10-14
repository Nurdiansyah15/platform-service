package repository

import (
	"context"

	"platform-service/internal/domain/tenant/entity"
)

type TenantRepository interface {
	// Save insert/update. Slug bentrok → CodeTenantSlugDuplicate.
	Save(ctx context.Context, t *entity.Tenant) error
	FindByID(ctx context.Context, id string) (*entity.Tenant, error)
}

type TenantMemberRepository interface {
	// Save insert/update keanggotaan (upsert per tenant_id + user_id).
	Save(ctx context.Context, m *entity.TenantMember) error
	Find(ctx context.Context, tenantID, userID string) (*entity.TenantMember, error)
}
