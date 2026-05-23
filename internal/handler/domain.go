package handler

import (
	"fmt"
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

type DomainHandler struct {
	domainService *service.DomainService
	permService   *service.PermissionService
	notifSvc      *notification.Service
	db            *gorm.DB
}

func NewDomainHandler(domainService *service.DomainService, permService *service.PermissionService, notifSvc *notification.Service, db *gorm.DB) *DomainHandler {
	return &DomainHandler{domainService: domainService, permService: permService, notifSvc: notifSvc, db: db}
}

func (h *DomainHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")

	nodes, err := h.domainService.GetUserNodes(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

func (h *DomainHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		ParentID uint64 `json:"parent_id" binding:"required"`
		Host     string `json:"host" binding:"required,min=1,max=64"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	node, err := h.domainService.CreateNode(req.ParentID, req.Host, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "create_domain", "domain_node", &node.ID,
		map[string]interface{}{"full_domain": node.FullDomain}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data":    node,
	})
}

func (h *DomainHandler) Get(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	node, err := h.domainService.GetNode(nodeID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *DomainHandler) Transfer(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var req struct {
		TargetUserID uint64 `json:"target_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	transferResult, err := h.domainService.TransferNode(nodeID, userID, req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, userID, req.TargetUserID, "transfer_domain", "domain_node", &nodeID,
		map[string]interface{}{}, c.ClientIP())

	ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "transfer",
		"node_id": nodeID,
	})
	ws.BroadcastToUser(req.TargetUserID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "transfer_received",
		"node_id": nodeID,
	})

	// Notify both parties
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error != nil {
			return
		}
		fromUsername := c.GetString("username")
		var targetUser model.User
		if h.db.First(&targetUser, req.TargetUserID).Error == nil {
			if err := h.notifSvc.Send(req.TargetUserID, notification.DomainTransferredTo(&node, fromUsername)); err != nil {
				log.Printf("[Notification] DomainTransferredTo failed: %v", err)
			}
			if err := h.notifSvc.Send(userID, notification.DomainTransferredAway(&node, targetUser.Username)); err != nil {
				log.Printf("[Notification] DomainTransferredAway failed: %v", err)
			}
		}

		// Notify new owner about existing delegations
		if transferResult.DelegationCount > 0 {
			if err := h.notifSvc.Send(req.TargetUserID, notification.DomainTransferredWithDelegations(&node, transferResult.DelegationCount)); err != nil {
				log.Printf("[Notification] DomainTransferredWithDelegations failed: %v", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "转移成功"})
}

func (h *DomainHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	// Load node before deletion for notification
	var node model.DomainNode
	nodeLoaded := h.db.First(&node, nodeID).Error == nil

	if err := h.domainService.DeleteNode(nodeID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_domain", "domain_node", &nodeID,
		nil, c.ClientIP())

	ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "delete",
		"node_id": nodeID,
	})

	// Notify the owner
	if nodeLoaded {
		go func() {
			defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
			if err := h.notifSvc.Send(node.OwnerID, notification.DomainDeleted(&node)); err != nil {
				log.Printf("[Notification] DomainDeleted failed: %v", err)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *DomainHandler) BatchDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req struct {
		NodeIDs []uint64 `json:"node_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	deleted, skipped, err := h.domainService.BatchDeleteNodes(req.NodeIDs, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "batch_delete_domains", "domain_node", nil,
		map[string]interface{}{"node_ids": req.NodeIDs, "deleted": deleted, "skipped": skipped}, c.ClientIP())

	msg := fmt.Sprintf("成功删除 %d 个域名", deleted)
	if skipped > 0 {
		msg += fmt.Sprintf("，%d 个跳过（有子域名或DNS记录）", skipped)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg, "data": gin.H{"deleted": deleted, "skipped": skipped}})
}

func (h *DomainHandler) GetTransferredAway(c *gin.Context) {
	userID := c.GetUint64("user_id")

	nodes, err := h.domainService.GetTransferredAwayNodes(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取转出域名失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

func (h *DomainHandler) ConvertToNode(c *gin.Context) {
	userID := c.GetUint64("user_id")

	parentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的父节点ID"})
		return
	}

	var req struct {
		Host string `json:"host" binding:"required,min=1,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Require write permission on the parent node
	if err := h.permService.RequireLevel(userID, parentID, 2); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	node, err := h.domainService.MaterializeNode(parentID, req.Host, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Count records that were linked to this node
	var affectedRecords int64
	h.db.Model(&model.DNSRecord{}).Where("own_node_id = ? AND deleted_at IS NULL", node.ID).Count(&affectedRecords)

	middleware.LogOperation(h.db, userID, "convert_to_node", "domain_node", &node.ID,
		map[string]interface{}{"full_domain": node.FullDomain, "host": req.Host}, c.ClientIP())

	ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
		"action":    "materialize",
		"node_id":   node.ID,
		"parent_id": parentID,
	})

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"node":             node,
			"affected_records": affectedRecords,
		},
	})
}

