package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

// BasicAuthRAM validates HTTP Basic Auth against RAM token AccessKeyID/AccessKeySecret.
func BasicAuthRAM(ramTokenSvc *service.RAMTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Basic ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
			c.Abort()
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid base64 in Authorization"})
			c.Abort()
			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Basic Auth format"})
			c.Abort()
			return
		}

		token, err := ramTokenSvc.LookupByAccessKeyID(parts[0])
		if err != nil || token.AccessKeySecret != parts[1] {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		c.Set("user_id", token.UserID)
		c.Set("ram_token_id", token.ID)
		c.Set("ram_token", token)
		c.Next()
	}
}