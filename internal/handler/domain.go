package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DomainHandler struct {
	domainService *service.DomainService
	db            *gorm.DB
}

func NewDomainHandler(domainService *service.DomainService, db *gorm.DB) *DomainHandler {
	return &DomainHandler{domainService: domainService, db: db}
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
		"message": "success",
		"data":    node,
	})
}

func (h *DomainHandler) Get(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
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

	middleware.LogOperation(h.db, userID, "transfer_domain", "domain_node", &nodeID,
		map[string]interface{}{"target_user_id": req.TargetUserID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "transfer successful"})
}

func (h *DomainHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	if err := h.domainService.DeleteNode(nodeID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_domain", "domain_node", &nodeID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "deleted successfully"})
}
