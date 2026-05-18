package middleware

import (
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

		if token == "" {
			var body struct {
				Token string `json:"token"`
			}
			if err := c.ShouldBindJSON(&body); err == nil {
				token = body.Token
				c.Set("_body_token", body)
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

		var user model.User
		if err := db.Where("token = ?", token).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Next()
	}
}
