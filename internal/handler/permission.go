package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"domainnest/internal/domain/notification"
	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"
	"domainnest/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PermissionHandler struct {
	permissionService *service.PermissionService
	notifSvc          *notification.Service
	db                *gorm.DB
}

func NewPermissionHandler(permissionService *service.PermissionService, notifSvc *notification.Service, db *gorm.DB) *PermissionHandler {
	return &PermissionHandler{permissionService: permissionService, notifSvc: notifSvc, db: db}
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
		SourceFilter *string          `json:"source_filter"`
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
	hostRulesJSON := ""
	if len(req.HostRules) > 0 {
		b, _ := json.Marshal(req.HostRules)
		hostRulesJSON = string(b)
	}

	// Load node for notification and ActionData
	var node model.DomainNode
	nodeLoaded := h.db.First(&node, nodeID).Error == nil

	domain := ""
	if nodeLoaded {
		domain = node.FullDomain
	}

	middleware.LogOperationUser(h.db, userID, req.TargetUserID, "grant_permission_pending", "domain_node", &nodeID,
		map[string]interface{}{"level": req.Level, "domain": domain}, c.ClientIP())

	// Send pending notification instead of executing immediately
	if nodeLoaded {
		pendingData := map[string]interface{}{
			"target_user_id": req.TargetUserID,
			"level":          req.Level,
			"allowed_types": allowedTypesJSON,
			"allowed_ips":    allowedIPsJSON,
			"host_prefix":    req.HostPrefix,
			"host_rules":     hostRulesJSON,
			"max_depth":      req.MaxDepth,
			"source_filter":  req.SourceFilter,
			"created_by":     userID,
		}
		pendingDataJSON, _ := json.Marshal(pendingData)
		go func() {
			defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
			if err := h.notifSvc.Send(req.TargetUserID, notification.PendingPermissionGrant(&node, req.Level, string(pendingDataJSON))); err != nil {
				log.Printf("[Notification] PendingPermissionGrant failed: %v", err)
				return
			}
			svc := service.NewMessageService(h.db)
			if count, err := svc.GetNotificationUnreadCount(req.TargetUserID); err == nil {
				ws.BroadcastToUser(req.TargetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限授予请求已发送，等待对方接受"})
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
		SourceFilter  *string          `json:"source_filter"`
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
	hostRulesJSON := ""
	if len(req.HostRules) > 0 {
		b, _ := json.Marshal(req.HostRules)
		hostRulesJSON = string(b)
	}

	// Load node for notifications
	var node model.DomainNode
	nodeLoaded := h.db.First(&node, nodeID).Error == nil
	domain := ""
	if nodeLoaded {
		domain = node.FullDomain
	}

	middleware.LogOperationUser(h.db, userID, 0, "grant_permission_pending", "domain_node", &nodeID,
		map[string]interface{}{"level": req.Level, "domain": domain, "batch": true, "count": len(uniqueIDs)}, c.ClientIP())

	results := make([]BatchGrantResult, len(uniqueIDs))
	for i, targetID := range uniqueIDs {
		// Skip self-authorization
		if targetID == userID {
			results[i] = BatchGrantResult{UserID: targetID, Success: false, Error: "不能给自己授权"}
			continue
		}

		// Send pending notification instead of executing immediately
		if nodeLoaded {
			pendingData := map[string]interface{}{
				"target_user_id": targetID,
				"level":          req.Level,
				"allowed_types":  allowedTypesJSON,
				"allowed_ips":    allowedIPsJSON,
				"host_prefix":    req.HostPrefix,
				"host_rules":     hostRulesJSON,
				"max_depth":      req.MaxDepth,
				"source_filter":  req.SourceFilter,
				"created_by":     userID,
			}
			pendingDataJSON, _ := json.Marshal(pendingData)
			go func(uid uint64) {
				defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
				if err := h.notifSvc.Send(uid, notification.PendingPermissionGrant(&node, req.Level, string(pendingDataJSON))); err != nil {
					log.Printf("[Notification] PendingPermissionGrant failed: %v", err)
					return
				}
				svc := service.NewMessageService(h.db)
				if count, err := svc.GetNotificationUnreadCount(uid); err == nil {
					ws.BroadcastToUser(uid, ws.TypeUnreadUpdate, gin.H{"count": count})
				}
			}(targetID)
		}

		results[i] = BatchGrantResult{UserID: targetID, Success: true}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results, "message": "权限授予请求已发送，等待对方接受"})
}

// BatchMultiDomainResult is the per-domain-user result of a batch multi-domain grant operation.
type BatchMultiDomainResult struct {
	DomainNodeID uint64 `json:"domain_node_id"`
	UserID       uint64 `json:"user_id"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

func (h *PermissionHandler) BatchGrantMultiDomain(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		DomainNodeIDs []uint64         `json:"domain_node_ids" binding:"required,min=1,max=20"`
		TargetUserIDs []uint64         `json:"target_user_ids" binding:"required,min=1,max=50"`
		Level         string           `json:"level" binding:"required"`
		AllowedTypes  []string         `json:"allowed_types"`
		AllowedIPs    []string         `json:"allowed_ips"`
		HostRules     []model.HostRule `json:"host_rules"`
		MaxDepth      *int             `json:"max_depth"`
		SourceFilter  *string          `json:"source_filter"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Deduplicate domain node IDs
	seenNodes := make(map[uint64]bool)
	uniqueNodeIDs := make([]uint64, 0, len(req.DomainNodeIDs))
	for _, id := range req.DomainNodeIDs {
		if !seenNodes[id] {
			seenNodes[id] = true
			uniqueNodeIDs = append(uniqueNodeIDs, id)
		}
	}

	// Deduplicate user IDs
	seenUsers := make(map[uint64]bool)
	uniqueUserIDs := make([]uint64, 0, len(req.TargetUserIDs))
	for _, id := range req.TargetUserIDs {
		if !seenUsers[id] {
			seenUsers[id] = true
			uniqueUserIDs = append(uniqueUserIDs, id)
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
	hostRulesJSON := ""
	if len(req.HostRules) > 0 {
		b, _ := json.Marshal(req.HostRules)
		hostRulesJSON = string(b)
	}

	// Pre-load nodes for notifications
	nodeMap := make(map[uint64]*model.DomainNode)
	for _, nodeID := range uniqueNodeIDs {
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			nodeMap[nodeID] = &node
		}
	}

	middleware.LogOperationUser(h.db, userID, 0, "grant_permission_pending", "domain_node", nil,
		map[string]interface{}{"level": req.Level, "batch_multi_domain": true, "node_count": len(uniqueNodeIDs), "user_count": len(uniqueUserIDs)}, c.ClientIP())

	results := make([]BatchMultiDomainResult, 0, len(uniqueNodeIDs)*len(uniqueUserIDs))
	for _, nodeID := range uniqueNodeIDs {
		node, nodeLoaded := nodeMap[nodeID]

		for _, targetID := range uniqueUserIDs {
			// Skip self-authorization
			if targetID == userID {
				results = append(results, BatchMultiDomainResult{DomainNodeID: nodeID, UserID: targetID, Success: false, Error: "不能给自己授权"})
				continue
			}

			// Send pending notification instead of executing immediately
			if nodeLoaded {
				pendingData := map[string]interface{}{
					"target_user_id": targetID,
					"level":          req.Level,
					"allowed_types":  allowedTypesJSON,
					"allowed_ips":    allowedIPsJSON,
					"host_prefix":    "",
					"host_rules":     hostRulesJSON,
					"max_depth":      req.MaxDepth,
					"source_filter":  req.SourceFilter,
					"created_by":     userID,
				}
				pendingDataJSON, _ := json.Marshal(pendingData)
				go func(uid uint64, n model.DomainNode, nid uint64) {
					defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
					if err := h.notifSvc.Send(uid, notification.PendingPermissionGrant(&n, req.Level, string(pendingDataJSON))); err != nil {
						log.Printf("[Notification] PendingPermissionGrant failed: %v", err)
						return
					}
					svc := service.NewMessageService(h.db)
					if count, err := svc.GetNotificationUnreadCount(uid); err == nil {
						ws.BroadcastToUser(uid, ws.TypeUnreadUpdate, gin.H{"count": count})
					}
				}(targetID, *node, nodeID)
			}

			results = append(results, BatchMultiDomainResult{DomainNodeID: nodeID, UserID: targetID, Success: true})
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results, "message": "权限授予请求已发送，等待对方接受"})
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
			if err := h.notifSvc.Send(targetUserID, notification.PermissionRevoked(&node)); err != nil {
				log.Printf("[Notification] PermissionRevoked failed: %v", err)
				return
			}
			svc := service.NewMessageService(h.db)
			if count, err := svc.GetNotificationUnreadCount(targetUserID); err == nil {
				ws.BroadcastToUser(targetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
			ws.BroadcastToUser(targetUserID, ws.TypeDomainTreeUpdate, gin.H{
				"action":  "permission_change",
				"node_id": nodeID,
			})
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
			if err := h.notifSvc.Send(targetUserID, notification.PermissionRevokeRequest(&node)); err != nil {
				log.Printf("[Notification] PermissionRevokeRequest failed: %v", err)
				return
			}
			svc := service.NewMessageService(h.db)
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

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			if err := h.notifSvc.Send(targetUserID, notification.PermissionReturned(&node)); err != nil {
				log.Printf("[Notification] PermissionReturned failed: %v", err)
				return
			}
			svc := service.NewMessageService(h.db)
			if count, err := svc.GetNotificationUnreadCount(targetUserID); err == nil {
				ws.BroadcastToUser(targetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
			ws.BroadcastToUser(targetUserID, ws.TypeDomainTreeUpdate, gin.H{
				"action":  "permission_change",
				"node_id": nodeID,
			})
		}
	}()

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

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			var user model.User
			if h.db.First(&user, targetUserID).Error == nil {
				if err := h.notifSvc.Send(node.OwnerID, notification.PermissionReturnRejected(&node, user.Username)); err != nil {
					log.Printf("[Notification] PermissionReturnRejected failed: %v", err)
					return
				}
				svc := service.NewMessageService(h.db)
				if count, err := svc.GetNotificationUnreadCount(node.OwnerID); err == nil {
					ws.BroadcastToUser(node.OwnerID, ws.TypeUnreadUpdate, gin.H{"count": count})
				}
			}
		}
	}()

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
