package tenant

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/tenant/constant"
	"platform-service/internal/domain/tenant/entity"
	"platform-service/internal/domain/tenant/service"
	userconstant "platform-service/internal/domain/user/constant"
	userentity "platform-service/internal/domain/user/entity"
)

type MemberUseCase struct {
	deps   TenantDeps
	access *service.AccessPolicy
	now    func() time.Time
}

func NewMemberUseCase(deps TenantDeps) *MemberUseCase {
	return &MemberUseCase{deps: deps, access: service.NewAccessPolicy(deps.Members), now: time.Now}
}

// List menampilkan member tenant beserta data user.
//
// Required — auth: user (platform admin, atau member active dengan tid = tenantId) | scope: -
func (uc *MemberUseCase) List(ctx context.Context, actor *principal.Principal, tenantID string, req dto.ListTenantMemberQuery) ([]dto.TenantMemberResponse, dto.Meta, error) {
	if err := uc.ensureTenant(ctx, actor, tenantID); err != nil {
		return nil, dto.Meta{}, err
	}
	items, meta, err := uc.deps.Query.ListMembers(ctx, port.TenantMemberListReadQuery{
		TenantID: tenantID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapTenantDomainError(err)
	}
	out := make([]dto.TenantMemberResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.TenantMemberResponse{
			TenantID: tenantID, UserID: it.UserID, Email: it.Email, Name: it.Name, Status: it.Status, JoinedAt: it.JoinedAt,
		})
	}
	return out, meta, nil
}

// Add menambahkan user terdaftar ke tenant. Member inactive diaktifkan kembali.
//
// Required — auth: user (platform admin, atau member active dengan tid = tenantId) | scope: -
func (uc *MemberUseCase) Add(ctx context.Context, actor *principal.Principal, tenantID string, req dto.AddTenantMemberRequest) (*dto.TenantMemberResponse, error) {
	if err := uc.ensureTenant(ctx, actor, tenantID); err != nil {
		return nil, err
	}
	u, err := uc.resolveUser(ctx, req)
	if err != nil {
		return nil, mapTenantDomainError(err)
	}

	now := uc.now()
	var m *entity.TenantMember
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		existing, err := uc.deps.Members.Find(ctx, tenantID, u.ID)
		switch {
		case err == nil:
			if err := existing.Rejoin(req.Status, now); err != nil {
				return err
			}
			m = existing
		case isDomainCode(err, constant.CodeMemberNotFound):
			if m, err = entity.NewTenantMember(tenantID, u.ID, req.Status, now); err != nil {
				return err
			}
		default:
			return err
		}
		if err := uc.deps.Members.Save(ctx, m); err != nil {
			return err
		}
		return uc.deps.Outbox.Append(ctx, memberAddedEvent(m))
	})
	if err != nil {
		return nil, mapTenantDomainError(err)
	}

	return &dto.TenantMemberResponse{
		TenantID: tenantID, UserID: u.ID, Email: u.Email.Value(), Name: u.Name, Status: m.Status.String(), JoinedAt: m.JoinedAt,
	}, nil
}

// ensureTenant memeriksa hak akses lalu memastikan tenant ada.
func (uc *MemberUseCase) ensureTenant(ctx context.Context, actor *principal.Principal, tenantID string) error {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), tenantID); err != nil {
		return mapTenantDomainError(err)
	}
	if _, err := uc.deps.Tenants.FindByID(ctx, tenantID); err != nil {
		return mapTenantDomainError(err)
	}
	return nil
}

func (uc *MemberUseCase) resolveUser(ctx context.Context, req dto.AddTenantMemberRequest) (*userentity.User, error) {
	var (
		u   *userentity.User
		err error
	)
	switch {
	case req.UserID != nil && req.Email != nil, req.UserID == nil && req.Email == nil:
		return nil, domainerr.New(constant.CodeMemberIdentityRequired)
	case req.UserID != nil:
		u, err = uc.deps.Users.FindByID(ctx, *req.UserID)
	default:
		u, err = uc.deps.Users.FindByEmail(ctx, *req.Email)
	}
	if err != nil {
		if isDomainCode(err, userconstant.CodeUserNotFound) {
			return nil, domainerr.New(constant.CodeMemberUserNotFound)
		}
		return nil, err
	}
	return u, nil
}
