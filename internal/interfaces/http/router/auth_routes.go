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

	protected := r.Authenticated.Group("/auth", middleware.RequireUser())
	protected.POST("/logout", h.Logout)
	protected.GET("/me", h.Me)
}

// RegisterUserRoutes — manajemen user (spec §3.1).
func RegisterUserRoutes(r Routes, h *api.UserHandler) {
	g := r.Authenticated.Group("/users")
	g.POST("", middleware.RequirePlatformAdmin(), h.Create)
}
