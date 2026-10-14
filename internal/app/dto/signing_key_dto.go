package dto

// JWKSResponse adalah JSON Web Key Set (RFC 7517). Dikirim apa adanya, tanpa
// envelope respond, karena dibaca langsung oleh library JWT di sisi solution.
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}
