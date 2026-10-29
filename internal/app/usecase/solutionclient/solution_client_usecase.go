package solutionclient

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	domainerr "platform-service/internal/domain/errors"
	solutionrepo "platform-service/internal/domain/solution/repository"
	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/entity"
	clientrepo "platform-service/internal/domain/solutionclient/repository"
	"platform-service/internal/domain/solutionclient/valueobject"
)

const (
	grantTypeClientCredentials = "client_credentials"
	tokenTypeBearer            = "Bearer"
)

type SolutionClientDeps struct {
	Repo      clientrepo.SolutionClientRepository
	Query     port.SolutionClientQueryReadModel
	Solutions solutionrepo.SolutionRepository
	Tokens    port.TokenIssuer
	Logger    *slog.Logger
}

type SolutionClientUseCase struct {
	deps SolutionClientDeps
	now  func() time.Time
}

func NewSolutionClientUseCase(deps SolutionClientDeps) *SolutionClientUseCase {
	return &SolutionClientUseCase{deps: deps, now: time.Now}
}

// Create membuat kredensial client baru untuk solution. Secret hanya dikembalikan sekali.
//
// Required — auth: user (platform admin) | scope: -
func (uc *SolutionClientUseCase) Create(ctx context.Context, solutionID string, req dto.CreateSolutionClientRequest) (*dto.SolutionClientWithSecretResponse, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, mapSolutionClientDomainError(err)
	}

	clientID, err := newClientID()
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	secret, secretHash, err := newClientSecret()
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}

	c, err := entity.NewSolutionClient(entity.NewSolutionClientParams{
		SolutionID:       solutionID,
		Name:             req.Name,
		ClientID:         clientID,
		ClientSecretHash: secretHash,
		Scopes:           req.Scopes,
	}, uc.now())
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	if err := uc.deps.Repo.Save(ctx, c); err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	return &dto.SolutionClientWithSecretResponse{SolutionClientResponse: toResponse(c), ClientSecret: secret}, nil
}

