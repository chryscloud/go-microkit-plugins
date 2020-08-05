package crypto

import (
	"crypto/hmac"
	"encoding/hex"
	"hash"
)

func ComputeHmac(hashFunc func() hash.Hash, message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(hashFunc, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateHMacSignature validates signature from a sent payload
func ValidateHmacSignature(hashFunc func() hash.Hash, payload string, secret string, hexDigest string) bool {
	hHash := hmac.New(hashFunc, []byte(secret))
	_, _ = hHash.Write([]byte(payload)) // assignations are required not to get an errcheck issue (linter)
	computedDigest := hHash.Sum(nil)

	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return false
	}

	return hmac.Equal(computedDigest, digest)

}
