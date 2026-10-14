package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/port"
	"platform-service/internal/app/principal"
	"platform-service/internal/interfaces/http/httperror"
)

const principalKey = "principal"

// Authenticate mewajibkan header "Authorization: Bearer <token>" dan menyimpan
// principal hasil verifikasi ke gin context. Token kosong/invalid → 401.
func Authenticate(authn port.TokenAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			httperror.Handle(c, apperror.Unauthorized(apperror.MsgTokenMissing))
			return
		}
		p, err := authn.Authenticate(c.Request.Context(), token)
		if err != nil || p == nil {
			httperror.Handle(c, apperror.Unauthorized(apperror.MsgTokenInvalid, err))
			return
		}
		c.Set(principalKey, p)
		c.Next()
	}
}

// PrincipalFrom mengambil principal yang diset Authenticate.
func PrincipalFrom(c *gin.Context) (*principal.Principal, bool) {
	v, ok := c.Get(principalKey)
	if !ok {
		return nil, false
	}
	p, ok := v.(*principal.Principal)
	return p, ok && p != nil
}

// RequireUser hanya menerima token user.
func RequireUser() gin.HandlerFunc {
	return guard(func(p *principal.Principal) *apperror.AppError {
		if !p.IsUser() {
			return apperror.Forbidden(apperror.MsgUserTokenRequired)
		}
		return nil
	})
}

// RequireService hanya menerima token service yang memiliki scope tersebut.
func RequireService(scope string) gin.HandlerFunc {
	return guard(func(p *principal.Principal) *apperror.AppError {
		if !p.IsService() {
			return apperror.Forbidden(apperror.MsgServiceTokenRequired)
		}
		if !p.HasScope(scope) {
			return apperror.Forbidden(apperror.MsgInsufficientScope)
		}
		return nil
	})
}

// RequireUserOrService menerima token user, atau token service dengan scope tersebut.
func RequireUserOrService(scope string) gin.HandlerFunc {
	return guard(func(p *principal.Principal) *apperror.AppError {
		if p.IsUser() {
			return nil
		}
		if !p.HasScope(scope) {
			return apperror.Forbidden(apperror.MsgInsufficientScope)
		}
		return nil
	})
}

// RequirePlatformAdmin hanya menerima token user dengan users.is_platform_admin = TRUE.
func RequirePlatformAdmin() gin.HandlerFunc {
	return guard(func(p *principal.Principal) *apperror.AppError {
		if !p.IsUser() || !p.IsPlatformAdmin {
			return apperror.Forbidden(apperror.MsgPlatformAdminRequired)
		}
		return nil
	})
}

// RequireActiveTenant menolak token user yang belum memilih tenant (claim tid kosong).
// Token service lolos; pembatasan tenant-nya dicek di usecase.
func RequireActiveTenant() gin.HandlerFunc {
	return guard(func(p *principal.Principal) *apperror.AppError {
		if p.IsUser() && !p.HasActiveTenant() {
			return apperror.Forbidden(apperror.MsgTenantNotSelected)
		}
		return nil
	})
}

// guard menjalankan check terhadap principal. Tanpa principal (Authenticate belum
// dipasang di route) → 401.
func guard(check func(p *principal.Principal) *apperror.AppError) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := PrincipalFrom(c)
		if !ok {
			httperror.Handle(c, apperror.Unauthorized(apperror.MsgTokenMissing))
			return
		}
		if err := check(p); err != nil {
			httperror.Handle(c, err)
			return
		}
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
