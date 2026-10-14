package signingkey

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"platform-service/internal/app/dto"
	"platform-service/internal/domain/signingkey/entity"
)

// toJWK mengonversi public key PEM (PKIX, RSA) ke JWK.
func toJWK(k *entity.SigningKey) (dto.JWK, error) {
	block, _ := pem.Decode([]byte(k.PublicKeyPEM))
	if block == nil {
		return dto.JWK{}, errors.New("public key bukan PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return dto.JWK{}, fmt.Errorf("parse public key: %w", err)
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return dto.JWK{}, errors.New("public key bukan RSA")
	}
	return dto.JWK{
		Kty: "RSA",
		Kid: k.KID,
		Use: "sig",
		Alg: k.Algorithm,
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}, nil
}