// List menampilkan client milik solution (tanpa secret).
//
// Required — auth: user (platform admin) | scope: -
func (uc *SolutionClientUseCase) List(ctx context.Context, solutionID string, req dto.ListSolutionClientQuery) ([]dto.SolutionClientResponse, dto.Meta, error) {
	if _, err := uc.deps.Solutions.FindByID(ctx, solutionID); err != nil {
		return nil, dto.Meta{}, mapSolutionClientDomainError(err)
	}
	items, meta, err := uc.deps.Query.List(ctx, port.SolutionClientListReadQuery{
		SolutionID: solutionID, Status: req.Status, Search: req.Search, PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return nil, dto.Meta{}, mapSolutionClientDomainError(err)
	}
	return items, meta, nil
}

// Get mengambil detail client (tanpa secret).
//
// Required — auth: user (platform admin) | scope: -
func (uc *SolutionClientUseCase) Get(ctx context.Context, id string) (*dto.SolutionClientResponse, error) {
	c, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	resp := toResponse(c)
	return &resp, nil
}

// Update mengubah nama / scope client. Scope baru langsung berlaku untuk token yang sudah
// terbit karena ResolveServicePrincipal meng-intersect dengan scope client terkini.
//
// Required — auth: user (platform admin) | scope: -
func (uc *SolutionClientUseCase) Update(ctx context.Context, id string, req dto.UpdateSolutionClientRequest) (*dto.SolutionClientResponse, error) {
	c, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	if err := c.Update(req.Name, req.Scopes, uc.now()); err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	if err := uc.deps.Repo.Save(ctx, c); err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	resp := toResponse(c)
	return &resp, nil
}

// Revoke menonaktifkan client secara permanen; token yang sudah terbit langsung ditolak.
// Idempotent.
//
// Required — auth: user (platform admin) | scope: -
func (uc *SolutionClientUseCase) Revoke(ctx context.Context, id string) (*dto.SolutionClientResponse, error) {
	c, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	if c.IsActive() {
		c.Revoke(uc.now())
		if err := uc.deps.Repo.Save(ctx, c); err != nil {
			return nil, mapSolutionClientDomainError(err)
		}
	}
	resp := toResponse(c)
	return &resp, nil
}

// RotateSecret membuat secret baru; secret lama langsung tidak berlaku.
// Token yang sudah terbit tetap berlaku sampai exp.
//
// Required — auth: user (platform admin) | scope: -
func (uc *SolutionClientUseCase) RotateSecret(ctx context.Context, id string) (*dto.SolutionClientWithSecretResponse, error) {
	c, err := uc.deps.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	secret, secretHash, err := newClientSecret()
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	if err := c.RotateSecret(secretHash, uc.now()); err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	if err := uc.deps.Repo.Save(ctx, c); err != nil {
		return nil, mapSolutionClientDomainError(err)
	}
	return &dto.SolutionClientWithSecretResponse{SolutionClientResponse: toResponse(c), ClientSecret: secret}, nil
}

// IssueToken menjalankan grant client_credentials. Error dikembalikan sebagai
// *dto.OAuthError (format RFC 6749).
//
// Required — auth: publik (kredensial client) | scope: -
func (uc *SolutionClientUseCase) IssueToken(ctx context.Context, req dto.ClientCredentialsRequest) (*dto.OAuthTokenResponse, error) {
	if req.GrantType == "" {
		return nil, dto.NewOAuthInvalidRequest("grant_type is required")
	}
	if req.GrantType != grantTypeClientCredentials {
		return nil, dto.NewOAuthUnsupportedGrantType()
	}
	invalidClient := dto.NewOAuthInvalidClient(req.UsedBasicAuth)
	if req.ClientID == "" || req.ClientSecret == "" {
		return nil, invalidClient
	}

	c, err := uc.deps.Repo.FindByClientID(ctx, req.ClientID)
	if err != nil {
		if isDomainCode(err, constant.CodeClientNotFound) {
			// Tetap hitung hash agar waktu respons setara dengan client yang ada.
			_ = hashSecret(req.ClientSecret)
			return nil, invalidClient
		}
		return nil, mapSolutionClientDomainError(err)
	}
	if !c.SecretMatches(hashSecret(req.ClientSecret)) || !c.IsActive() {
		return nil, invalidClient
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, c.SolutionID)
	if err != nil || !sol.IsActive() {
		return nil, invalidClient
	}

	requested, err := valueobject.ParseScopeParam(req.Scope)
	if err != nil {
		return nil, dto.NewOAuthInvalidScope()
	}
	granted, err := c.GrantableScopes(requested)
	if err != nil {
		return nil, dto.NewOAuthInvalidScope()
	}

	issued, err := uc.deps.Tokens.IssueServiceToken(ctx, port.ServiceTokenClaims{
		ClientID:   c.ClientID,
		SolutionID: c.SolutionID,
		Scopes:     granted.Values(),
	})
	if err != nil {
		return nil, mapSolutionClientDomainError(err)
	}

	if err := uc.deps.Repo.TouchLastUsed(ctx, c.ID, uc.now()); err != nil {
		uc.deps.Logger.WarnContext(ctx, "gagal mencatat last_used_at solution client",
			slog.String("client_id", c.ClientID), slog.Any("error", err))
	}

	return &dto.OAuthTokenResponse{
		AccessToken: issued.Token,
		TokenType:   tokenTypeBearer,
		ExpiresIn:   int64(time.Until(issued.ExpiresAt).Round(time.Second).Seconds()),
		Scope:       strings.Join(granted.Values(), " "),
	}, nil
}

// ResolveServicePrincipal memvalidasi token service terhadap DB di setiap request:
// client masih active, solution masih active, dan claim sol cocok. Scope dipotong
// ke scope client saat ini sehingga pengurangan scope langsung berlaku.
//
// Required — auth: internal (authenticator) | scope: -
func (uc *SolutionClientUseCase) ResolveServicePrincipal(ctx context.Context, p *principal.Principal) (*principal.Principal, error) {
	invalid := domainerr.New(constant.CodeClientInvalid)

	c, err := uc.deps.Repo.FindByClientID(ctx, p.ClientID)
	if err != nil {
		return nil, err
	}
	if !c.IsActive() || c.SolutionID != p.SolutionID {
		return nil, invalid
	}
	sol, err := uc.deps.Solutions.FindByID(ctx, c.SolutionID)
	if err != nil {
		return nil, err
	}
	if !sol.IsActive() {
		return nil, invalid
	}

	claimed, err := valueobject.NewScopes(p.Scopes)
	if err != nil {
		return nil, invalid
	}
	resolved := *p
	resolved.Scopes = claimed.Intersect(c.Scopes).Values()
	return &resolved, nil
}
