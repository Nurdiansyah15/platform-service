package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"platform-service/internal/app/apperror"
	"platform-service/internal/domain/authsession/constant"
	domainerr "platform-service/internal/domain/errors"
	solutionconstant "platform-service/internal/domain/solution/constant"
	userconstant "platform-service/internal/domain/user/constant"
)

// refreshTokenBytes — 256 bit entropi.
const refreshTokenBytes = 32

// newRefreshToken membuat refresh token opaque beserta hash yang disimpan di DB.
func newRefreshToken() (token, hash string, err error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth: generate refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashRefreshToken(token), nil
}

// hashRefreshToken — SHA-256 cukup (bukan bcrypt) karena token acak 256 bit,
// bukan password yang bisa ditebak.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func mapAuthDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeInvalidCredentials, constant.CodeRefreshTokenInvalid:
		return apperror.Unauthorized(string(de.Code))
	case constant.CodeUserNotActive, constant.CodeCompanyNotSelectable, constant.CodeAuthorizeConsoleOnly:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeSolutionUnavailable, constant.CodeInvalidRedirectURI, constant.CodeAuthCodeChallengeInvalid:
		return apperror.Unprocessable(string(de.Code), nil)
	case constant.CodeInvalidGrant:
		return apperror.BadRequest(string(de.Code))
	case constant.CodeSessionNotFound, constant.CodeSessionRevoked, constant.CodeSessionExpired, constant.CodeSessionInvalid:
		return apperror.Unauthorized(apperror.MsgTokenInvalid)
	case userconstant.CodeUserNotFound, solutionconstant.CodeSolutionNotFound:
		return apperror.NotFound(string(de.Code))
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

// asAppError — error dari dalam Transactor.WithTx bisa sudah berupa AppError (dipetakan
// di dalam fn) atau error mentah transaksi.
func asAppError(err error) error {
	var ae *apperror.AppError
	if errors.As(err, &ae) {
		return err
	}
	return mapAuthDomainError(err)
}

func isNotFound(err error) bool {
	return isDomainCode(err, userconstant.CodeUserNotFound) ||
		isDomainCode(err, solutionconstant.CodeSolutionNotFound) ||
		isDomainCode(err, constant.CodeSessionNotFound)
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
