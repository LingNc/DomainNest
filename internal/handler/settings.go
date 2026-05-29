package handler

import (
	"encoding/json"
	"net/http"

	"domainnest/internal/errs"
	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SettingsHandler struct {
	settingsService *service.SettingsService
	db              *gorm.DB
}

func NewSettingsHandler(db *gorm.DB, settingsService *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService, db: db}
}

func (h *SettingsHandler) Get(c *gin.Context) {
	category := c.Param("category")
	value, err := h.settingsService.Get(category)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	if value == "" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": nil})
		return
	}

	var data interface{}
	if json.Unmarshal([]byte(value), &data) != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": value})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": data})
}

func (h *SettingsHandler) Set(c *gin.Context) {
	category := c.Param("category")

	body, err := c.GetRawData()
	if err != nil {
		errs.JSONErrorCode(c, errs.ReadRequestBodyFailed)
		return
	}

	var check interface{}
	if json.Unmarshal(body, &check) != nil {
		errs.JSONErrorCode(c, errs.InvalidJSON)
		return
	}

	if err := h.settingsService.Set(category, string(body)); err != nil {
		errs.JSONError(c, err)
		return
	}

	userID := c.GetUint64("user_id")
	targetID := uint64(0)
	middleware.LogOperation(h.db, userID, "update_settings", "setting", &targetID,
		map[string]interface{}{"category": category}, c.ClientIP())

	if category == "smtp" {
		go func() {
			svc := service.NewMessageService(h.db)
			// Notify all admins
			var admins []model.User
			h.db.Where("role = ? OR is_super_admin = ?", "admin", true).Find(&admins)
			for _, admin := range admins {
				if admin.ID != userID {
					svc.SendSystemNotification(admin.ID, "SMTP配置变更", "系统SMTP邮件配置已被修改", "", "")
				}
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "设置已保存"})
}

func (h *SettingsHandler) TestSMTP(c *gin.Context) {
	cfg := h.settingsService.GetSMTPConfig()
	if cfg == nil {
		errs.JSONErrorCode(c, errs.SMTPNotConfigured)
		return
	}

	var req struct {
		To string `json:"to" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	emailSvc := service.NewEmailService(cfg)
	if err := emailSvc.SendTestEmail(req.To); err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "测试邮件已发送"})
}
