package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"domainnest/internal/domain/notification"
	"domainnest/internal/errs"
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	// Must be at least admin level to view permissions
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		errs.JSONError(c, err)
		return
	}

	perms, err := h.permissionService.ListPermissions(nodeID)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}

func (h *PermissionHandler) Grant(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNodeID)
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
		errs.JSONError(c, err)
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

	if err := h.permissionService.Grant(req.TargetUserID, nodeID, req.Level, allowedTypesJSON, allowedIPsJSON, req.HostPrefix, req.HostRules, req.MaxDepth, req.SourceFilter, userID); err != nil {
		errs.JSONError(c, err)
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
	if nodeLoaded {
		go func() {
			defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
			if err := h.notifSvc.Send(req.TargetUserID, notification.PermissionGranted(&node, req.Level)); err != nil {
				log.Printf("[Notification] PermissionGranted failed: %v", err)
				return
			}
			svc := service.NewMessageService(h.db)
			if count, err := svc.GetNotificationUnreadCount(req.TargetUserID); err == nil {
				ws.BroadcastToUser(req.TargetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
			ws.BroadcastToUser(req.TargetUserID, ws.TypeDomainTreeUpdate, gin.H{
				"action":  "permission_change",
				"node_id": nodeID,
			})
		}()
	}

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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
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
		errs.JSONError(c, err)
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

		err := h.permissionService.Grant(targetID, nodeID, req.Level, allowedTypesJSON, allowedIPsJSON, req.HostPrefix, req.HostRules, req.MaxDepth, req.SourceFilter, userID)
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
				if err := h.notifSvc.Send(uid, notification.PermissionGranted(&node, req.Level)); err != nil {
					log.Printf("[Notification] PermissionGranted failed: %v", err)
					return
				}
				svc := service.NewMessageService(h.db)
				if count, err := svc.GetNotificationUnreadCount(uid); err == nil {
					ws.BroadcastToUser(uid, ws.TypeUnreadUpdate, gin.H{"count": count})
				}
				ws.BroadcastToUser(uid, ws.TypeDomainTreeUpdate, gin.H{
					"action":  "permission_change",
					"node_id": nodeID,
				})
			}(targetID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results})
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
		errs.JSONError(c, err)
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

	// Pre-load nodes for notifications
	nodeMap := make(map[uint64]*model.DomainNode)
	for _, nodeID := range uniqueNodeIDs {
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			nodeMap[nodeID] = &node
		}
	}

	results := make([]BatchMultiDomainResult, 0, len(uniqueNodeIDs)*len(uniqueUserIDs))
	for _, nodeID := range uniqueNodeIDs {
		node, nodeLoaded := nodeMap[nodeID]
		domain := ""
		if nodeLoaded {
			domain = node.FullDomain
		}

		for _, targetID := range uniqueUserIDs {
			// Skip self-authorization
			if targetID == userID {
				results = append(results, BatchMultiDomainResult{DomainNodeID: nodeID, UserID: targetID, Success: false, Error: "不能给自己授权"})
				continue
			}

			err := h.permissionService.Grant(targetID, nodeID, req.Level, allowedTypesJSON, allowedIPsJSON, "", req.HostRules, req.MaxDepth, req.SourceFilter, userID)
			if err != nil {
				results = append(results, BatchMultiDomainResult{DomainNodeID: nodeID, UserID: targetID, Success: false, Error: err.Error()})
				continue
			}

			results = append(results, BatchMultiDomainResult{DomainNodeID: nodeID, UserID: targetID, Success: true})

			middleware.LogOperationUser(h.db, userID, targetID, "grant_permission", "domain_node", &nodeID,
				map[string]interface{}{"level": req.Level, "domain": domain, "batch_multi_domain": true}, c.ClientIP())

			// Notify target user
			if nodeLoaded {
				go func(uid uint64, n model.DomainNode, nid uint64) {
					defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
					if err := h.notifSvc.Send(uid, notification.PermissionGranted(&n, req.Level)); err != nil {
						log.Printf("[Notification] PermissionGranted failed: %v", err)
						return
					}
					svc := service.NewMessageService(h.db)
					if count, err := svc.GetNotificationUnreadCount(uid); err == nil {
						ws.BroadcastToUser(uid, ws.TypeUnreadUpdate, gin.H{"count": count})
					}
					ws.BroadcastToUser(uid, ws.TypeDomainTreeUpdate, gin.H{
						"action":  "permission_change",
						"node_id": nid,
					})
				}(targetID, *node, nodeID)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": results})
}

func (h *PermissionHandler) Revoke(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	// Must be at least admin level to revoke; owner (level 4) can always revoke
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		errs.JSONError(c, err)
		return
	}

	if err := h.permissionService.Revoke(targetUserID, nodeID); err != nil {
		errs.JSONError(c, err)
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
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}

func (h *PermissionHandler) RevokeRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	// Must be at least admin level to request revoke
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		errs.JSONError(c, err)
		return
	}

	if err := h.permissionService.RevokeRequest(targetUserID, nodeID, userID); err != nil {
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	// Only the target user (whose permission is being revoked) can accept
	if userID != targetUserID {
		// Or an admin/owner can force accept
		if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
			errs.JSONErrorCode(c, errs.OnlyHolderOrAdminAccept)
			return
		}
	}

	var req struct {
		Action       string  `json:"action" binding:"required"` // keep/delete/transfer
		TargetUserID *uint64 `json:"target_user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	if err := h.permissionService.AcceptReturn(targetUserID, nodeID, req.Action, req.TargetUserID); err != nil {
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	// Only the target user can reject
	if userID != targetUserID {
		errs.JSONErrorCode(c, errs.OnlyHolderReject)
		return
	}

	if err := h.permissionService.RejectReturn(targetUserID, nodeID); err != nil {
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	// Must be at least admin level
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		errs.JSONError(c, err)
		return
	}

	records, err := h.permissionService.GetPendingRecords(nodeID)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
}

func (h *PermissionHandler) AssignPendingRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	// Must be at least admin level
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		errs.JSONError(c, err)
		return
	}

	var req struct {
		RecordIDs []uint64 `json:"record_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	if err := h.permissionService.AssignPendingRecords(req.RecordIDs); err != nil {
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	// Must be at least admin level
	if err := h.permissionService.RequireLevel(userID, nodeID, 3); err != nil {
		errs.JSONError(c, err)
		return
	}

	var req struct {
		RecordIDs []uint64 `json:"record_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	if err := h.permissionService.DeletePendingRecords(req.RecordIDs); err != nil {
		errs.JSONError(c, err)
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
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": perms})
}
