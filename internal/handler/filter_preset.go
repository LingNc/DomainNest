package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FilterPresetHandler struct {
	filterPresetService *service.FilterPresetService
	db                  *gorm.DB
}

func NewFilterPresetHandler(filterPresetService *service.FilterPresetService, db *gorm.DB) *FilterPresetHandler {
	return &FilterPresetHandler{filterPresetService: filterPresetService, db: db}
}

func (h *FilterPresetHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")

	presets, err := h.filterPresetService.ListPresets(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": presets})
}

func (h *FilterPresetHandler) Save(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Name    string `json:"name" binding:"required"`
		Filters string `json:"filters" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	preset, err := h.filterPresetService.SavePreset(userID, req.Name, req.Filters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": preset})
}

func (h *FilterPresetHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	presetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的预设ID"})
		return
	}

	if err := h.filterPresetService.DeletePreset(presetID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}
