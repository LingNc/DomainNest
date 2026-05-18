package handler

import (
	"net/http"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type DDNSHandler struct {
	ddnsService *service.DDNSService
}

func NewDDNSHandler(ddnsService *service.DDNSService) *DDNSHandler {
	return &DDNSHandler{ddnsService: ddnsService}
}

func (h *DDNSHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Domain     string `json:"domain" binding:"required"`
		IP         string `json:"ip" binding:"required"`
		RecordType string `json:"record_type"`
		TTL        int    `json:"ttl"`
		Token      string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	result, err := h.ddnsService.Update(userID, req.Domain, req.IP, req.RecordType, req.TTL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
