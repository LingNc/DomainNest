package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"
	"domainnest/internal/ws"

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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var req struct {
		TargetUserID uint64           `json:"target_user_id" binding:"required"`
		Level        string           `json:"level" binding:"required"`
		AllowedTypes []string         `json:"allowed_types"`
		AllowedIPs   []string         `json:"allowed_ips"`
		HostPrefix   string           `json:"host_prefix"`
		HostRules    []model.HostRule `json:"host_rules"`
		MaxDepth     *int             `json:"max_depth"`
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

	if err := h.permissionService.Grant(req.TargetUserID, nodeID, req.Level, allowedTypesJSON, allowedIPsJSON, req.HostPrefix, req.HostRules, req.MaxDepth, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	// Load node for notification
	var node model.DomainNode
	nodeLoaded := h.db.First(&node, nodeID).Error == nil

	domain := ""
	if nodeLoaded {
		domain = node.FullDomain
	}

	middleware.LogOperationUser(h.db, userID, req.TargetUserID, "grant_permission", "domain_node", &nodeID,
		map[string]interface{}{"level": req.Level, "domain": domain}, c.ClientIP())

	// Notify target user
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		if nodeLoaded {
			svc := service.NewMessageService(h.db)
			svc.SendSystemNotification(req.TargetUserID, "权限授予",
				fmt.Sprintf("你已被授予 %s 域名的 %s 权限", node.FullDomain, req.Level),
				"permission_grant", fmt.Sprintf("{\"domain_node_id\":%d,\"level\":\"%s\"}", nodeID, req.Level))
			ws.BroadcastToUser(req.TargetUserID, ws.TypeNewNotification, gin.H{"type": "permission_change"})
			if count, err := svc.GetNotificationUnreadCount(req.TargetUserID); err == nil {
				ws.BroadcastToUser(req.TargetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限已授予"})
}

// BatchGrantResult is the per-user result of a batch grant operation.
type BatchGrantResult struct {
	UserID  uint64 `json:"user_id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *PermissionHandler) BatchGrant(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var req struct {
		TargetUserIDs []uint64         `json:"target_user_ids" binding:"required,min=1,max=50"`
		Level         string           `json:"level" binding:"required"`
		AllowedTypes  []string         `json:"allowed_types"`
		AllowedIPs    []string         `json:"allowed_ips"`
		HostPrefix    string           `json:"host_prefix"`
		HostRules     []model.HostRule `json:"host_rules"`
		MaxDepth      *int             `json:"max_depth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Deduplicate user IDs
	seen := make(map[uint64]bool)
	uniqueIDs := make([]uint64, 0, len(req.TargetUserIDs))
	for _, id := range req.TargetUserIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
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

	// Load node for notifications
	var node model.DomainNode
	nodeLoaded := h.db.First(&node, nodeID).Error == nil
	domain := ""
	if nodeLoaded {
		domain = node.FullDomain
	}

	results := make([]BatchGrantResult, len(uniqueIDs))
	for i, targetID := range uniqueIDs {
		// Skip self-authorization
		if targetID == userID {
			results[i] = BatchGrantResult{UserID: targetID, Success: false, Error: "不能给自己授权"}
			continue
		}

		err := h.permissionService.Grant(targetID, nodeID, req.Level, allowedTypesJSON, allowedIPsJSON, req.HostPrefix, req.HostRules, req.MaxDepth, userID)
		if err != nil {
			results[i] = BatchGrantResult{UserID: targetID, Success: false, Error: err.Error()}
			continue
		}

		results[i] = BatchGrantResult{UserID: targetID, Success: true}

		middleware.LogOperationUser(h.db, userID, targetID, "grant_permission", "domain_node", &nodeID,
			map[string]interface{}{"level": req.Level, "domain": domain, "batch": true}, c.ClientIP())

		// Notify target user
		if nodeLoaded {
			go func(uid uint64) {
				defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
				svc := service.NewMessageService(h.db)
				svc.SendSystemNotification(uid, "权限授予",
					fmt.Sprintf("你已被授予 %s 域名的 %s 权限", node.FullDomain, req.Level),
					"permission_grant", fmt.Sprintf("{\"domain_node_id\":%d,\"level\":\"%s\"}", nodeID, req.Level))
				ws.BroadcastToUser(uid, ws.TypeNewNotification, gin.H{"type": "permission_change"})
				if count, err := svc.GetNotificationUnreadCount(uid); err == nil {
					ws.BroadcastToUser(uid, ws.TypeUnreadUpdate, gin.H{"count": count})
				}
			}(targetID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results})
}

func (h *PermissionHandler) Revoke(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
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

	middleware.LogOperationUser(h.db, userID, targetUserID, "revoke_permission", "domain_node", &nodeID,
		map[string]interface{}{}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			svc := service.NewMessageService(h.db)
			svc.SendSystemNotification(targetUserID, "权限撤销",
				fmt.Sprintf("你在 %s 域名的权限已被撤销", node.FullDomain), "", "")
			ws.BroadcastToUser(targetUserID, ws.TypeNewNotification, gin.H{"type": "permission_change"})
			if count, err := svc.GetNotificationUnreadCount(targetUserID); err == nil {
				ws.BroadcastToUser(targetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限已撤销"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
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

	middleware.LogOperationUser(h.db, userID, targetUserID, "revoke_request", "domain_node", &nodeID,
		map[string]interface{}{}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			svc := service.NewMessageService(h.db)
			svc.SendSystemNotification(targetUserID, "权限归还请求",
				fmt.Sprintf("管理员请求归还你在 %s 域名的权限", node.FullDomain), "", "")
			ws.BroadcastToUser(targetUserID, ws.TypeNewNotification, gin.H{"type": "permission_change"})
			if count, err := svc.GetNotificationUnreadCount(targetUserID); err == nil {
				ws.BroadcastToUser(targetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "撤销请求已发送"})
}

func (h *PermissionHandler) AcceptReturn(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	// Only the target user (whose permission is being revoked) can accept
	if userID != targetUserID {
		// Or an admin/owner can force accept
		if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅权限持有者或管理员可接受"})
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

	middleware.LogOperationUser(h.db, userID, targetUserID, "accept_return", "domain_node", &nodeID,
		map[string]interface{}{"action": req.Action}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限归还已接受"})
}

func (h *PermissionHandler) RejectReturn(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	// Only the target user can reject
	if userID != targetUserID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅权限持有者可拒绝"})
		return
	}

	if err := h.permissionService.RejectReturn(targetUserID, nodeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, userID, targetUserID, "reject_return", "domain_node", &nodeID,
		map[string]interface{}{}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "归还请求已拒绝"})
}

func (h *PermissionHandler) GetPendingRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录已分配"})
}

func (h *PermissionHandler) DeletePendingRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录已删除"})
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
