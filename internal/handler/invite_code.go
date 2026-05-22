package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type InviteCodeHandler struct {
	inviteCodeService *service.InviteCodeService
}

func NewInviteCodeHandler(inviteCodeService *service.InviteCodeService) *InviteCodeHandler {
	return &InviteCodeHandler{inviteCodeService: inviteCodeService}
}

// GenerateInviteCodes generates invite codes for the authenticated user.
// POST /api/v1/auth/invite-codes
func (h *InviteCodeHandler) Generate(c *gin.Context) {
	var req struct {
		Count int `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	userID := c.GetUint64("user_id")
	codes, err := h.inviteCodeService.GenerateCodes(userID, req.Count)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": codes})
}

// ListMyInviteCodes lists invite codes created by the authenticated user.
// GET /api/v1/auth/invite-codes
func (h *InviteCodeHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	codes, total, err := h.inviteCodeService.ListUserCodes(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     codes,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// DeleteInviteCode deletes an unused invite code and restores quota.
// DELETE /api/v1/auth/invite-codes/:id
func (h *InviteCodeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的ID"})
		return
	}
	userID := c.GetUint64("user_id")
	if err := h.inviteCodeService.DeleteCode(id, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "邀请码已删除"})
}

// BatchDeleteInviteCodes deletes multiple unused invite codes and restores quota.
// POST /api/v1/auth/invite-codes/batch-delete
func (h *InviteCodeHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	userID := c.GetUint64("user_id")
	deleted, err := h.inviteCodeService.BatchDeleteCodes(req.IDs, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除 " + strconv.Itoa(deleted) + " 个邀请码", "data": gin.H{"deleted": deleted}})
}
