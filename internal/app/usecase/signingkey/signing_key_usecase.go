package signingkey

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	"platform-service/internal/app/port"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/signingkey/constant"
	"platform-service/internal/domain/signingkey/entity"
	signingrepo "platform-service/internal/domain/signingkey/repository"
)

type SigningKeyDeps struct {
	Repo     signingrepo.SigningKeyRepository
	KeyStore port.KeyStore
	Logger   *slog.Logger
}

type SigningKeyUseCase struct {
	deps SigningKeyDeps
	now  func() time.Time
}

func NewSigningKeyUseCase(deps SigningKeyDeps) *SigningKeyUseCase {
	return &SigningKeyUseCase{deps: deps, now: time.Now}
}

// EnsureActiveKey membuat signing key active bila belum ada. Dipanggil saat startup.
// Aman dijalankan bersamaan oleh beberapa instance: yang kalah balapan memakai key
// milik pemenang (partial unique index uq_signing_keys_single_active).
//
// Required — auth: internal (startup) | scope: -
func (uc *SigningKeyUseCase) EnsureActiveKey(ctx context.Context) error {
	if _, err := uc.deps.Repo.FindActive(ctx); err == nil {
		return nil
	} else if !isDomainCode(err, constant.CodeSigningKeyNotFound) {
		return err
	}

	ref, pubPEM, err := uc.deps.KeyStore.Generate(ctx)
	if err != nil {
		return err
	}
	key, err := entity.NewSigningKey(pubPEM, ref, uc.now())
	if err != nil {
		return err
	}
	if err := uc.deps.Repo.Save(ctx, key); err != nil {
		if isDomainCode(err, constant.CodeSigningKeyActiveExists) {
			return nil
		}
		return err
	}
	uc.deps.Logger.InfoContext(ctx, "signing key active dibuat", slog.String("kid", key.KID))
	return nil
}

// JWKS mengembalikan public key active + retiring.
//
// Required — auth: publik | scope: -
func (uc *SigningKeyUseCase) JWKS(ctx context.Context) (*dto.JWKSResponse, error) {
	keys, err := uc.deps.Repo.ListPublished(ctx)
	if err != nil {
		return nil, apperror.Internal(string(constant.CodeSigningKeyQueryFailed), err)
	}

	resp := &dto.JWKSResponse{Keys: make([]dto.JWK, 0, len(keys))}
	for _, k := range keys {
		jwk, err := toJWK(k)
		if err != nil {
			// Satu key rusak tidak boleh membuat semua solution gagal memverifikasi token.
			uc.deps.Logger.ErrorContext(ctx, "signing key tidak valid, dilewati dari JWKS",
				slog.String("kid", k.KID), slog.Any("error", err))
			continue
		}
		resp.Keys = append(resp.Keys, jwk)
	}
	return resp, nil
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}
