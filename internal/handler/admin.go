package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"domainnest/internal/domain/notification"
	"domainnest/internal/errs"
	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"
	"domainnest/internal/ws"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db                *gorm.DB
	domainService     *service.DomainService
	notifSvc          *notification.Service
	inviteCodeService *service.InviteCodeService
	providerService  *service.ProviderService
}

func NewAdminHandler(db *gorm.DB, domainService *service.DomainService, notifSvc *notification.Service, inviteCodeService *service.InviteCodeService, providerService *service.ProviderService) *AdminHandler {
	return &AdminHandler{db: db, domainService: domainService, notifSvc: notifSvc, inviteCodeService: inviteCodeService, providerService: providerService}
}

func (h *AdminHandler) CreateRootDomain(c *gin.Context) {
	var req struct {
		ProviderID uint64 `json:"provider_id" binding:"required"`
		DomainName string `json:"domain_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	adminID := c.GetUint64("user_id")

	// Verify provider exists
	var provider model.DNSProvider
	if err := h.db.First(&provider, req.ProviderID).Error; err != nil {
		errs.JSONErrorCode(c, errs.DNSProviderNotFound)
		return
	}

	var existing model.DomainNode
	if err := h.db.Where("full_domain = ?", req.DomainName).First(&existing).Error; err == nil {
		errs.JSONErrorCode(c, errs.DomainAlreadyExists)
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
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	var req struct {
		UserID uint64 `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	var user model.User
	if err := h.db.First(&user, req.UserID).Error; err != nil {
		errs.JSONErrorCode(c, errs.TargetUserNotFound)
		return
	}

	var node model.DomainNode
	if err := h.db.First(&node, nodeID).Error; err != nil {
		errs.JSONErrorCode(c, errs.DomainNotFound)
		return
	}
	oldOwnerID := node.OwnerID

	if err := h.domainService.AdminTransferNode(nodeID, req.UserID); err != nil {
		errs.JSONError(c, err)
		return
	}

	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, req.UserID, "assign_domain", "domain_node", &nodeID,
		map[string]interface{}{"full_domain": node.FullDomain, "old_owner_id": oldOwnerID, "new_owner_id": req.UserID}, c.ClientIP())

	ws.BroadcastToUser(oldOwnerID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "transfer",
		"node_id": nodeID,
	})
	ws.BroadcastToUser(req.UserID, ws.TypeDomainTreeUpdate, gin.H{
		"action":  "transfer_received",
		"node_id": nodeID,
	})

	// Send notifications to both owners
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		// Notify new owner
		if err := h.notifSvc.Send(req.UserID, notification.AdminAssignedDomain(&node)); err != nil {
			log.Printf("[Notification] AdminAssignedDomain failed: %v", err)
		}
		// Notify old owner
		if oldOwnerID != req.UserID {
			if err := h.notifSvc.Send(oldOwnerID, notification.AdminRemovedDomain(&node)); err != nil {
				log.Printf("[Notification] AdminRemovedDomain failed: %v", err)
			}
		}
		// Broadcast unread counts to both
		svc := service.NewMessageService(h.db)
		for _, uid := range []uint64{req.UserID, oldOwnerID} {
			if count, err := svc.GetNotificationUnreadCount(uid); err == nil {
				ws.BroadcastToUser(uid, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "域名分配成功"})
}

