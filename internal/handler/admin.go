package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"
	"domainnest/internal/ws"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db            *gorm.DB
	domainService *service.DomainService
}

func NewAdminHandler(db *gorm.DB, domainService *service.DomainService) *AdminHandler {
	return &AdminHandler{db: db, domainService: domainService}
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "DNS服务商不存在"})
		return
	}

	var existing model.DomainNode
	if err := h.db.Where("full_domain = ?", req.DomainName).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "域名已存在"})
		return
	}

	host := extractHostFromDomain(req.DomainName)
	// Clean up soft-deleted row that still occupies the unique index
	h.db.Unscoped().Where("full_domain = ?", req.DomainName).Delete(&model.DomainNode{})
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

	callerID := c.GetUint64("user_id")
	middleware.LogOperation(h.db, callerID, "create_root_domain", "domain_node", &node.ID,
		map[string]interface{}{"domain": node.FullDomain}, c.ClientIP())

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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不存在"})
		return
	}

	var node model.DomainNode
	if err := h.db.First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "域名不存在"})
		return
	}
	oldOwnerID := node.OwnerID

	if err := h.domainService.AdminTransferNode(nodeID, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, req.UserID, "assign_domain", "domain_node", &nodeID,
		map[string]interface{}{"old_owner_id": oldOwnerID, "new_owner_id": req.UserID}, c.ClientIP())

	ws.BroadcastToUser(oldOwnerID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "transfer",
		"node_id": nodeID,
	})
	ws.BroadcastToUser(req.UserID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "transfer_received",
		"node_id": nodeID,
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "域名分配成功"})
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
	query = applyLogFilters(query, c)

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	query.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "username", "nickname")
	}).Preload("TargetUser", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "username", "nickname")
	}).Order("created_at DESC").
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	if err := h.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).
		Updates(map[string]interface{}{
			"sync_status":     "pending",
			"sync_attempts":   0,
			"next_sync_at":    nil,
			"last_sync_error": "",
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已加入同步重试队列"})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
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
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已被占用"})
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
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅超级管理员可修改用户角色"})
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
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "邀请额度上限不能为负数"})
			return
		}
		// Cannot decrease below current invite_count (skip for superadmin)
		var targetUser model.User
		if err := h.db.First(&targetUser, userID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
			return
		}
		if !caller.IsSuperAdmin && *req.InviteLimit < targetUser.InviteCount {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("邀请额度上限不能低于已使用数量 (%d)", targetUser.InviteCount)})
			return
		}

		// Pool model: if admin is not super_admin, deduct from admin's available pool
		if !caller.IsSuperAdmin {
			additionalAmount := *req.InviteLimit - targetUser.InviteLimit
			if additionalAmount > 0 {
				available := caller.InviteLimit - caller.InviteCount
				if available < additionalAmount {
					c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("您的邀请额度池不足（可用: %d，需要: %d)", available, additionalAmount)})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有要更新的字段"})
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	callerID2 := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID2, userID, "update_user", "user", &userID, updates, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户信息已更新"})
}

func (h *AdminHandler) AdminResetPassword(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	var targetUser model.User
	if err := h.db.First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}
	callerID := c.GetUint64("user_id")
	if targetUser.IsSuperAdmin && callerID != targetUser.ID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "不可重置超级管理员密码"})
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": friendlyValidationError(err)})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, callerID, userID, "admin_reset_password", "user", &userID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "密码重置成功"})
}

func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	tx := h.db.Begin()
	defer tx.Rollback()

	// Handle invite pool cleanup: free the registration slot from the inviter
	var user model.User
	if err := tx.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}

	if user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "不可禁用超级管理员账号"})
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

		// Log the reclaim
		tx.Create(&model.InviteLog{
			InviterID: *user.InvitedBy,
			InviteeID: userID,
			Action:    "revoke",
			Amount:    unusedQuota + 1,
		})
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

	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, userID, "disable_user", "user", &userID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户已禁用"})
}

func (h *AdminHandler) PromoteToAdmin(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	// Check caller is super_admin
	callerID := c.GetUint64("user_id")
	var caller model.User
	if err := h.db.First(&caller, callerID).Error; err != nil || !caller.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅超级管理员可提升用户"})
		return
	}

	var target model.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}

	if target.Role == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该用户已是管理员"})
		return
	}

	if err := h.db.Model(&target).Update("role", "admin").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, callerID, targetID, "promote_to_admin", "user", &targetID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户已提升为管理员"})
}

func (h *AdminHandler) DemoteFromAdmin(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	callerID := c.GetUint64("user_id")
	var caller model.User
	if err := h.db.First(&caller, callerID).Error; err != nil || !caller.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "仅超级管理员可降级管理员"})
		return
	}

	var target model.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户不存在"})
		return
	}

	if target.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "不可降级超级管理员"})
		return
	}

	if target.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "该用户不是管理员"})
		return
	}

	if err := h.db.Model(&target).Update("role", "user").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, callerID, targetID, "demote_from_admin", "user", &targetID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "管理员已降级为普通用户"})
}

func (h *AdminHandler) GetDomainTree(c *gin.Context) {
	var nodes []model.DomainNode
	if err := h.db.Preload("Owner").Order("id ASC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// Count permissions per node
	nodeIDs := make([]uint64, len(nodes))
	for i, n := range nodes {
		nodeIDs[i] = n.ID
	}

	permCounts := make(map[uint64]int64)
	if len(nodeIDs) > 0 {
		type permCount struct {
			DomainNodeID uint64
			Count        int64
		}
		var results []permCount
		h.db.Model(&model.DomainPermission{}).
			Where("domain_node_id IN ?", nodeIDs).
			Select("domain_node_id, count(*) as count").
			Group("domain_node_id").
			Scan(&results)
		for _, r := range results {
			permCounts[r.DomainNodeID] = r.Count
		}
	}

	type treeNode struct {
		ID              uint64      `json:"id"`
		ParentID        *uint64     `json:"parent_id"`
		FullDomain      string      `json:"full_domain"`
		Owner           model.User  `json:"owner"`
		Status          string      `json:"status"`
		PermissionCount int64       `json:"permission_count"`
		CreatedAt       interface{} `json:"created_at"`
	}

	items := make([]treeNode, len(nodes))
	for i, n := range nodes {
		items[i] = treeNode{
			ID:              n.ID,
			ParentID:        n.ParentID,
			FullDomain:      n.FullDomain,
			Owner:           n.Owner,
			Status:          n.Status,
			PermissionCount: permCounts[n.ID],
			CreatedAt:       n.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

func (h *AdminHandler) GetDomainDetail(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var node model.DomainNode
	if err := h.db.Preload("Owner").Preload("Provider").First(&node, nodeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "域名节点不存在"})
		return
	}

	var permissions []model.DomainPermission
	h.db.Where("domain_node_id = ?", nodeID).Preload("User").Find(&permissions)

	transferHistory, _ := h.domainService.GetTransferHistory(nodeID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"node":             node,
			"permissions":      permissions,
			"transfer_history": transferHistory,
		},
	})
}

func (h *AdminHandler) RevokePermission(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	result := h.db.Where("domain_node_id = ? AND user_id = ?", nodeID, userID).Delete(&model.DomainPermission{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "权限记录不存在"})
		return
	}

	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, userID, "revoke_permission", "domain_node", &nodeID,
		map[string]interface{}{"target_user_id": userID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限已撤销"})
}
