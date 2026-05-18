package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) CreateRootDomain(c *gin.Context) {
	var req struct {
		Host         string `json:"host" binding:"required"`
		DomainSuffix string `json:"domain_suffix" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	fullDomain := req.Host + "." + req.DomainSuffix

	var existing model.DomainNode
	if err := h.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "domain already exists"})
		return
	}

	adminID := c.GetUint64("user_id")

	node := &model.DomainNode{
		Host:       req.Host,
		FullDomain: fullDomain,
		OwnerID:    adminID,
	}

	if err := h.db.Create(node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func (h *AdminHandler) AssignDomain(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid node id"})
		return
	}

	var req struct {
		UserID uint64 `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var user model.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "target user not found"})
		return
	}

	if err := h.db.Model(&model.DomainNode{}).Where("id = ?", nodeID).Update("owner_id", req.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "domain assigned successfully"})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": users})
}

func (h *AdminHandler) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userID := c.Query("user_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&model.OperationLog{})
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     logs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *AdminHandler) RetrySync(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid record id"})
		return
	}

	if err := h.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).
		Update("sync_status", "pending").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "sync retry queued"})
}
