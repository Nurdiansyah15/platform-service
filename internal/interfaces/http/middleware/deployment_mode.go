package middleware

import (
	"github.com/gin-gonic/gin"

	"platform-service/internal/app/apperror"
	"platform-service/internal/config"
	"platform-service/internal/interfaces/http/httperror"
)

// DeploymentGuard membatasi endpoint berdasarkan deployment mode (spec §2.14).
// Pengecekan state license (read_only/invalid) ditambahkan di Fase 17.
type DeploymentGuard struct {
	mode config.DeploymentMode
}

func NewDeploymentGuard(mode config.DeploymentMode) *DeploymentGuard {
	return &DeploymentGuard{mode: mode}
}

// LicenseManaged menolak endpoint yang datanya dikelola lewat license di mode
// on_prem (403 LICENSE_MANAGED). Di mode cloud tidak berpengaruh.
func (g *DeploymentGuard) LicenseManaged() gin.HandlerFunc {
	return func(c *gin.Context) {
		if g.mode == config.DeploymentOnPrem {
			httperror.Handle(c, apperror.LicenseManaged())
			return
		}
		c.Next()
	}
}

// OnPremOnly menyembunyikan endpoint (404) di mode cloud, mis. endpoint license.
func (g *DeploymentGuard) OnPremOnly() gin.HandlerFunc {
	return g.onlyIn(config.DeploymentOnPrem)
}

// CloudOnly menyembunyikan endpoint (404) di mode on_prem.
func (g *DeploymentGuard) CloudOnly() gin.HandlerFunc {
	return g.onlyIn(config.DeploymentCloud)
}

func (g *DeploymentGuard) onlyIn(mode config.DeploymentMode) gin.HandlerFunc {
	return func(c *gin.Context) {
		if g.mode != mode {
			httperror.Handle(c, apperror.RouteNotFound())
			return
		}
		c.Next()
	}
}
