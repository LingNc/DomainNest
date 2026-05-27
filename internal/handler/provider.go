package handler

import (
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

type ProviderHandler struct {
	providerService *service.ProviderService
	notifSvc       *notification.Service
	messageService *service.MessageService
	db             *gorm.DB
}

func NewProviderHandler(providerService *service.ProviderService, notifSvc *notification.Service, messageService *service.MessageService, db *gorm.DB) *ProviderHandler {
	return &ProviderHandler{providerService: providerService, notifSvc: notifSvc, messageService: messageService, db: db}
}

func (h *ProviderHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req struct {
		ProviderType    string `json:"provider_type" binding:"required"`
		Name            string `json:"name" binding:"required"`
		AccessKeyID     string `json:"access_key_id" binding:"required"`
		AccessKeySecret string `json:"access_key_secret" binding:"required"`
		Endpoint        string `json:"endpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	provider, err := h.providerService.Create(userID, req.ProviderType, req.Name, req.AccessKeyID, req.AccessKeySecret, req.Endpoint)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	middleware.LogOperation(h.db, userID, "create_provider", "dns_provider", &provider.ID,
		map[string]interface{}{"name": provider.Name, "type": provider.ProviderType}, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": provider})
}

func (h *ProviderHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	providers, err := h.providerService.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": providers})
}

func (h *ProviderHandler) Get(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的ID"})
		return
	}
	provider, err := h.providerService.Get(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": provider})
}

func (h *ProviderHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的ID"})
		return
	}
	var req struct {
		Name     string `json:"name"`
		Endpoint string `json:"endpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.providerService.Update(id, userID, req.Name, req.Endpoint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "update_provider", "dns_provider", &id,
		map[string]interface{}{"name": req.Name, "endpoint": req.Endpoint}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已更新"})
}

func (h *ProviderHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的ID"})
		return
	}

	// Load provider name before deletion for notification
	var providerName string
	var provider model.DNSProvider
	if h.db.First(&provider, id).Error == nil {
		providerName = provider.Name
	}

	confirm := c.Query("confirm") == "true"
	count, err := h.providerService.Delete(id, userID, confirm)
	if err != nil {
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"code": 409, "message": err.Error(), "data": gin.H{"linked_domains": count}})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error()})
		return
	}
	middleware.LogOperation(h.db, userID, "delete_provider", "dns_provider", &id, nil, c.ClientIP())

	// Notify the user
	if providerName != "" {
		go func() {
			defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
			if err := h.notifSvc.Send(userID, notification.ProviderDeleted(providerName)); err != nil {
				log.Printf("[Notification] ProviderDeleted failed: %v", err)
			}
			if count, err := h.messageService.UnreadCount(userID); err == nil {
				ws.BroadcastToUser(userID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已删除"})
}

func (h *ProviderHandler) ListDomains(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的ID"})
		return
	}
	domains, err := h.providerService.ListDomainsWithStatus(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": domains})
}

func (h *ProviderHandler) ClaimDomain(c *gin.Context) {
	userID := c.GetUint64("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的ID"})
		return
	}
	var req struct {
		DomainName string `json:"domain_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	node, err := h.providerService.ClaimDomain(userID, id, req.DomainName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	middleware.LogOperation(h.db, userID, "claim_domain", "domain_node", &node.ID,
		map[string]interface{}{"domain": node.FullDomain, "provider_id": id}, c.ClientIP())

	// Notify the user
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if err := h.notifSvc.Send(userID, notification.DomainClaimed(node)); err != nil {
			log.Printf("[Notification] DomainClaimed failed: %v", err)
		}
		if count, err := h.messageService.UnreadCount(userID); err == nil {
			ws.BroadcastToUser(userID, ws.TypeUnreadUpdate, gin.H{"count": count})
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}
