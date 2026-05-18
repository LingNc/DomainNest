package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecordHandler struct {
	recordService *service.RecordService
	db            *gorm.DB
}

func NewRecordHandler(recordService *service.RecordService, db *gorm.DB) *RecordHandler {
	return &RecordHandler{recordService: recordService, db: db}
}

func (h *RecordHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	records, err := h.recordService.GetRecords(nodeID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

func (h *RecordHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	var req struct {
		Host       string `json:"host" binding:"required"`
		RecordType string `json:"record_type" binding:"required"`
		Value      string `json:"value" binding:"required"`
		TTL        int    `json:"ttl"`
		Priority   *int   `json:"priority"`
		Line       string `json:"line"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.CreateRecord(nodeID, userID, req.Host, req.RecordType, req.Value, req.TTL, req.Priority, req.Line)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "create_record", "dns_record", &record.ID,
		map[string]interface{}{"host": record.Host, "type": record.RecordType, "value": record.Value}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid record id"})
		return
	}

	var req struct {
		Value    string `json:"value"`
		TTL      *int   `json:"ttl"`
		Priority *int   `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.UpdateRecord(recordID, userID, req.Value, req.TTL, req.Priority)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "update_record", "dns_record", &recordID,
		map[string]interface{}{"value": record.Value}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid record id"})
		return
	}

	if err := h.recordService.DeleteRecord(recordID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_record", "dns_record", &recordID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted successfully"})
}
