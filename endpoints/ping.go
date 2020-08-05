package endpoints

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// PingEndpoint godoc
// @Summary PingEndpoint
// @Description Often used for healthchecks on kubernetes deployments (lightweight service)
// @Tags Public
// @Accept json
// @Produce json
// @Router /ping [get]
func PingEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong at " + time.Now().Format("20060102150405"),
	})
}