func (h *AdminHandler) BatchDeleteDomains(c *gin.Context) {
	var req struct {
		NodeIDs []uint64 `json:"node_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	// Fetch nodes and owners before deletion so we can notify them
	type nodeOwner struct {
		NodeID   uint64
		OwnerID  uint64
		FullDomain string
	}
	var toNotify []nodeOwner
	var nodes []model.DomainNode
	h.db.Where("id IN ?", req.NodeIDs).Find(&nodes)
	domainNames := make(map[uint64]string)
	for _, n := range nodes {
		domainNames[n.ID] = n.FullDomain
	}
	for _, nodeID := range req.NodeIDs {
		var n model.DomainNode
		if err := h.db.First(&n, nodeID).Error; err != nil {
			continue
		}
		var childCount int64
		h.db.Model(&model.DomainNode{}).Where("parent_id = ? AND deleted_at IS NULL", nodeID).Count(&childCount)
		if childCount > 0 {
			continue
		}
		var recordCount int64
		h.db.Model(&model.DNSRecord{}).Where("node_id = ? AND deleted_at IS NULL", nodeID).Count(&recordCount)
		if recordCount > 0 {
			continue
		}
		toNotify = append(toNotify, nodeOwner{NodeID: n.ID, OwnerID: n.OwnerID, FullDomain: n.FullDomain})
	}

	deleted, skipped, err := h.domainService.AdminBatchDeleteNodes(req.NodeIDs)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	callerID := c.GetUint64("user_id")
	var domainNameList []string
	for _, id := range req.NodeIDs {
		if name, ok := domainNames[id]; ok {
			domainNameList = append(domainNameList, name)
		}
	}
	middleware.LogOperation(h.db, callerID, "batch_delete_domains", "domain_node", nil,
		map[string]interface{}{"node_ids": req.NodeIDs, "domains": domainNameList, "deleted": deleted, "skipped": skipped}, c.ClientIP())

	// Send notifications to affected owners
	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		svc := service.NewMessageService(h.db)
		notified := make(map[uint64]struct{})
		for _, n := range toNotify {
			node := &model.DomainNode{ID: n.NodeID, FullDomain: n.FullDomain}
			if err := h.notifSvc.Send(n.OwnerID, notification.AdminRemovedDomain(node)); err != nil {
				log.Printf("[Notification] AdminRemovedDomain failed for node %d: %v", n.NodeID, err)
			} else {
				if _, ok := notified[n.OwnerID]; !ok {
					notified[n.OwnerID] = struct{}{}
					if count, err := svc.GetNotificationUnreadCount(n.OwnerID); err == nil {
						ws.BroadcastToUser(n.OwnerID, ws.TypeUnreadUpdate, gin.H{"count": count})
					}
				}
			}
		}
	}()

	msg := fmt.Sprintf("成功删除 %d 个域名", deleted)
	if skipped > 0 {
		msg += fmt.Sprintf("，%d 个跳过（有子域名或DNS记录）", skipped)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": msg, "data": gin.H{"deleted": deleted, "skipped": skipped}})
}

func (h *AdminHandler) ListDomains(c *gin.Context) {
	var nodes []model.DomainNode
	if err := h.db.Preload("Owner").Order("id ASC").Find(&nodes).Error; err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": nodes})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.db.Find(&users).Error; err != nil {
		errs.JSONError(c, err)
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
	}).
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
		errs.JSONErrorCode(c, errs.InvalidRecordID)
		return
	}

	if err := h.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).
		Updates(map[string]interface{}{
			"sync_status":     "pending",
			"sync_attempts":   0,
			"next_sync_at":    nil,
			"last_sync_error": "",
		}).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已加入同步重试队列"})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
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
		errs.JSONError(c, err)
		return
	}

	// Handle username change separately (needs uniqueness check)
	if req.Username != "" {
		var existing model.User
		if err := h.db.Where("username = ? AND id != ?", req.Username, userID).First(&existing).Error; err == nil {
			errs.JSONErrorCode(c, errs.UsernameExists)
			return
		}
		if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("username", req.Username).Error; err != nil {
			errs.JSONError(c, err)
			return
		}
	}

	callerID := c.GetUint64("user_id")
	var caller model.User
	h.db.First(&caller, callerID)

	// Fetch target user for permission checks
	var targetUser model.User
	if err := h.db.First(&targetUser, userID).Error; err != nil {
		errs.JSONErrorCode(c, errs.UserNotFound)
		return
	}

	// Non-superadmin cannot edit superadmin
	if targetUser.IsSuperAdmin && !caller.IsSuperAdmin {
		errs.JSONErrorCode(c, errs.CannotEditSuperAdmin)
		return
	}
	// Non-superadmin cannot edit other admins
	if targetUser.Role == "admin" && !caller.IsSuperAdmin {
		errs.JSONErrorCode(c, errs.CannotEditOtherAdmin)
		return
	}

	updates := map[string]interface{}{}
	if req.Role != "" {
		if !caller.IsSuperAdmin {
			errs.JSONErrorCode(c, errs.OnlySuperAdminChangeRole)
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
			errs.JSONErrorCode(c, errs.InviteLimitNegative)
			return
		}
		// Cannot decrease below current invite_count (skip for superadmin)
		if !caller.IsSuperAdmin && *req.InviteLimit < targetUser.InviteCount {
			errs.JSONErrorCode(c, errs.InviteLimitNegative)
			return
		}

		// Pool model: if admin is not super_admin, deduct from admin's available pool
		if !caller.IsSuperAdmin {
			additionalAmount := *req.InviteLimit - targetUser.InviteLimit
			if additionalAmount > 0 {
				available := caller.InviteLimit - caller.InviteCount
				if available < additionalAmount {
					errs.JSONErrorCode(c, errs.InviteLimitNegative)
					return
				}
				// Deduct from admin's pool
				if err := h.db.Model(&caller).UpdateColumn("invite_count", gorm.Expr("invite_count + ?", additionalAmount)).Error; err != nil {
					errs.JSONError(c, err)
					return
				}
			} else if additionalAmount < 0 {
				// Return quota to admin's pool
				if err := h.db.Model(&caller).UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count + ?, 0)", additionalAmount)).Error; err != nil {
					errs.JSONError(c, err)
					return
				}
			}
		}

		updates["invite_limit"] = *req.InviteLimit
	}

	if len(updates) == 0 {
		errs.JSONErrorCode(c, errs.NoFieldsToUpdate)
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	callerID2 := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID2, userID, "update_user", "user", &userID, updates, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户信息已更新"})
}

func (h *AdminHandler) AdminResetPassword(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	var targetUser model.User
	if err := h.db.First(&targetUser, userID).Error; err != nil {
		errs.JSONErrorCode(c, errs.UserNotFound)
		return
	}
	callerID := c.GetUint64("user_id")
	if targetUser.IsSuperAdmin && callerID != targetUser.ID {
		errs.JSONErrorCode(c, errs.CannotResetSuperAdminPassword)
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONErrorCode(c, errs.InvalidRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		errs.JSONErrorCode(c, errs.PasswordEncryptFailed)
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("password", string(hashedPassword)).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	middleware.LogOperationUser(h.db, callerID, userID, "admin_reset_password", "user", &userID,
		map[string]interface{}{"target_username": targetUser.Username}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if err := h.notifSvc.Send(userID, notification.AdminPasswordReset()); err != nil {
			log.Printf("[Notification] AdminPasswordReset failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "密码重置成功"})
}

func (h *AdminHandler) DisableUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	// Prevent self-disable
	callerID := c.GetUint64("user_id")
	if userID == callerID {
		errs.JSONErrorCode(c, errs.CannotDisableSelf)
		return
	}

	tx := h.db.Begin()
	defer tx.Rollback()

	var caller model.User
	if err := tx.First(&caller, callerID).Error; err != nil {
		tx.Rollback()
		errs.JSONErrorCode(c, errs.InternalError)
		return
	}

	// Handle invite pool cleanup: free the registration slot from the inviter
	var user model.User
	if err := tx.First(&user, userID).Error; err != nil {
		errs.JSONErrorCode(c, errs.UserNotFound)
		return
	}

	if user.IsSuperAdmin {
		if !caller.IsSuperAdmin {
			errs.JSONErrorCode(c, errs.OnlySuperAdminDisableSuperAdmin)
			return
		}
	}
	// Non-superadmin cannot disable other admins
	if user.Role == "admin" && !caller.IsSuperAdmin {
		errs.JSONErrorCode(c, errs.CannotDisableOtherAdmin)
		return
	}

	if user.InvitedBy != nil {
		if err := tx.Model(&model.User{}).Where("id = ?", *user.InvitedBy).
			UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - 1, 0)")).Error; err != nil {
			errs.JSONError(c, err)
			return
		}

		// Reclaim unused invite quota back to the inviter
		unusedQuota := user.InviteLimit - user.InviteCount
		if unusedQuota > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", *user.InvitedBy).
				UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", unusedQuota)).Error; err != nil {
				errs.JSONError(c, err)
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
			errs.JSONError(c, err)
			return
		}
	}

	if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("status", 0).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	tx.Commit()

	middleware.LogOperationUser(h.db, callerID, userID, "disable_user", "user", &userID,
		map[string]interface{}{"target_username": user.Username}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if err := h.notifSvc.Send(userID, notification.AccountDisabled()); err != nil {
			log.Printf("[Notification] AccountDisabled failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户已禁用"})
}

func (h *AdminHandler) PromoteToAdmin(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	// Check caller is super_admin
	callerID := c.GetUint64("user_id")
	var caller model.User
	if err := h.db.First(&caller, callerID).Error; err != nil || !caller.IsSuperAdmin {
		errs.JSONErrorCode(c, errs.OnlySuperAdminPromote)
		return
	}

	var target model.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		errs.JSONErrorCode(c, errs.UserNotFound)
		return
	}

	if target.Role == "admin" {
		errs.JSONErrorCode(c, errs.UserAlreadyAdmin)
		return
	}

	if err := h.db.Model(&target).Update("role", "admin").Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	middleware.LogOperationUser(h.db, callerID, targetID, "promote_to_admin", "user", &targetID,
		map[string]interface{}{"target_username": target.Username, "from_role": target.Role}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if err := h.notifSvc.Send(targetID, notification.RolePromoted()); err != nil {
			log.Printf("[Notification] RolePromoted failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户已提升为管理员"})
}

func (h *AdminHandler) DemoteFromAdmin(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	callerID := c.GetUint64("user_id")
	var caller model.User
	if err := h.db.First(&caller, callerID).Error; err != nil || !caller.IsSuperAdmin {
		errs.JSONErrorCode(c, errs.OnlySuperAdminDemote)
		return
	}

	var target model.User
	if err := h.db.First(&target, targetID).Error; err != nil {
		errs.JSONErrorCode(c, errs.UserNotFound)
		return
	}

	if target.IsSuperAdmin {
		errs.JSONErrorCode(c, errs.CannotDemoteSuperAdmin)
		return
	}

	if target.Role != "admin" {
		errs.JSONErrorCode(c, errs.UserNotAdmin)
		return
	}

	if err := h.db.Model(&target).Update("role", "user").Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	middleware.LogOperationUser(h.db, callerID, targetID, "demote_from_admin", "user", &targetID,
		map[string]interface{}{"target_username": target.Username}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if err := h.notifSvc.Send(targetID, notification.RoleDemoted()); err != nil {
			log.Printf("[Notification] RoleDemoted failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "管理员已降级为普通用户"})
}

func (h *AdminHandler) GetDomainTree(c *gin.Context) {
	var nodes []model.DomainNode
	if err := h.db.Preload("Owner").Order("id ASC").Find(&nodes).Error; err != nil {
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	var node model.DomainNode
	if err := h.db.Preload("Owner").Preload("Provider").First(&node, nodeID).Error; err != nil {
		errs.JSONErrorCode(c, errs.DomainNodeNotFound)
		return
	}

	var permissions []model.DomainPermission
	h.db.Where("domain_node_id = ?", nodeID).Preload("User").Find(&permissions)

	transferHistory, _ := h.domainService.GetTransferHistory(nodeID)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"domain":           node,
			"permissions":      permissions,
			"transfer_history": transferHistory,
		},
	})
}

// ListDomainRecords admin views all DNS records for a domain (no ownership check)
func (h *AdminHandler) ListDomainRecords(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	// Verify domain exists
	var node model.DomainNode
	if err := h.db.First(&node, nodeID).Error; err != nil {
		errs.JSONErrorCode(c, errs.DomainNodeNotFound)
		return
	}

	includeTrashed := c.Query("include_trashed") == "true"

	query := h.db.Where("node_id = ?", nodeID)
	if !includeTrashed {
		query = query.Where("trashed_at IS NULL")
	}

	var records []model.DNSRecord
	if err := query.Order("id ASC").Find(&records).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	// Compute stats
	var totalEnabled, totalDisabled, platformCount, providerCount int
	for _, r := range records {
		if r.Enabled {
			totalEnabled++
		} else {
			totalDisabled++
		}
		if r.Source == "provider" {
			providerCount++
		} else {
			platformCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"records": records,
			"stats": gin.H{
				"total":          len(records),
				"enabled":        totalEnabled,
				"disabled":       totalDisabled,
				"platform_count": platformCount,
				"provider_count": providerCount,
			},
		},
	})
}

// AdminDeleteRecord permanently deletes a DNS record (admin, no ownership check)
func (h *AdminHandler) AdminDeleteRecord(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("rid"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidRecordID)
		return
	}

	var record model.DNSRecord
	h.db.First(&record, recordID)

	// Load node to get owner
	var node model.DomainNode
	h.db.First(&node, record.NodeID)

	result := h.db.Unscoped().Where("id = ?", recordID).Delete(&model.DNSRecord{})
	if result.Error != nil {
		errs.JSONError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		errs.JSONErrorCode(c, errs.RecordNotFound)
		return
	}

	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, node.OwnerID, "admin_delete_record", "dns_record", &recordID,
		map[string]interface{}{"host": record.Host, "type": record.RecordType, "node_id": record.NodeID, "domain": node.FullDomain}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "记录已永久删除"})
}

// AdminToggleRecord admin toggles a DNS record enabled/disabled (no ownership check)
func (h *AdminHandler) AdminToggleRecord(c *gin.Context) {
	recordID, err := strconv.ParseUint(c.Param("rid"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidRecordID)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	result := h.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).Update("enabled", req.Enabled)
	if result.Error != nil {
		errs.JSONError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		errs.JSONErrorCode(c, errs.RecordNotFound)
		return
	}

	var record model.DNSRecord
	h.db.First(&record, recordID)

	// Load node to get owner
	var node model.DomainNode
	h.db.First(&node, record.NodeID)

	action := "admin_enable_record"
	if !req.Enabled {
		action = "admin_disable_record"
	}
	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, node.OwnerID, action, "dns_record", &recordID,
		map[string]interface{}{"enabled": req.Enabled, "host": record.Host, "type": record.RecordType, "node_id": record.NodeID, "domain": node.FullDomain}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "状态已更新"})
}

// BroadcastNotification sends a system notification to all users (or a specified list).
func (h *AdminHandler) BroadcastNotification(c *gin.Context) {
	var req struct {
		Title    string   `json:"title" binding:"required"`
		Content  string   `json:"content" binding:"required"`
		Category string   `json:"category"`
		Priority int      `json:"priority"`
		UserIDs  []uint64 `json:"user_ids"` // optional: send to specific users only
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	targetIDs := req.UserIDs
	if len(targetIDs) == 0 {
		// Broadcast to all active users
		var users []model.User
		if err := h.db.Where("status = 1").Select("id").Find(&users).Error; err != nil {
			errs.JSONError(c, err)
			return
		}
		if len(users) > 10000 {
			errs.JSONErrorCode(c, errs.InvalidParams)
			return
		}
		targetIDs = make([]uint64, len(users))
		for i, u := range users {
			targetIDs[i] = u.ID
		}
	}

	n := notification.Notification{
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Priority: notification.Priority(req.Priority),
	}

	if err := h.notifSvc.SendToMultiple(targetIDs, n); err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "通知已发送", "data": gin.H{"count": len(targetIDs)}})
}

// ListAllNotifications returns all system notifications with filters (admin view).
func (h *AdminHandler) ListAllNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&model.Message{}).Where("type = ?", "system")

	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("receiver_id = ?", userID)
	}
	if start := c.Query("start_date"); start != "" {
		query = query.Where("created_at >= ?", start)
	}
	if end := c.Query("end_date"); end != "" {
		query = query.Where("created_at <= ?", end)
	}

	var total int64
	query.Count(&total)

	var notifications []model.Message
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&notifications)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     notifications,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetNotificationStats returns notification statistics.
func (h *AdminHandler) GetNotificationStats(c *gin.Context) {
	var total int64
	h.db.Model(&model.Message{}).Where("type = ?", "system").Count(&total)

	var unread int64
	h.db.Model(&model.Message{}).Where("type = ? AND read_at IS NULL", "system").Count(&unread)

	type categoryCount struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}
	var byCategory []categoryCount
	h.db.Model(&model.Message{}).
		Where("type = ?", "system").
		Select("category, count(*) as count").
		Group("category").
		Scan(&byCategory)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total":       total,
			"unread":      unread,
			"by_category": byCategory,
		},
	})
}

// AdminDeleteNotification permanently deletes a notification (admin, no ownership check).
func (h *AdminHandler) AdminDeleteNotification(c *gin.Context) {
	notifID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNotificationID)
		return
	}

	result := h.db.Where("id = ? AND type = ?", notifID, "system").Delete(&model.Message{})
	if result.Error != nil {
		errs.JSONError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		errs.JSONErrorCode(c, errs.NotificationNotFound)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "通知已删除"})
}

// AdminPurgeExpiredNotifications manually triggers expired notification purge.
func (h *AdminHandler) AdminPurgeExpiredNotifications(c *gin.Context) {
	deleted := h.notifSvc.PurgeExpired()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已清理过期通知", "data": gin.H{"deleted": deleted}})
}

func (h *AdminHandler) RevokePermission(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNodeID)
		return
	}

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	result := h.db.Where("domain_node_id = ? AND user_id = ?", nodeID, userID).Delete(&model.DomainPermission{})
	if result.Error != nil {
		errs.JSONError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		errs.JSONErrorCode(c, errs.PermissionNotFound)
		return
	}

	callerID := c.GetUint64("user_id")
	middleware.LogOperationUser(h.db, callerID, userID, "revoke_permission", "domain_node", &nodeID,
		map[string]interface{}{"target_user_id": userID}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		var node model.DomainNode
		if h.db.First(&node, nodeID).Error == nil {
			if err := h.notifSvc.Send(userID, notification.PermissionRevoked(&node)); err != nil {
				log.Printf("[Notification] PermissionRevoked failed: %v", err)
				return
			}
			svc := service.NewMessageService(h.db)
			if count, err := svc.GetNotificationUnreadCount(userID); err == nil {
				ws.BroadcastToUser(userID, ws.TypeUnreadUpdate, gin.H{"count": count})
			}
			ws.BroadcastToUser(userID, ws.TypeDomainTreeUpdate, gin.H{
				"action":  "permission_change",
				"node_id": nodeID,
			})
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "权限已撤销"})
}

func (h *AdminHandler) GenerateInviteCodes(c *gin.Context) {
	var req struct {
		Count int `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}
	callerID := c.GetUint64("user_id")
	codes, err := h.inviteCodeService.GenerateCodes(callerID, req.Count)
	if err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": codes})
}

