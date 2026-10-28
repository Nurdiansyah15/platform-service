package organization

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/service"
	userconstant "platform-service/internal/domain/user/constant"
	userentity "platform-service/internal/domain/user/entity"
)

type CompanyMemberUseCase struct {
	deps   OrganizationDeps
	access *service.AccessPolicy
	now    func() time.Time
}

func NewCompanyMemberUseCase(deps OrganizationDeps) *CompanyMemberUseCase {
	return &CompanyMemberUseCase{deps: deps, access: service.NewAccessPolicy(deps.Members, deps.Subscriptions), now: time.Now}
}

// List menampilkan member company beserta data user.
//
// Required — auth: user (platform admin, atau member active dengan cid = companyId) | scope: -
func (uc *CompanyMemberUseCase) List(ctx context.Context, actor *principal.Principal, companyID string, req dto.ListCompanyMemberQuery) ([]dto.CompanyMemberResponse, dto.Meta, error) {
	if err := uc.ensureCompany(ctx, actor, companyID); err != nil {
		return nil, dto.Meta{}, err
	}
	items, meta, err := uc.deps.CompanyQuery.ListMembers(ctx, port.CompanyMemberListReadQuery{
		CompanyID: companyID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapCompanyDomainError(err)
	}
	out := make([]dto.CompanyMemberResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.CompanyMemberResponse{
			CompanyID: companyID, UserID: it.UserID, Email: it.Email, Name: it.Name, Status: it.Status, JoinedAt: it.JoinedAt,
		})
	}
	return out, meta, nil
}

// Add menambahkan user terdaftar ke company. Member inactive diaktifkan kembali.
//
// Required — auth: user (platform admin, atau member active dengan cid = companyId) | scope: -
func (uc *CompanyMemberUseCase) Add(ctx context.Context, actor *principal.Principal, companyID string, req dto.AddCompanyMemberRequest) (*dto.CompanyMemberResponse, error) {
	if err := uc.ensureCompany(ctx, actor, companyID); err != nil {
		return nil, err
	}
	u, err := uc.resolveUser(ctx, req)
	if err != nil {
		return nil, mapCompanyDomainError(err)
	}

	now := uc.now()
	var m *entity.CompanyMember
	err = uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		existing, err := uc.deps.Members.Find(ctx, companyID, u.ID)
		switch {
		case err == nil:
			if err := existing.Rejoin(req.Status, now); err != nil {
				return err
			}
			m = existing
		case isDomainCode(err, constant.CodeMemberNotFound):
			if m, err = entity.NewCompanyMember(companyID, u.ID, req.Status, now); err != nil {
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
		return nil, mapCompanyDomainError(err)
	}

	return &dto.CompanyMemberResponse{
		CompanyID: companyID, UserID: u.ID, Email: u.Email.Value(), Name: u.Name, Status: m.Status.String(), JoinedAt: m.JoinedAt,
	}, nil
}

// ensureCompany memeriksa hak akses lalu memastikan company ada.
func (uc *CompanyMemberUseCase) ensureCompany(ctx context.Context, actor *principal.Principal, companyID string) error {
	if err := uc.access.EnsureCanAccess(ctx, accessorOf(actor), companyID); err != nil {
		return mapCompanyDomainError(err)
	}
	if _, err := uc.deps.Companies.FindByID(ctx, companyID); err != nil {
		return mapCompanyDomainError(err)
	}
	return nil
}

func (uc *CompanyMemberUseCase) resolveUser(ctx context.Context, req dto.AddCompanyMemberRequest) (*userentity.User, error) {
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
