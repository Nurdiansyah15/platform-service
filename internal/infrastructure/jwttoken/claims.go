package jwttoken

import "github.com/golang-jwt/jwt/v5"

const (
	typUser    = "user"
	typService = "service"

	// audiencePlatform adalah aud token service (spec §2.8).
	audiencePlatform = "platform"
)

// payload adalah bentuk claim saat ditandatangani. aud ditulis sebagai string
// tunggal agar sesuai spec (aud = solutions.code / "platform").
type payload struct {
	Iss   string `json:"iss"`
	Sub   string `json:"sub"`
	Aud   string `json:"aud"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
	Jti   string `json:"jti"`
	Typ   string `json:"typ"`
	Cid   string `json:"cid,omitempty"`
	Sid   string `json:"sid,omitempty"`
	Sol   string `json:"sol,omitempty"`
	Scope string `json:"scope,omitempty"`
}

// parsedClaims adalah bentuk claim saat diverifikasi.
type parsedClaims struct {
	jwt.RegisteredClaims
	Typ   string `json:"typ"`
	Cid   string `json:"cid"`
	Sid   string `json:"sid"`
	Sol   string `json:"sol"`
	Scope string `json:"scope"`
}