func (h *AdminHandler) ListInviteCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	codes, total, err := h.inviteCodeService.ListCodes(page, pageSize)
	if err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     codes,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *AdminHandler) DeleteInviteCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidID)
		return
	}
	// Admin can delete any code — find it first to check usage
	var code model.InviteCode
	if err := h.db.First(&code, id).Error; err != nil {
		errs.JSONErrorCode(c, errs.InviteCodeNotFound)
		return
	}
	if code.UsedBy != nil {
		errs.JSONErrorCode(c, errs.CannotDeleteUsedInviteCode)
		return
	}
	creatorID := code.CreatorID
	creatorUsername := ""
	var creator model.User
	if h.db.Select("id", "username").First(&creator, creatorID).Error == nil {
		creatorUsername = creator.Username
	}
	if err := h.db.Delete(&code).Error; err != nil {
		errs.JSONError(c, err)
		return
	}

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[Notification] panic: %v", r) } }()
		if err := h.notifSvc.Send(creatorID, notification.InviteCodeDeletedByAdmin(code.Code)); err != nil {
			log.Printf("[Notification] InviteCodeDeletedByAdmin failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "邀请码已删除", "data": gin.H{"creator_id": creatorID, "creator_username": creatorUsername}})
}

func (h *AdminHandler) ListAllProviders(c *gin.Context) {
	providers, err := h.providerService.ListAll()
	if err != nil {
		errs.JSONError(c, err)
		return
	}
	// Hide access_key_secret
	for i := range providers {
		providers[i].AccessKeySecret = ""
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": providers})
}

func (h *AdminHandler) GetProviderDetail(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidProviderIDAdmin)
		return
	}

	var provider model.DNSProvider
	if err := h.db.Preload("User").First(&provider, providerID).Error; err != nil {
		errs.JSONErrorCode(c, errs.ProviderNotFound)
		return
	}
	provider.AccessKeySecret = ""

	domains, err := h.providerService.ListDomainsWithStatus(providerID)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"provider": provider,
		"domains":  domains,
	}})
}

func (h *AdminHandler) UpdateProvider(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidProviderIDAdmin)
		return
	}

	var req struct {
		Name     string `json:"name"`
		Endpoint string `json:"endpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}

	if err := h.providerService.AdminUpdate(providerID, req.Name, req.Endpoint); err != nil {
		errs.JSONError(c, err)
		return
	}

	userID := c.GetUint64("user_id")
	middleware.LogOperation(h.db, userID, "update_provider", "dns_provider", &providerID,
		map[string]interface{}{"name": req.Name, "endpoint": req.Endpoint}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *AdminHandler) DeleteProvider(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidProviderIDAdmin)
		return
	}

	confirm := c.Query("confirm") == "true"

	affected, err := h.providerService.AdminDelete(providerID, confirm)
	if err != nil {
		if affected > 0 {
			errs.JSONError(c, err)
			return
		}
		errs.JSONError(c, err)
		return
	}

	userID := c.GetUint64("user_id")
	middleware.LogOperation(h.db, userID, "delete_provider", "dns_provider", &providerID,
		map[string]interface{}{"affected_domains": affected}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
