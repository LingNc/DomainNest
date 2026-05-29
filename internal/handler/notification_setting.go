package handler

import (
	"net/http"

	"domainnest/internal/errs"
	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotificationSettingHandler handles notification mute preferences.
type NotificationSettingHandler struct {
	db *gorm.DB
}

func NewNotificationSettingHandler(db *gorm.DB) *NotificationSettingHandler {
	return &NotificationSettingHandler{db: db}
}

// List returns the current user's notification settings.
func (h *NotificationSettingHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var settings []model.NotificationSetting
	if err := h.db.Where("user_id = ?", userID).Find(&settings).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": settings})
}

// Update batch-creates or updates mute preferences for the current user.
func (h *NotificationSettingHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req []struct {
		Category string `json:"category" binding:"required"`
		Muted    bool   `json:"muted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	for _, item := range req {
		setting := model.NotificationSetting{
			UserID:   userID,
			Category: item.Category,
			Muted:    item.Muted,
		}
		h.db.Where("user_id = ? AND category = ?", userID, item.Category).
			Assign(model.NotificationSetting{Muted: item.Muted}).
			FirstOrCreate(&setting)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "设置已更新"})
}
