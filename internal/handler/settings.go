package handler

import (
	"encoding/json"
	"net/http"

	"domainnest/internal/middleware"
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求体无效"})
		return
	}

	var check interface{}
	if json.Unmarshal(body, &check) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "JSON格式无效"})
		return
	}

	if err := h.settingsService.Set(category, string(body)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	userID := c.GetUint64("user_id")
	targetID := uint64(0)
	middleware.LogOperation(h.db, userID, "update_settings", "setting", &targetID,
		map[string]interface{}{"category": category}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "设置已保存"})
}

func (h *SettingsHandler) TestSMTP(c *gin.Context) {
	cfg := h.settingsService.GetSMTPConfig()
	if cfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SMTP未配置"})
		return
	}

	var req struct {
		To string `json:"to" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	emailSvc := service.NewEmailService(cfg)
	if err := emailSvc.SendTestEmail(req.To); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "发送测试邮件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "测试邮件已发送"})
}
