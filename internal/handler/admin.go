package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"domainnest/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
		ProviderID uint64 `json:"provider_id" binding:"required"`
		DomainName string `json:"domain_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	adminID := c.GetUint64("user_id")

	// Verify provider exists
	var provider model.DNSProvider
	if err := h.db.First(&provider, req.ProviderID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "provider not found"})
		return
	}

	var existing model.DomainNode
	if err := h.db.Where("full_domain = ?", req.DomainName).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "domain already exists"})
		return
	}

	host := extractHostFromDomain(req.DomainName)
	node := &model.DomainNode{
		Host:       host,
		FullDomain: req.DomainName,
		OwnerID:    adminID,
		ProviderID: &req.ProviderID,
	}
	if err := h.db.Create(node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": node})
}

func extractHostFromDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return domain
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

func (h *AdminHandler) ListDomains(c *gin.Context) {
	var nodes []model.DomainNode
	if err := h.db.Preload("Owner").Order("id ASC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
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
	// Enhanced filters
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if targetType := c.Query("target_type"); targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if startTime := c.Query("start_time"); startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	query.Preload("User").Order("created_at DESC").
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

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	var req struct {
		Username    string `json:"username"`
		Role        string `json:"role"`
		Status      *int   `json:"status"`
		Nickname    string `json:"nickname"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		InviteLimit *int   `json:"invite_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Handle username change separately (needs uniqueness check)
	if req.Username != "" {
		var existing model.User
		if err := h.db.Where("username = ? AND id != ?", req.Username, userID).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "username already taken"})
			return
		}
		if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("username", req.Username).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	callerID := c.GetUint64("user_id")
	var caller model.User
	h.db.First(&caller, callerID)

	updates := map[string]interface{}{}
	if req.Role != "" {
		if !caller.IsSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only super_admin can change user roles"})
			return
		}
		updates["role"] = req.Role
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.InviteLimit != nil {
		if *req.InviteLimit < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invite_limit cannot be negative"})
			return
		}
		// Cannot decrease below current invite_count (skip for superadmin)
		var targetUser model.User
		if err := h.db.First(&targetUser, userID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user not found"})
			return
		}
		if !caller.IsSuperAdmin && *req.InviteLimit < targetUser.InviteCount {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("invite_limit cannot be less than current invite_count (%d)", targetUser.InviteCount)})
			return
		}

		// Pool model: if admin is not super_admin, deduct from admin's available pool
		if !caller.IsSuperAdmin {
			additionalAmount := *req.InviteLimit - targetUser.InviteLimit
			if additionalAmount > 0 {
				available := caller.InviteLimit - caller.InviteCount
				if available < additionalAmount {
					c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("insufficient invite quota in your pool (available: %d, requested: %d)", available, additionalAmount)})
					return
				}
				// Deduct from admin's pool
				if err := h.db.Model(&caller).UpdateColumn("invite_count", gorm.Expr("invite_count + ?", additionalAmount)).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
					return
				}
			} else if additionalAmount < 0 {
				// Return quota to admin's pool
				if err := h.db.Model(&caller).UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count + ?, 0)", additionalAmount)).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
					return
				}
			}
		}

		updates["invite_limit"] = *req.InviteLimit
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "no fields to update"})
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "user updated"})
}

func (h *AdminHandler) AdminResetPassword(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	var targetUser model.User
	if err := h.db.First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user not found"})
		return
	}
	if targetUser.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "cannot reset super admin password"})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to hash password"})
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "password reset successfully"})
}

func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	tx := h.db.Begin()
	defer tx.Rollback()

	// Handle invite pool cleanup: free the registration slot from the inviter
	var user model.User
	if err := tx.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user not found"})
		return
	}

	if user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "cannot disable super admin account"})
		return
	}

	if user.InvitedBy != nil {
		if err := tx.Model(&model.User{}).Where("id = ?", *user.InvitedBy).
			UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - 1, 0)")).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}

		// Reclaim unused invite quota back to the inviter
		unusedQuota := user.InviteLimit - user.InviteCount
		if unusedQuota > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", *user.InvitedBy).
				UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", unusedQuota)).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
				return
			}
		}
	} else {
		// No inviter — just zero out the orphaned quota
		if err := tx.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("invite_limit", 0).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
			return
		}
	}

	if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("status", 0).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "user disabled"})
}

func (h *AdminHandler) PromoteToAdmin(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	// Check caller is super_admin
	callerID := c.GetUint64("user_id")
	var caller model.User
	if err := h.db.First(&caller, callerID).Error; err != nil || !caller.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only super_admin can promote users"})
		return
	}

	var target model.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user not found"})
		return
	}

	if target.Role == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user is already admin"})
		return
	}

	if err := h.db.Model(&target).Update("role", "admin").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "user promoted to admin"})
}

func (h *AdminHandler) DemoteFromAdmin(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	callerID := c.GetUint64("user_id")
	var caller model.User
	if err := h.db.First(&caller, callerID).Error; err != nil || !caller.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "only super_admin can demote admins"})
		return
	}

	var target model.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user not found"})
		return
	}

	if target.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "cannot demote super_admin"})
		return
	}

	if target.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user is not admin"})
		return
	}

	if err := h.db.Model(&target).Update("role", "user").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "admin demoted to user"})
}
