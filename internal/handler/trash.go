package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type TrashHandler struct {
	trashService *service.TrashService
}

func NewTrashHandler(trashService *service.TrashService) *TrashHandler {
	return &TrashHandler{trashService: trashService}
}

func (h *TrashHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var q service.TrashQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	result, err := h.trashService.ListTrash(userID, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *TrashHandler) Trash(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	if err := h.trashService.TrashRecord(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已移入回收站"})
}

func (h *TrashHandler) Restore(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	if err := h.trashService.RestoreRecord(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已恢复"})
}

func (h *TrashHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	if err := h.trashService.PermanentDelete(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已永久删除"})
}

func (h *TrashHandler) Empty(c *gin.Context) {
	userID := c.GetUint64("user_id")

	count, err := h.trashService.EmptyTrash(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已清空回收站", "data": gin.H{"deleted": count}})
}

func (h *TrashHandler) BatchTrash(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		RecordIDs []uint64 `json:"record_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	trashed, failed := h.trashService.BatchTrash(req.RecordIDs, userID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"trashed": trashed, "failed": failed}})
}

func (h *TrashHandler) BatchRestore(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		RecordIDs []uint64 `json:"record_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	restored, failed := h.trashService.BatchRestore(req.RecordIDs, userID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"restored": restored, "failed": failed}})
}
