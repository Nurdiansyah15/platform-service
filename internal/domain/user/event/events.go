// Package event mendefinisikan event outbox user (spec §2.9).
package event

import "time"

const (
	TypeUserUpdated       = "user.updated"
	TypeUserStatusChanged = "user.status_changed"
)

// UserPayload — event user tidak punya company & solution, jadi user_id wajib ada
// agar bisa di-route ke solution tempat user menjadi member company (spec §2.9).
type UserPayload struct {
	UserID          string    `json:"user_id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	IsPlatformAdmin bool      `json:"is_platform_admin"`
	UpdatedAt       time.Time `json:"updated_at"`
}
