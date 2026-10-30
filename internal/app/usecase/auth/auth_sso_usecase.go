package auth

import (
	"context"
	"net/url"
	"time"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/principal"
	"platform-service/internal/domain/authsession/constant"
	"platform-service/internal/domain/authsession/entity"
	domainerr "platform-service/internal/domain/errors"
)

// authorizationCodeTTL — kode SSO hanya untuk satu redirect, jadi umurnya pendek.
const authorizationCodeTTL = 60 * time.Second

// Authorize menerbitkan authorization code sekali pakai untuk masuk ke solution lain
// tanpa login ulang (SSO). Hanya sesi console yang boleh memanggilnya: console berperan
// sebagai identity provider, solution menukar kode lewat ExchangeCode.
//
// Required — auth: user (sesi console) | scope: -
func (uc *AuthUseCase) Authorize(ctx context.Context, actor *principal.Principal, req dto.AuthorizeRequest) (*dto.AuthorizeResponse, error) {
	if actor.SolutionCode != uc.deps.ConsoleSolutionCode {
		return nil, mapAuthDomainError(domainerr.New(constant.CodeAuthorizeConsoleOnly))
	}
	sol, err := uc.activeSolutionByCode(ctx, req.Solution)
	if err != nil {
		return nil, err
	}
	if !sol.AllowsRedirect(req.RedirectURI) {
		return nil, mapAuthDomainError(domainerr.New(constant.CodeInvalidRedirectURI))
	}

	companyID, err := uc.authorizeCompany(ctx, actor, sol.ID, req.CompanyID)
	if err != nil {
		return nil, err
	}

	code, codeHash, err := newRefreshToken()
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	ac, err := entity.NewAuthorizationCode(entity.NewAuthorizationCodeParams{
		CodeHash:            codeHash,
		UserID:              actor.UserID,
		SolutionID:          sol.ID,
		CompanyID:           companyID,
		ParentSessionID:     actor.SessionID,
		RedirectURI:         req.RedirectURI,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		TTL:                 authorizationCodeTTL,
	}, uc.now())
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	if err := uc.deps.Codes.Create(ctx, ac); err != nil {
		return nil, mapAuthDomainError(err)
	}

	return &dto.AuthorizeResponse{
		RedirectTo: redirectWithCode(req.RedirectURI, code, req.State),
		Code:       code,
		ExpiresAt:  ac.ExpiresAt,
	}, nil
}

// authorizeCompany menentukan company aktif sesi solution: company yang diminta (wajib bisa
// dipilih), lalu company aktif console, lalu satu-satunya company yang bisa dipilih.
func (uc *AuthUseCase) authorizeCompany(ctx context.Context, actor *principal.Principal, solutionID string, requested *string) (*string, error) {
	if requested != nil {
		ok, err := uc.companyStillAccessible(ctx, actor.UserID, *requested)
		if err != nil {
			return nil, mapAuthDomainError(err)
		}
		if !ok {
			return nil, mapAuthDomainError(domainerr.New(constant.CodeCompanyNotSelectable))
		}
		// GATE-SUBSCRIPTION (nonaktif sementara): company tidak wajib subscribe solution tujuan.
		// usable, err := uc.deps.Subscriptions.HasUsable(ctx, *requested, solutionID)
		// if err != nil {
		// 	return nil, mapAuthDomainError(err)
		// }
		// if !usable {
		// 	return nil, mapAuthDomainError(domainerr.New(constant.CodeCompanyNotSelectable))
		// }
		return requested, nil
	}
	if actor.CompanyID != "" {
		ok, err := uc.companyStillAccessible(ctx, actor.UserID, actor.CompanyID)
		if err != nil {
			return nil, mapAuthDomainError(err)
		}
		// GATE-SUBSCRIPTION (nonaktif sementara): saat aktif, company console juga wajib
		// subscribe solution tujuan; bila tidak, jatuh ke pemilihan company di bawah.
		if ok {
			cid := actor.CompanyID
			return &cid, nil
		}
	}
	companyID, err := uc.singleSelectableCompany(ctx, actor.UserID, solutionID)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	return companyID, nil
}

