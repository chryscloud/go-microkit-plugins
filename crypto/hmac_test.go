package crypto

import (
	"crypto/sha256"
	"testing"
)

func TestHMac(t *testing.T) {
	payload := "this is example payload"
	mac := ComputeHmac(sha256.New, payload, "mysecret")

	isValid := ValidateHmacSignature(sha256.New, payload, "mysecret", mac)
	if !isValid {
		t.Error("hmac validation failed")
	}
}
