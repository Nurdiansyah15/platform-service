// Package service berisi domain service webhook.
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Sign menghasilkan nilai header X-Platform-Signature: "sha256=" + hex(HMAC-SHA256(secret, timestamp + "." + body)).
// Penerima memverifikasi dengan rumus yang sama dan menolak timestamp yang terlalu lama.
func Sign(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify membandingkan tanda tangan secara constant-time (dipakai test & SDK).
func Verify(secret string, timestamp int64, body []byte, signature string) bool {
	return hmac.Equal([]byte(Sign(secret, timestamp, body)), []byte(signature))
}
