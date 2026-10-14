package solutionclient

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	domainerr "platform-service/internal/domain/errors"
	solutionconstant "platform-service/internal/domain/solution/constant"
	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/entity"
)

const (
	clientIDPrefix    = "cli_"
	clientIDBytes     = 12 // 96 bit → 24 karakter hex
	clientSecretBytes = 32 // 256 bit
)

func newClientID() (string, error) {
	b := make([]byte, clientIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("solution client: generate client_id: %w", err)
	}
	return clientIDPrefix + hex.EncodeToString(b), nil
}

func newClientSecret() (secret, hash string, err error) {
	b := make([]byte, clientSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("solution client: generate secret: %w", err)
	}
	secret = base64.RawURLEncoding.EncodeToString(b)
	return secret, hashSecret(secret), nil
}

// hashSecret — SHA-256 cukup karena secret acak 256 bit (bukan password manusia).
func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func mapSolutionClientDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeClientNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeClientNameInvalid, constant.CodeClientScopeInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	case constant.CodeClientRevoked:
		return apperror.Conflict(string(de.Code), err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

func toResponse(c *entity.SolutionClient) dto.SolutionClientResponse {
	return dto.SolutionClientResponse{
		ID:         c.ID,
		SolutionID: c.SolutionID,
		Name:       c.Name,
		ClientID:   c.ClientID,
		Scopes:     c.Scopes.Values(),
		Status:     c.Status.String(),
		LastUsedAt: c.LastUsedAt,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		RevokedAt:  c.RevokedAt,
	}
}