func (h *DomainHandler) DemoteNode(c *gin.Context) {
	userID := c.GetUint64("user_id")

	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	// Require owner permission (level 4) to demote a node
	if err := h.permService.RequireLevel(userID, nodeID, 4); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	if err := h.domainService.DemoteNode(nodeID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "demote_node", "domain_node", &nodeID,
		nil, c.ClientIP())

	ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "demote",
		"node_id": nodeID,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "降级成功"})
}

func (h *DomainHandler) GetConversionLogs(c *gin.Context) {
	userID := c.GetUint64("user_id")

	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	// Require read permission on the node
	if err := h.permService.RequireLevel(userID, nodeID, 1); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	logs, err := h.domainService.GetConversionLogs(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": logs})
}

func (h *DomainHandler) ReclaimDomain(c *gin.Context) {
	userID := c.GetUint64("user_id")
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的服务商ID"})
		return
	}
	domainNodeID, err := strconv.ParseUint(c.Param("did"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	// Load the node to get old owner before reclaim
	var node model.DomainNode
	if err := h.db.First(&node, domainNodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}
	oldOwnerID := node.OwnerID

	if err := h.domainService.ForceReclaim(domainNodeID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "reclaim_domain", "domain_node", &domainNodeID,
		map[string]interface{}{"provider_id": providerID, "old_owner_id": oldOwnerID}, c.ClientIP())

	ws.BroadcastToUser(oldOwnerID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "reclaimed",
		"node_id": domainNodeID,
	})
	ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "reclaim",
		"node_id": domainNodeID,
	})

	// Notify the previous owner
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		byUsername := c.GetString("username")
		if err := h.notifSvc.Send(oldOwnerID, notification.DomainReclaimed(&node, byUsername)); err != nil {
			log.Printf("[Notification] DomainReclaimed failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "回收成功"})
}

func (h *DomainHandler) ReactivateDomain(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var req struct {
		ProviderID uint64 `json:"provider_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.domainService.ReactivateNode(nodeID, req.ProviderID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "reactivate_domain", "domain_node", &nodeID,
		map[string]interface{}{"provider_id": req.ProviderID}, c.ClientIP())

	ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "reactivate",
		"node_id": nodeID,
	})

	// Notify the user
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			if err := h.notifSvc.Send(userID, notification.DomainReactivated(&node)); err != nil {
				log.Printf("[Notification] DomainReactivated failed: %v", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "重新激活成功"})
}

func (h *DomainHandler) ArchiveInfo(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	if err := h.permService.RequireLevel(userID, nodeID, 1); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}

	var node model.DomainNode
	if err := h.db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	result := gin.H{
		"status":               node.Status,
		"archived_provider_id": node.ArchivedProviderID,
	}

	if node.ArchivedProviderID != nil {
		var provider model.DNSProvider
		if err := h.db.First(&provider, *node.ArchivedProviderID).Error; err == nil {
			result["archived_provider_name"] = provider.Name
			result["archived_provider_type"] = provider.ProviderType
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *DomainHandler) ArchiveDomain(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}
	userID := c.GetUint64("user_id")

	if err := h.domainService.ArchiveDomainTree(nodeID, userID); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Get node info for notification
	var node model.DomainNode
	h.db.First(&node, nodeID)

	byUsername := c.GetString("username")

	// Notify all permission holders
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		var perms []model.DomainPermission
		h.db.Where("domain_node_id = ?", nodeID).Find(&perms)
		for _, p := range perms {
			if err := h.notifSvc.Send(p.UserID, notification.DomainArchived(&node, byUsername)); err != nil {
				log.Printf("[Notification] DomainArchived failed: %v", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "域名已归档"})
}

func (h *DomainHandler) RestoreDomain(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}
	userID := c.GetUint64("user_id")

	if err := h.domainService.RestoreDomainTree(nodeID, userID); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error()})
		return
	}

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			if err := h.notifSvc.Send(userID, notification.DomainRestored(&node)); err != nil {
				log.Printf("[Notification] DomainRestored failed: %v", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "域名已恢复"})
}

func (h *DomainHandler) GetArchivedDomains(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodes, err := h.domainService.GetArchivedDomains(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

func (h *DomainHandler) ListArchivedChildren(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")

	list, err := h.domainService.ListArchivedChildren(id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

func (h *DomainHandler) RestoreArchivedChild(c *gin.Context) {
	childID, _ := strconv.ParseUint(c.Param("childId"), 10, 64)
	userID := c.GetUint64("user_id")

	err := h.domainService.RestoreArchivedChild(childID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "恢复成功"})
}

func (h *DomainHandler) ReturnSubdomain(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}
	userID := c.GetUint64("user_id")

	// Get node before return for notification
	var node model.DomainNode
	if err := h.db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	if err := h.domainService.ReturnSubdomainToClaimer(nodeID, userID); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Notify claimer (parent owner)
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if node.ParentID != nil && *node.ParentID != 0 {
			var parent model.DomainNode
			if h.db.First(&parent, *node.ParentID).Error == nil {
				byUsername := c.GetString("username")
				if err := h.notifSvc.Send(parent.OwnerID, notification.SubdomainReturned(&node, byUsername)); err != nil {
					log.Printf("[Notification] SubdomainReturned failed: %v", err)
				}
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "子域名已归还认领人"})
}
