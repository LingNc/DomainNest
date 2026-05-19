package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TokenAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// Priority: URL param > Body token > Authorization header
		token = c.Query("token")

		if token == "" && c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				// Restore body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				var body map[string]interface{}
				if json.Unmarshal(bodyBytes, &body) == nil {
					if t, ok := body["token"].(string); ok {
						token = t
					}
				}
			}
		}

		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing token"})
			c.Abort()
			return
		}

		// Try user token first (DDNS token)
		var user model.User
		if err := db.Where("token = ?", token).First(&user).Error; err == nil {
			c.Set("user_id", user.ID)
			c.Set("username", user.Username)
			c.Set("role", user.Role)
			c.Set("auth_type", "user_token")
			c.Next()
			return
		}

		// Try RAM token
		var ramToken model.RAMToken
		if err := db.Where("token = ? AND enabled = ?", token, true).First(&ramToken).Error; err == nil {
			c.Set("user_id", ramToken.UserID)
			c.Set("ram_token_id", ramToken.ID)
			c.Set("auth_type", "ram_token")
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
		c.Abort()
	}
}
