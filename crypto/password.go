// Copyright 2020 Wearless Tech Inc All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"hash"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword - stronger encryption for storing passwords
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash if the hash (usually stored in database) matches users entered password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSecretKey generated n-byte secret key and returns in hex format string
func GenerateSecretKey(numBytes int) (string, error) {
	key := make([]byte, numBytes)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	dst := make([]byte, hex.EncodedLen(len(key)))
	hex.Encode(dst, key)

	return string(dst), nil
}

func ComputeHmac256(hashFunc func() hash.Hash, message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(hashFunc, key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateHMac256Signature validates signature from a sent payload
// Bash example creating signature: apisig=`echo -n "$nonce$key" | openssl dgst -sha256 -hmac "mysecret" -binary | xxd -p -c 256`
func ValidateHmac256Signature(hashFunc func() hash.Hash, payload string, secret string, hexDigest string) bool {
	hHash := hmac.New(hashFunc, []byte(secret))
	_, _ = hHash.Write([]byte(payload)) // assignations are required not to get an errcheck issue (linter)
	computedDigest := hHash.Sum(nil)

	digest, err := hex.DecodeString(hexDigest)
	if err != nil {
		return false
	}

	return hmac.Equal(computedDigest, digest)

}
