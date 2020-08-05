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
	"net/http"
	"regexp"

	"github.com/chryscloud/go-microkit-plugins/config"
	"github.com/gin-gonic/gin"
)

// TokenMiddleware simple token authorization
func TokenMiddleware(conf *config.YamlConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if conf.AuthToken.Enabled {
			shouldCheck := true
			if conf.AuthToken.Path != "" {
				matched, _ := regexp.MatchString(conf.AuthToken.Path, c.Request.RequestURI)
				shouldCheck = matched
			}
			if shouldCheck {
				reqToken := c.GetHeader(conf.AuthToken.Header)
				if reqToken != conf.AuthToken.Token {
					c.JSON(http.StatusForbidden, gin.H{"error": "Authorization failed"})
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}
