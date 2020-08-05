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

package auth

import (
	"errors"
	"net/http"

	"github.com/chryscloud/go-microkit-plugins/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

const (
	// JWTTokenContextKey holds the key used to store a JWT Token in the
	// context.
	JWTTokenContextKey string = "JWTToken"

	// JWTClaimsContextKey holds the key used to store the JWT Claims in the
	// context.
	JWTClaimsContextKey string = "JWTClaims"
)

var (
	// ErrClaimNotFound when claim is expected in the gin context
	ErrClaimNotFound = errors.New("claim not found in context")
)

// NewJWTToken - method for generating new jwt tokens
func NewJWTToken(key []byte, method jwt.SigningMethod, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(method, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// JwtMiddleware for Gin server if enabled
func JwtMiddleware(conf *config.YamlConfig, newClaims jwt.Claims, method jwt.SigningMethod, keyFunc jwt.Keyfunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if conf.JWTToken.Enabled {
			reqToken := c.GetHeader("Authorization")
			if reqToken == "" {
				// also check cookies if not found in header Authorization
				cook, err := c.Cookie(conf.JWTToken.CookieName)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization failed"})
					c.Abort()
					return
				}
				reqToken = cook
			}
			token, err := jwt.ParseWithClaims(reqToken, newClaims, func(token *jwt.Token) (interface{}, error) {
				// since we only use the one private key to sign the tokens,
				// we also only use its public counter part to verify
				return keyFunc(token)
			})
			if err != nil {
				if e, ok := err.(*jwt.ValidationError); ok {
					switch {
					case e.Errors&jwt.ValidationErrorMalformed != 0:
						c.JSON(http.StatusBadRequest, gin.H{"error": "JWT Token is malformed"})
						c.Abort()
						return
					case e.Errors&jwt.ValidationErrorExpired != 0:
						c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT Token is expired"})
						c.Abort()
						return
					case e.Errors&jwt.ValidationErrorNotValidYet != 0:
						c.JSON(http.StatusUnauthorized, gin.H{"error": "token is not valid yet"})
						c.Abort()
						return
					case e.Inner != nil:
						c.JSON(http.StatusUnauthorized, gin.H{"error": e.Inner.Error()})
						c.Abort()
						return
					}
				}
				c.JSON(http.StatusUnauthorized, gin.H{"error": "internal error"})
				c.Abort()
				return
			}

			if !token.Valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
				c.Abort()
				return
			}

			c.Set(JWTClaimsContextKey, token.Claims)
			c.Set(JWTTokenContextKey, token)
		}
		c.Next()
	}
}
