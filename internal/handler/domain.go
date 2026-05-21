package handler

import (
	"net/http"
	"strconv"

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
	db            *gorm.DB
}

func NewDomainHandler(domainService *service.DomainService, permService *service.PermissionService, db *gorm.DB) *DomainHandler {
	return &DomainHandler{domainService: domainService, permService: permService, db: db}
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

	if err := h.domainService.TransferNode(nodeID, userID, req.TargetUserID); err != nil {
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "转移成功"})
}

func (h *DomainHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
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

	// Require owner permission (level 4) or super admin (level 5)
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
