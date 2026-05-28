package middleware

import (
	"net/http"
	"strings"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

// CloudflareAuth verifies Cloudflare API token from Authorization header.
func CloudflareAuth(ramTokenSvc *service.RAMTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"errors":  []gin.H{{"code": 10000, "message": "Missing Authorization header"}},
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"errors":  []gin.H{{"code": 10000, "message": "Invalid Authorization format"}},
			})
			c.Abort()
			return
		}

		ramToken, err := ramTokenSvc.ValidateAndLookup(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"errors":  []gin.H{{"code": 10000, "message": "Invalid API token"}},
			})
			c.Abort()
			return
		}

		c.Set("user_id", ramToken.UserID)
		c.Set("ram_token_id", ramToken.ID)
		c.Set("ram_token", ramToken)
		c.Next()
	}
}