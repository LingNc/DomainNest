package middleware

import (
	"time"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OnlineTracker updates the user's last_active_at on each authenticated request.
// Uses a goroutine to avoid blocking the response.
func OnlineTracker(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// Only track if user is authenticated (user_id exists in context)
		if userID, exists := c.Get("user_id"); exists {
			now := time.Now()
			go db.Model(&model.User{}).Where("id = ?", userID.(uint64)).
				UpdateColumn("last_active_at", now)
		}
	}
}
