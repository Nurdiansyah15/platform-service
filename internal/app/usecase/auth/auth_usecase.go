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
	userrepo "platform-service/internal/domain/user/repository"
)

type AuthDeps struct {
	Users           userrepo.UserRepository
	Solutions       solutionrepo.SolutionRepository
	Sessions        sessionrepo.AuthSessionRepository
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
	session, err := entity.NewAuthSession(entity.NewAuthSessionParams{
		UserID:           u.ID,
		SolutionID:       sol.ID,
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
