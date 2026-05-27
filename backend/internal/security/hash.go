package security

import (
	"crypto/sha256"
	"encoding/hex"
)

// Sha256Hex returns the SHA256 hash of the input string in hex.
func Sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
