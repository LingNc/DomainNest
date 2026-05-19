package handler

import (
	"encoding/json"
	"net/http"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	settingsService *service.SettingsService
}

func NewSettingsHandler(settingsService *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request body"})
		return
	}

	var check interface{}
	if json.Unmarshal(body, &check) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid JSON"})
		return
	}

	if err := h.settingsService.Set(category, string(body)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "settings saved"})
}

func (h *SettingsHandler) TestSMTP(c *gin.Context) {
	cfg := h.settingsService.GetSMTPConfig()
	if cfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "SMTP not configured"})
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
	go emailSvc.SendPasswordReset(req.To, "https://example.com/test-reset-link")

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "test email sent"})
}
