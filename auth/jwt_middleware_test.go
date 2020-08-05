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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chryscloud/go-microkit-plugins/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"gopkg.in/go-playground/assert.v1"
)

type customClaims struct {
	MyProperty string `json:"myproperty"`
	jwt.StandardClaims
}

func setupRouter(conf *config.YamlConfig) *gin.Engine {
	r := gin.Default()

	keys := func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JWTToken.SecretKey), nil
	}
	mw := JwtMiddleware(conf, &customClaims{}, jwt.SigningMethodHS256, keys)

	jwtSecured := r.Group("/test", mw)
	{
		jwtSecured.GET("/ping", func(c *gin.Context) {
			if claims, ok := c.Get(JWTClaimsContextKey); ok {
				custom := claims.(*customClaims)
				fmt.Printf(custom.MyProperty)
				if custom.MyProperty == "" {
					c.String(404, "customClaim MyProperty expected")
					return
				}
			} else {
				c.String(404, "custom claim not found")
				return
			}
			c.String(200, "pong")
		})
	}

	return r
}

func TestJwtAuth(t *testing.T) {

	conf := &config.YamlConfig{
		JWTToken: config.JWTTokenSection{
			Enabled:   true,
			SecretKey: "my test secret key here",
		},
	}
	router := setupRouter(conf)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/ping", nil)

	testClaim := customClaims{
		MyProperty: "MyProperty",
	}

	token, err := NewJWTToken([]byte(conf.JWTToken.SecretKey), jwt.SigningMethodHS256, testClaim)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", token)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestJWTWithCookie(t *testing.T) {
	conf := &config.YamlConfig{
		JWTToken: config.JWTTokenSection{
			Enabled:    true,
			SecretKey:  "my test secret key here",
			CookieName: "testcookie",
		},
	}
	router := setupRouter(conf)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/ping", nil)

	testClaim := customClaims{
		MyProperty: "MyProperty",
	}

	token, err := NewJWTToken([]byte(conf.JWTToken.SecretKey), jwt.SigningMethodHS256, testClaim)
	if err != nil {
		t.Fatal(err)
	}

	req.AddCookie(&http.Cookie{
		Name:     conf.JWTToken.CookieName,
		Path:     "/",
		Domain:   "",
		HttpOnly: true,
		Value:    token,
	})
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestMalformedKey(t *testing.T) {
	conf := &config.YamlConfig{
		JWTToken: config.JWTTokenSection{
			Enabled:   true,
			SecretKey: "my test secret key here",
		},
	}
	router := setupRouter(conf)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/ping", nil)
	req.Header.Set("Authorization", "test token")
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestInvalidKey(t *testing.T) {
	conf := &config.YamlConfig{
		JWTToken: config.JWTTokenSection{
			Enabled:   true,
			SecretKey: "my test secret key here",
		},
	}
	router := setupRouter(conf)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/ping", nil)

	token, err := NewJWTToken([]byte("another key"), jwt.SigningMethodHS256, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", token)
	router.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code)
}
