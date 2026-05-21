package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncHandler struct {
	syncService   *service.SyncService
	recordService *service.RecordService
	db            *gorm.DB
}

func NewSyncHandler(syncService *service.SyncService, recordService *service.RecordService, db *gorm.DB) *SyncHandler {
	return &SyncHandler{
		syncService:   syncService,
		recordService: recordService,
		db:            db,
	}
}

func (h *SyncHandler) ManualSync(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	userID := c.GetUint64("user_id")
	if err := h.recordService.CheckPermission(userID, nodeID, 4); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作"})
		return
	}

	var recordIDs []uint64
	h.db.Model(&model.DNSRecord{}).
		Where("node_id = ? AND (sync_status = ? OR sync_status = ?)", nodeID, "pending", "failed").
		Pluck("id", &recordIDs)

	if len(recordIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"synced": 0, "failed": 0}})
		return
	}

	synced, failed := h.syncService.ManualSync(recordIDs)

	middleware.LogOperation(h.db, userID, "manual_sync", "domain_node", &nodeID,
		map[string]interface{}{"record_count": len(recordIDs), "synced": synced, "failed": failed}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"synced": synced, "failed": failed}})
}

func (h *SyncHandler) GetSyncLogs(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	userID := c.GetUint64("user_id")
	if err := h.recordService.CheckPermission(userID, nodeID, 1); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权访问"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.syncService.GetSyncLogs(nodeID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
