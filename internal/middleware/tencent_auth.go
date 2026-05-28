package middleware

import (
	"net/http"
	"strings"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

func TencentAuth(ramTokenSvc *service.RAMTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusOK, gin.H{
				"Response": gin.H{
					"Error": gin.H{
						"Code":    "AuthFailure",
						"Message": "Missing Authorization header",
					},
				},
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			// Try TC3 format: TC3-HMAC-SHA256 Credential=xxx/...
			// For now, reject non-Bearer tokens
			c.JSON(http.StatusOK, gin.H{
				"Response": gin.H{
					"Error": gin.H{
						"Code":    "AuthFailure",
						"Message": "Invalid Authorization format. Use: Bearer <token>",
					},
				},
			})
			c.Abort()
			return
		}

		ramToken, err := ramTokenSvc.ValidateAndLookup(token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"Response": gin.H{
					"Error": gin.H{
						"Code":    "AuthFailure",
						"Message": "Invalid token",
					},
				},
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