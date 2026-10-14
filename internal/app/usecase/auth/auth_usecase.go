package auth

import (
	"context"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/app/usecase/user"
	"platform-service/internal/domain/authsession/constant"
	"platform-service/internal/domain/authsession/entity"
	sessionrepo "platform-service/internal/domain/authsession/repository"
	domainerr "platform-service/internal/domain/errors"
	solutionentity "platform-service/internal/domain/solution/entity"
	solutionrepo "platform-service/internal/domain/solution/repository"
	subrepo "platform-service/internal/domain/subscription/repository"
	tenantconstant "platform-service/internal/domain/tenant/constant"
	tenantrepo "platform-service/internal/domain/tenant/repository"
	userrepo "platform-service/internal/domain/user/repository"
)

type AuthDeps struct {
	Users           userrepo.UserRepository
	Solutions       solutionrepo.SolutionRepository
	Sessions        sessionrepo.AuthSessionRepository
	Tenants         tenantrepo.TenantRepository
	Members         tenantrepo.TenantMemberRepository
	Subscriptions   subrepo.SubscriptionRepository
	Selectable      port.SubscriptionQueryReadModel
	Hasher          port.PasswordHasher
	Tokens          port.TokenIssuer
	RefreshTokenTTL time.Duration
}

type AuthUseCase struct {
	deps AuthDeps
	now  func() time.Time
	// dummyHash dipakai saat email tidak terdaftar agar waktu respons login
	// setara dengan email terdaftar (mencegah enumerasi user lewat timing).
	dummyHash string
}

func NewAuthUseCase(deps AuthDeps) (*AuthUseCase, error) {
	dummy, err := deps.Hasher.Hash("dummy-password-for-timing")
	if err != nil {
		return nil, err
	}
	return &AuthUseCase{deps: deps, now: time.Now, dummyHash: dummy}, nil
}

