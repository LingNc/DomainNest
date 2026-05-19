package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RAMTokenHandler struct {
	ramTokenService *service.RAMTokenService
	db              *gorm.DB
}

func NewRAMTokenHandler(ramTokenService *service.RAMTokenService, db *gorm.DB) *RAMTokenHandler {
	return &RAMTokenHandler{ramTokenService: ramTokenService, db: db}
}

func (h *RAMTokenHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")

	tokens, err := h.ramTokenService.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": tokens})
}

func (h *RAMTokenHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Name           string   `json:"name" binding:"required"`
		AllowedDomains []uint64 `json:"allowed_domains"`
		AllowedTypes   []string `json:"allowed_types"`
		AllowedIPs     []string `json:"allowed_ips"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	token, err := h.ramTokenService.Create(userID, req.Name, req.AllowedDomains, req.AllowedTypes, req.AllowedIPs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "create_ram_token", "ram_token", &token.ID,
		map[string]interface{}{"name": token.Name}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": token})
}

func (h *RAMTokenHandler) Get(c *gin.Context) {
	userID := c.GetUint64("user_id")
	tokenID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的令牌ID"})
		return
	}

	token, err := h.ramTokenService.Get(tokenID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": token})
}

func (h *RAMTokenHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")
	tokenID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的令牌ID"})
		return
	}

	var req struct {
		Name           string   `json:"name"`
		Enabled        *bool    `json:"enabled"`
		AllowedDomains []uint64 `json:"allowed_domains"`
		AllowedTypes   []string `json:"allowed_types"`
		AllowedIPs     []string `json:"allowed_ips"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	token, err := h.ramTokenService.Update(tokenID, userID, req.Name, req.Enabled, req.AllowedDomains, req.AllowedTypes, req.AllowedIPs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "update_ram_token", "ram_token", &tokenID,
		map[string]interface{}{"name": token.Name}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": token})
}

func (h *RAMTokenHandler) ResetToken(c *gin.Context) {
	userID := c.GetUint64("user_id")
	tokenID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的令牌ID"})
		return
	}

	token, err := h.ramTokenService.ResetToken(tokenID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "reset_ram_token", "ram_token", &tokenID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": token})
}

func (h *RAMTokenHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	tokenID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的令牌ID"})
		return
	}

	if err := h.ramTokenService.Delete(tokenID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_ram_token", "ram_token", &tokenID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}

func parseJSONUint64Array(s string) []uint64 {
	if s == "" || s == "[]" {
		return nil
	}
	var arr []uint64
	json.Unmarshal([]byte(s), &arr)
	return arr
}

func parseJSONStringArray(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var arr []string
	json.Unmarshal([]byte(s), &arr)
	return arr
}
