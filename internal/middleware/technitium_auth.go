package middleware

import (
	"bytes"
	"io"
	"net/http"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

func TechnitiumAuth(ramTokenSvc *service.RAMTokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil {
			c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": "missing request body"})
			c.Abort()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": "failed to read body"})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		c.Request.ParseForm()
		token := c.PostForm("token")
		if token == "" {
			c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": "missing token parameter"})
			c.Abort()
			return
		}

		ramToken, err := ramTokenSvc.ValidateAndLookup(token)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": "invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", ramToken.UserID)
		c.Set("ram_token_id", ramToken.ID)
		c.Set("ram_token", ramToken)
		c.Next()
	}
}