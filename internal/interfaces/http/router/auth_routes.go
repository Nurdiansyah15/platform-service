package router

import (
	"platform-service/internal/interfaces/http/handler/api"
	"platform-service/internal/interfaces/http/middleware"
)

// RegisterAuthRoutes — Authentication & Identity (spec §3.1).
func RegisterAuthRoutes(r Routes, h *api.AuthHandler) {
	public := r.V1.Group("/auth")
	public.POST("/login", h.Login)
	public.POST("/refresh", h.Refresh)
	public.POST("/token-exchange", h.TokenExchange)

	protected := r.Authenticated.Group("/auth", middleware.RequireUser())
	protected.POST("/logout", h.Logout)
	protected.GET("/me", h.Me)
	protected.GET("/companies", h.Companies)
	protected.POST("/switch-company", h.SwitchCompany)
	protected.POST("/authorize", h.Authorize)
	protected.GET("/apps", h.Apps)
}

// RegisterUserRoutes — manajemen user (spec §3.1).
func RegisterUserRoutes(r Routes, h *api.UserHandler) {
	g := r.Authenticated.Group("/users")
	g.GET("/me", middleware.RequireUser(), h.Me)
	g.PATCH("/me", middleware.RequireUser(), h.UpdateMe)

	admin := g.Group("", middleware.RequirePlatformAdmin())
	admin.GET("", h.List)
	admin.POST("", h.Create)
	admin.GET("/:userId", h.Get)
	admin.PATCH("/:userId", h.Update)
	admin.PATCH("/:userId/status", h.ChangeStatus)
}
