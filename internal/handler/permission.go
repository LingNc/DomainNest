package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PermissionHandler struct {
	permissionService *service.PermissionService
	db                *gorm.DB
}

func NewPermissionHandler(permissionService *service.PermissionService, db *gorm.DB) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService, db: db}
}

func (h *PermissionHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	// Must be at least admin level to view permissions
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	perms, err := h.permissionService.ListPermissions(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}

func (h *PermissionHandler) Grant(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	var req struct {
		TargetUserID uint64   `json:"target_user_id" binding:"required"`
		Level        string   `json:"level" binding:"required"`
		AllowedTypes []string `json:"allowed_types"`
		AllowedIPs   []string `json:"allowed_ips"`
		HostPrefix   string   `json:"host_prefix"`
		MaxDepth     *int     `json:"max_depth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	allowedTypesJSON := ""
	if len(req.AllowedTypes) > 0 {
		b, _ := json.Marshal(req.AllowedTypes)
		allowedTypesJSON = string(b)
	}
	allowedIPsJSON := ""
	if len(req.AllowedIPs) > 0 {
		b, _ := json.Marshal(req.AllowedIPs)
		allowedIPsJSON = string(b)
	}

	if err := h.permissionService.Grant(req.TargetUserID, nodeID, req.Level, allowedTypesJSON, allowedIPsJSON, req.HostPrefix, req.MaxDepth, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "grant_permission", "domain_node", &nodeID,
		map[string]interface{}{"target_user": req.TargetUserID, "level": req.Level}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "permission granted"})
}

func (h *PermissionHandler) Revoke(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	// Must be at least admin level to revoke
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	if err := h.permissionService.Revoke(targetUserID, nodeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "revoke_permission", "domain_node", &nodeID,
		map[string]interface{}{"target_user": targetUserID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "permission revoked"})
}

func (h *PermissionHandler) MyPermissions(c *gin.Context) {
	userID := c.GetUint64("user_id")

	perms, err := h.permissionService.GetUserPermissions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}

func (h *PermissionHandler) RevokeRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	// Must be at least admin level to request revoke
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	if err := h.permissionService.RevokeRequest(targetUserID, nodeID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "revoke_request", "domain_node", &nodeID,
		map[string]interface{}{"target_user": targetUserID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "revoke request sent"})
}

func (h *PermissionHandler) AcceptReturn(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	// Only the target user (whose permission is being revoked) can accept
	if userID != targetUserID {
		// Or an admin/owner can force accept
		if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only the permission holder or admin can accept"})
			return
		}
	}

	var req struct {
		Action       string  `json:"action" binding:"required"` // keep/delete/transfer
		TargetUserID *uint64 `json:"target_user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.permissionService.AcceptReturn(targetUserID, nodeID, req.Action, req.TargetUserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "accept_return", "domain_node", &nodeID,
		map[string]interface{}{"target_user": targetUserID, "action": req.Action}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "permission return accepted"})
}

func (h *PermissionHandler) RejectReturn(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	// Only the target user can reject
	if userID != targetUserID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only the permission holder can reject"})
		return
	}

	if err := h.permissionService.RejectReturn(targetUserID, nodeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "reject_return", "domain_node", &nodeID,
		map[string]interface{}{"target_user": targetUserID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "return request rejected"})
}

func (h *PermissionHandler) GetPendingRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	// Must be at least admin level
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	records, err := h.permissionService.GetPendingRecords(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

func (h *PermissionHandler) AssignPendingRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	// Must be at least admin level
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	var req struct {
		RecordIDs []uint64 `json:"record_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.permissionService.AssignPendingRecords(req.RecordIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "assign_pending_records", "domain_node", &nodeID,
		map[string]interface{}{"record_ids": req.RecordIDs}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "records assigned"})
}

func (h *PermissionHandler) DeletePendingRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	// Must be at least admin level
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	var req struct {
		RecordIDs []uint64 `json:"record_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.permissionService.DeletePendingRecords(req.RecordIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_pending_records", "domain_node", &nodeID,
		map[string]interface{}{"record_ids": req.RecordIDs}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "records deleted"})
}

func (h *PermissionHandler) GetPendingReturns(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var perms []model.DomainPermission
	if err := h.db.Preload("DomainNode").Preload("Creator").
		Where("user_id = ? AND status = ?", userID, "pending_return").Find(&perms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}
