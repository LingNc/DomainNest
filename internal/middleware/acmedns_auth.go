package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func AcmeDNSAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Basic ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid auth"})
			c.Abort()
			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid auth format"})
			c.Abort()
			return
		}

		var account model.AcmeDNSAccount
		if err := db.Where("username = ?", parts[0]).First(&account).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(parts[1])); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			c.Abort()
			return
		}

		c.Set("user_id", account.UserID)
		c.Set("acmedns_account", &account)
		c.Next()
	}
}