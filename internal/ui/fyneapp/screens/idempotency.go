package screens

import (
	"crypto/rand"
	"encoding/hex"
)

func newIdempotencyKey() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "fallback-key"
	}
	return hex.EncodeToString(buf)
}