// Login memverifikasi kredensial dan membuat sesi baru pada solution tujuan.
//
// Required — auth: publik | scope: -
func (uc *AuthUseCase) Login(ctx context.Context, req dto.LoginRequest, client dto.ClientInfo) (*dto.AuthTokenResponse, error) {
	sol, err := uc.activeSolutionByCode(ctx, req.Solution)
	if err != nil {
		return nil, err
	}

	u, err := uc.deps.Users.FindByEmail(ctx, req.Email)
	if err != nil {
		if !isNotFound(err) {
			return nil, mapAuthDomainError(err)
		}
		uc.deps.Hasher.Verify(uc.dummyHash, req.Password)
		return nil, mapAuthDomainError(domainerr.New(constant.CodeInvalidCredentials))
	}
	if !u.HasPassword() || !uc.deps.Hasher.Verify(*u.PasswordHash, req.Password) {
		return nil, mapAuthDomainError(domainerr.New(constant.CodeInvalidCredentials))
	}
	// Status dicek setelah password benar agar tidak membocorkan status akun.
	if !u.IsActive() {
		return nil, mapAuthDomainError(domainerr.New(constant.CodeUserNotActive))
	}

	refreshToken, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	// Spec §2.7: bila user hanya punya satu tenant yang bisa dipilih untuk solution
	// ini, tenant tersebut langsung menjadi tenant aktif sesi.
	selectable, err := uc.deps.Selectable.ListSelectableTenants(ctx, u.ID, sol.ID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	var tenantID *string
	if len(selectable) == 1 {
		tenantID = &selectable[0].ID
	}

	session, err := entity.NewAuthSession(entity.NewAuthSessionParams{
		UserID:           u.ID,
		SolutionID:       sol.ID,
		TenantID:         tenantID,
		RefreshTokenHash: refreshHash,
		UserAgent:        client.UserAgent,
		IPAddress:        client.IPAddress,
		TTL:              uc.deps.RefreshTokenTTL,
	}, uc.now())
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	if err := uc.deps.Sessions.Create(ctx, session); err != nil {
		return nil, mapAuthDomainError(err)
	}

	return uc.issueTokens(ctx, session, sol.Code.Value(), refreshToken)
}

// Refresh menukar refresh token dengan pasangan token baru (refresh token di-rotate).
//
// Required — auth: publik (refresh token) | scope: -
func (uc *AuthUseCase) Refresh(ctx context.Context, req dto.RefreshTokenRequest) (*dto.AuthTokenResponse, error) {
	invalid := mapAuthDomainError(domainerr.New(constant.CodeRefreshTokenInvalid))

	session, err := uc.deps.Sessions.FindByRefreshTokenHash(ctx, hashRefreshToken(req.RefreshToken))
	if err != nil {
		if isNotFound(err) {
			return nil, invalid
		}
		return nil, mapAuthDomainError(err)
	}
	if err := session.EnsureUsable(uc.now()); err != nil {
		return nil, invalid
	}

	u, err := uc.deps.Users.FindByID(ctx, session.UserID)
	if err != nil || !u.IsActive() {
		return nil, invalid
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, session.SolutionID)
	if err != nil || !sol.IsActive() {
		return nil, invalid
	}
	// Spec §2.7: refresh ditolak bila membership atau tenant aktif sesi tidak lagi active.
	if session.TenantID != nil {
		if ok, err := uc.tenantStillAccessible(ctx, u.ID, *session.TenantID); err != nil {
			return nil, mapAuthDomainError(err)
		} else if !ok {
			return nil, invalid
		}
	}

	refreshToken, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	previousHash, err := session.Rotate(refreshHash, uc.deps.RefreshTokenTTL, uc.now())
	if err != nil {
		return nil, invalid
	}
	if err := uc.deps.Sessions.SaveRotation(ctx, session, previousHash); err != nil {
		if isDomainCode(err, constant.CodeSessionRotationConflict) {
			return nil, invalid
		}
		return nil, mapAuthDomainError(err)
	}

	return uc.issueTokens(ctx, session, sol.Code.Value(), refreshToken)
}

// Logout me-revoke sesi milik token yang sedang dipakai.
//
// Required — auth: user | scope: -
func (uc *AuthUseCase) Logout(ctx context.Context, actor *principal.Principal) error {
	session, err := uc.deps.Sessions.FindByID(ctx, actor.SessionID)
	if err != nil {
		return mapAuthDomainError(err)
	}
	session.Revoke(uc.now())
	if err := uc.deps.Sessions.SaveRevocation(ctx, session); err != nil {
		return mapAuthDomainError(err)
	}
	return nil
}

// Me mengembalikan user dan sesi dari token yang sedang dipakai.
//
// Required — auth: user | scope: -
func (uc *AuthUseCase) Me(ctx context.Context, actor *principal.Principal) (*dto.MeResponse, error) {
	u, err := uc.deps.Users.FindByID(ctx, actor.UserID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	session, err := uc.deps.Sessions.FindByID(ctx, actor.SessionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, session.SolutionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	return &dto.MeResponse{
		User: user.ToUserResponse(u),
		Session: dto.SessionResponse{
			ID:        session.ID,
			Solution:  dto.SolutionSummary{ID: sol.ID, Code: sol.Code.Value(), Name: sol.Name},
			TenantID:  session.TenantID,
			ExpiresAt: session.ExpiresAt,
			CreatedAt: session.CreatedAt,
		},
	}, nil
}

// Tenants menampilkan tenant yang bisa dipilih untuk solution sesi: user member active,
// tenant active, dan tenant subscribe solution (trialing / active).
//
// Required — auth: user (tid boleh kosong) | scope: -
func (uc *AuthUseCase) Tenants(ctx context.Context, actor *principal.Principal) ([]dto.SelectableTenantResponse, error) {
	session, err := uc.deps.Sessions.FindByID(ctx, actor.SessionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	items, err := uc.deps.Selectable.ListSelectableTenants(ctx, actor.UserID, session.SolutionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	out := make([]dto.SelectableTenantResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.SelectableTenantResponse{
			ID: it.ID, Name: it.Name, Slug: it.Slug, Status: it.Status, IsActive: it.ID == actor.TenantID,
		})
	}
	return out, nil
}

// SwitchTenant mengganti tenant aktif sesi lalu menerbitkan access & refresh token baru
// (spec §2.7). Refresh token lama tidak berlaku lagi.
//
// Required — auth: user (tid boleh kosong) | scope: -
func (uc *AuthUseCase) SwitchTenant(ctx context.Context, actor *principal.Principal, req dto.SwitchTenantRequest) (*dto.AuthTokenResponse, error) {
	session, err := uc.deps.Sessions.FindByID(ctx, actor.SessionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, session.SolutionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}

	notSelectable := mapAuthDomainError(domainerr.New(constant.CodeTenantNotSelectable))
	ok, err := uc.tenantStillAccessible(ctx, actor.UserID, req.TenantID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	if !ok {
		return nil, notSelectable
	}
	usable, err := uc.deps.Subscriptions.HasUsable(ctx, req.TenantID, session.SolutionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	if !usable {
		return nil, notSelectable
	}

	refreshToken, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	previousHash, err := session.SwitchTenant(req.TenantID, refreshHash, uc.deps.RefreshTokenTTL, uc.now())
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	if err := uc.deps.Sessions.SaveRotation(ctx, session, previousHash); err != nil {
		if isDomainCode(err, constant.CodeSessionRotationConflict) {
			return nil, mapAuthDomainError(domainerr.New(constant.CodeSessionInvalid))
		}
		return nil, mapAuthDomainError(err)
	}
	return uc.issueTokens(ctx, session, sol.Code.Value(), refreshToken)
}

// tenantStillAccessible — tenant ada & active dan user member active-nya.
func (uc *AuthUseCase) tenantStillAccessible(ctx context.Context, userID, tenantID string) (bool, error) {
	t, err := uc.deps.Tenants.FindByID(ctx, tenantID)
	if err != nil {
		if isDomainCode(err, tenantconstant.CodeTenantNotFound) {
			return false, nil
		}
		return false, err
	}
	if !t.IsActive() {
		return false, nil
	}
	m, err := uc.deps.Members.Find(ctx, tenantID, userID)
	if err != nil {
		if isDomainCode(err, tenantconstant.CodeMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return m.IsActive(), nil
}

// ResolveUserPrincipal memvalidasi principal dari token user terhadap DB dan
// melengkapinya (IsPlatformAdmin). Dipanggil authenticator di setiap request:
// token ditolak bila sesi revoked/expired, user tidak active, solution tidak
// active, atau claim token tidak cocok dengan sesi.
//
// Required — auth: internal (authenticator) | scope: -
func (uc *AuthUseCase) ResolveUserPrincipal(ctx context.Context, p *principal.Principal) (*principal.Principal, error) {
	invalid := domainerr.New(constant.CodeSessionInvalid)

	session, err := uc.deps.Sessions.FindByID(ctx, p.SessionID)
	if err != nil {
		return nil, err
	}
	if err := session.EnsureUsable(uc.now()); err != nil {
		return nil, err
	}
	if session.UserID != p.UserID || derefString(session.TenantID) != p.TenantID {
		return nil, invalid
	}

	sol, err := uc.deps.Solutions.FindByID(ctx, session.SolutionID)
	if err != nil {
		return nil, err
	}
	if !sol.IsActive() || sol.Code.Value() != p.SolutionCode {
		return nil, invalid
	}

	u, err := uc.deps.Users.FindByID(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive() {
		return nil, invalid
	}

	resolved := *p
	resolved.IsPlatformAdmin = u.IsPlatformAdmin
	return &resolved, nil
}

func (uc *AuthUseCase) activeSolutionByCode(ctx context.Context, code string) (*solutionentity.Solution, error) {
	sol, err := uc.deps.Solutions.FindByCode(ctx, code)
	if err != nil {
		if isNotFound(err) {
			return nil, mapAuthDomainError(domainerr.New(constant.CodeSolutionUnavailable))
		}
		return nil, mapAuthDomainError(err)
	}
	if !sol.IsActive() {
		return nil, mapAuthDomainError(domainerr.New(constant.CodeSolutionUnavailable))
	}
	return sol, nil
}

func (uc *AuthUseCase) issueTokens(ctx context.Context, s *entity.AuthSession, solutionCode, refreshToken string) (*dto.AuthTokenResponse, error) {
	access, err := uc.deps.Tokens.IssueUserToken(ctx, port.UserTokenClaims{
		UserID:       s.UserID,
		SolutionCode: solutionCode,
		TenantID:     derefString(s.TenantID),
		SessionID:    s.ID,
	})
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	return &dto.AuthTokenResponse{
		AccessToken:           access.Token,
		TokenType:             "Bearer",
		ExpiresIn:             int64(time.Until(access.ExpiresAt).Round(time.Second).Seconds()),
		AccessTokenExpiresAt:  access.ExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: s.ExpiresAt,
		SessionID:             s.ID,
		TenantID:              s.TenantID,
	}, nil
}