// ExchangeCode menukar authorization code (+ code_verifier PKCE) menjadi sesi baru pada
// solution tujuan. Token hasilnya identik dengan hasil Login, sehingga refresh,
// switch-company, dan authz berjalan seperti biasa. Kode hanya bisa ditukar sekali.
//
// Required — auth: publik (authorization code) | scope: -
func (uc *AuthUseCase) ExchangeCode(ctx context.Context, req dto.TokenExchangeRequest, client dto.ClientInfo) (*dto.AuthTokenResponse, error) {
	invalid := mapAuthDomainError(domainerr.New(constant.CodeInvalidGrant))

	var resp *dto.AuthTokenResponse
	err := uc.deps.Tx.WithTx(ctx, func(ctx context.Context) error {
		ac, err := uc.deps.Codes.FindByCodeHash(ctx, hashRefreshToken(req.Code))
		if err != nil {
			if isDomainCode(err, constant.CodeAuthCodeNotFound) {
				return invalid
			}
			return mapAuthDomainError(err)
		}
		if err := ac.Redeem(req.CodeVerifier, req.RedirectURI, uc.now()); err != nil {
			return invalid
		}
		if err := uc.deps.Codes.MarkUsed(ctx, ac); err != nil {
			if isDomainCode(err, constant.CodeAuthCodeAlreadyUsed) {
				return invalid
			}
			return mapAuthDomainError(err)
		}

		// Kondisi bisa berubah sejak authorize: sesi console di-logout, user / solution
		// dinonaktifkan, atau user keluar dari company.
		parent, err := uc.deps.Sessions.FindByID(ctx, ac.ParentSessionID)
		if err != nil || parent.EnsureUsable(uc.now()) != nil {
			return invalid
		}
		u, err := uc.deps.Users.FindByID(ctx, ac.UserID)
		if err != nil || !u.IsActive() {
			return invalid
		}
		sol, err := uc.deps.Solutions.FindByID(ctx, ac.SolutionID)
		if err != nil || !sol.IsActive() {
			return invalid
		}
		if ac.CompanyID != nil {
			ok, err := uc.companyStillAccessible(ctx, ac.UserID, *ac.CompanyID)
			if err != nil {
				return mapAuthDomainError(err)
			}
			if !ok {
				return invalid
			}
		}

		parentID := parent.ID
		resp, err = uc.startSession(ctx, sessionStart{
			UserID: ac.UserID, Solution: sol, CompanyID: ac.CompanyID, ParentSessionID: &parentID, Client: client,
		})
		return err
	})
	if err != nil {
		return nil, asAppError(err)
	}
	return resp, nil
}

// Apps menampilkan solution yang bisa dibuka user untuk company aktif token: launcher di
// console dan menu apps di navbar setiap solution. Token dari solution mana pun diterima.
// Tanpa company aktif hanya platform admin yang melihat daftar.
// GATE-SUBSCRIPTION (nonaktif sementara): saat aktif, hanya solution yang di-subscribe
// company (trialing / active) yang ditampilkan.
//
// Required — auth: user | scope: -
func (uc *AuthUseCase) Apps(ctx context.Context, actor *principal.Principal) ([]dto.AppResponse, error) {
	out := []dto.AppResponse{}
	if actor.CompanyID == "" && !actor.IsPlatformAdmin {
		return out, nil
	}
	if actor.CompanyID != "" {
		ok, err := uc.companyStillAccessible(ctx, actor.UserID, actor.CompanyID)
		if err != nil {
			return nil, mapAuthDomainError(err)
		}
		if !ok && !actor.IsPlatformAdmin {
			return out, nil
		}
	}

	items, err := uc.deps.Solutions.ListLaunchable(ctx)
	if err != nil {
		return nil, mapAuthDomainError(err)
	}
	for _, s := range items {
		// GATE-SUBSCRIPTION (nonaktif sementara): lewati solution yang tidak di-subscribe company.
		// if actor.CompanyID != "" && s.Code.Value() != uc.deps.ConsoleSolutionCode {
		// 	usable, err := uc.deps.Subscriptions.HasUsable(ctx, actor.CompanyID, s.ID)
		// 	if err != nil {
		// 		return nil, mapAuthDomainError(err)
		// 	}
		// 	if !usable {
		// 		continue
		// 	}
		// }
		out = append(out, dto.AppResponse{
			ID:          s.ID,
			Code:        s.Code.Value(),
			Name:        s.Name,
			Description: s.Description,
			IconURL:     s.IconURL,
			LaunchURL:   *s.LaunchURL,
			IsCurrent:   s.Code.Value() == actor.SolutionCode,
		})
	}
	return out, nil
}

// redirectWithCode menambahkan code & state ke query redirect_uri (query lama dipertahankan).
func redirectWithCode(redirectURI, code, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
