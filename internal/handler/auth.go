package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"time"

	"domainnest/internal/config"
	"domainnest/internal/domain/notification"
	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"
	"domainnest/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

func friendlyValidationError(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			field := fe.Field()
			switch fe.Tag() {
			case "min":
				if field == "Password" || field == "NewPassword" {
					return fmt.Sprintf("密码长度不能少于%s个字符", fe.Param())
				}
				if field == "Username" {
					return fmt.Sprintf("用户名长度不能少于%s个字符", fe.Param())
				}
				return fmt.Sprintf("%s长度不足，最少需要%s个字符", field, fe.Param())
			case "required":
				return fmt.Sprintf("%s不能为空", field)
			case "max":
				return fmt.Sprintf("%s长度超出限制，最多%s个字符", field, fe.Param())
			case "email":
				return "邮箱格式不正确"
			case "oneof":
				return fmt.Sprintf("%s的值无效，可选值：%s", field, fe.Param())
			}
		}
	}
	return "请求参数无效"
}

type AuthHandler struct {
	authService     *service.AuthService
	emailSvc        *service.EmailService
	emailVerifySvc  *service.EmailVerifyService
	settingsService *service.SettingsService
	notifSvc        *notification.Service
	db              *gorm.DB
	jwtSecret       string
	jwtExpire       int
}

func NewAuthHandler(authService *service.AuthService, emailSvc *service.EmailService, emailVerifySvc *service.EmailVerifyService, settingsService *service.SettingsService, notifSvc *notification.Service, db *gorm.DB, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		emailSvc:        emailSvc,
		emailVerifySvc:  emailVerifySvc,
		settingsService: settingsService,
		notifSvc:        notifSvc,
		db:              db,
		jwtSecret:       cfg.Secret,
		jwtExpire:       cfg.ExpireHours,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required,min=3,max=64"`
		Password   string `json:"password" binding:"required,min=6"`
		Email      string `json:"email"`
		InviteCode string `json:"invite_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": friendlyValidationError(err)})
		return
	}

	user, err := h.authService.Register(req.Username, req.Password, req.Email, req.InviteCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, user.ID, "register", "user", &user.ID,
		map[string]interface{}{"username": user.Username, "invited_by": user.InvitedBy}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) CheckUsername(c *gin.Context) {
	username := c.Query("username")
	if len(username) < 3 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"available": false}})
		return
	}

	var user model.User
	err := h.db.Where("username = ?", username).First(&user).Error
	available := err != nil // gorm.ErrRecordNotFound means available
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"available": available}})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
		return
	}

	token, err := middleware.GenerateToken(h.jwtSecret, user.ID, user.Username, user.Role, h.jwtExpire)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成令牌失败"})
		return
	}

	middleware.LogOperation(h.db, user.ID, "login", "user", &user.ID,
		map[string]interface{}{"username": user.Username}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "成功",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":             user.ID,
				"username":       user.Username,
				"nickname":       user.Nickname,
				"avatar":         user.Avatar,
				"role":           user.Role,
				"is_super_admin": user.IsSuperAdmin,
			},
		},
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"nickname":     user.Nickname,
			"phone":        user.Phone,
			"avatar":       user.Avatar,
			"role":         user.Role,
			"is_super_admin": user.IsSuperAdmin,
			"ddns_token":   user.Token,
			"invite_code":  user.InviteCode,
			"invite_limit": user.InviteLimit,
			"invite_count": user.InviteCount,
		},
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	changed := map[string]interface{}{}
	if req.Username != "" {
		if err := h.authService.UpdateUsername(userID, req.Username); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		changed["username"] = req.Username
	}

	if err := h.authService.UpdateProfile(userID, req.Nickname, req.Phone, req.Email, req.Avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	if req.Nickname != "" {
		changed["nickname"] = req.Nickname
	}
	if req.Phone != "" {
		changed["phone"] = req.Phone
	}
	if req.Email != "" {
		changed["email"] = req.Email
	}
	if req.Avatar != "" {
		changed["avatar"] = "[updated]"
	}

	middleware.LogOperation(h.db, userID, "update_profile", "user", &userID, changed, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "个人资料已更新"})
}

func (h *AuthHandler) ResetToken(c *gin.Context) {
	userID := c.GetUint64("user_id")

	newToken, err := h.authService.ResetToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "重置令牌失败"})
		return
	}

	middleware.LogOperation(h.db, userID, "reset_token", "user", &userID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "令牌重置成功",
		"data": gin.H{
			"token": newToken,
		},
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": friendlyValidationError(err)})
		return
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "change_password", "user", &userID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "密码修改成功"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 无论邮箱是否存在都返回成功（防枚举）
	var user model.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "如果该邮箱存在，验证码已发送"})
		return
	}

	code, err := service.GenerateVerifyCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成验证码失败"})
		return
	}

	// Get security settings for token expiry
	expiryMinutes := 5 // default
	if raw, _ := h.settingsService.Get("security"); raw != "" {
		var secSettings map[string]interface{}
		if json.Unmarshal([]byte(raw), &secSettings) == nil {
			if v, ok := secSettings["reset_token_expiry_minutes"].(float64); ok && v > 0 {
				expiryMinutes = int(v)
			}
		}
	}

	reset := &model.PasswordReset{
		UserID:    user.ID,
		Token:     code,
		ExpiresAt: time.Now().Add(time.Duration(expiryMinutes) * time.Minute),
	}
	if err := h.db.Create(reset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建重置验证码失败"})
		return
	}

	go func() {
		if err := h.emailSvc.SendPasswordReset(user.Email, code, expiryMinutes); err != nil {
			log.Printf("[Auth] Failed to send reset email to %s: %v", user.Email, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "如果该邮箱存在，验证码已发送"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": friendlyValidationError(err)})
		return
	}

	var reset model.PasswordReset
	if err := h.db.Where("token = ? AND used = false AND expires_at > ?", req.Token, time.Now()).First(&reset).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "验证码无效或已过期"})
		return
	}

	if err := h.authService.AdminResetPassword(reset.UserID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "重置密码失败"})
		return
	}

	h.db.Model(&reset).Update("used", true)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "密码重置成功"})
}

func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetUint64("user_id")

	// Limit upload size to 5MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件上传失败"})
		return
	}
	defer file.Close()

	// Detect format and decode
	var img image.Image
	header := make([]byte, 512)
	n, _ := file.Read(header)
	file.Seek(0, 0)
	contentType := http.DetectContentType(header[:n])

	switch contentType {
	case "image/jpeg":
		img, err = jpeg.Decode(file)
	case "image/png":
		img, err = png.Decode(file)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持 JPEG 和 PNG 格式"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "failed to decode image: " + err.Error()})
		return
	}

	// Resize to 128x128
	resized := resize.Resize(128, 128, img, resize.Lanczos3)

	// Encode to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "图片编码失败"})
		return
	}

	// Convert to base64 data URI
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURI := fmt.Sprintf("data:image/jpeg;base64,%s", b64)

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("avatar", dataURI).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存头像失败"})
		return
	}

	middleware.LogOperation(h.db, userID, "upload_avatar", "user", &userID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"avatar": dataURI}})
}

func (h *AuthHandler) MyLogs(c *gin.Context) {
	userID := c.GetUint64("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&model.OperationLog{})
	view := c.DefaultQuery("view", "all")
	switch view {
	case "actor":
		query = query.Where("operation_logs.user_id = ?", userID)
	case "target":
		query = query.Where("operation_logs.target_user_id = ?", userID)
	default:
		query = query.Where("operation_logs.user_id = ? OR operation_logs.target_user_id = ?", userID, userID)
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

func (h *AuthHandler) GrantInviteQuota(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		TargetUserID uint64 `json:"target_user_id" binding:"required"`
		Amount       int    `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.authService.GrantInviteQuota(userID, req.TargetUserID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, userID, req.TargetUserID, "grant_invite", "user", &req.TargetUserID,
		map[string]interface{}{"amount": req.Amount}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		if err := h.notifSvc.Send(req.TargetUserID, notification.InviteGranted(req.Amount)); err != nil {
			log.Printf("[Notification] InviteGranted failed: %v", err)
			return
		}
		svc := service.NewMessageService(h.db)
		if count, err := svc.GetNotificationUnreadCount(req.TargetUserID); err == nil {
			ws.BroadcastToUser(req.TargetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "邀请额度已分配"})
}

func (h *AuthHandler) RevokeInviteQuota(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		TargetUserID uint64 `json:"target_user_id" binding:"required"`
		Amount       int    `json:"amount" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.authService.RevokeInviteQuota(userID, req.TargetUserID, req.Amount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperationUser(h.db, userID, req.TargetUserID, "revoke_invite", "user", &req.TargetUserID,
		map[string]interface{}{"amount": req.Amount}, c.ClientIP())

	go func() {
		defer func() { if r := recover(); r != nil { log.Printf("[WS] BroadcastToUser panic: %v", r) } }()
		if err := h.notifSvc.Send(req.TargetUserID, notification.InviteRevoked(req.Amount)); err != nil {
			log.Printf("[Notification] InviteRevoked failed: %v", err)
			return
		}
		svc := service.NewMessageService(h.db)
		if count, err := svc.GetNotificationUnreadCount(req.TargetUserID); err == nil {
			ws.BroadcastToUser(req.TargetUserID, ws.TypeUnreadUpdate, gin.H{"count": count})
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "邀请额度已撤销"})
}

func (h *AuthHandler) GetInviteLogs(c *gin.Context) {
	userID := c.GetUint64("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	h.db.Model(&model.InviteLog{}).Where("inviter_id = ? OR invitee_id = ?", userID, userID).Count(&total)

	var logs []model.InviteLog
	h.db.Where("inviter_id = ? OR invitee_id = ?", userID, userID).
		Preload("Inviter", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "username", "nickname")
		}).Preload("Invitee", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "username", "nickname")
		}).
		Order("created_at DESC").
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

func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetUint64("user_id")
	if err := h.authService.DeleteAccount(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_account", "user", &userID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "账号已注销"})
}

// SendVerifyEmail sends a verification code to the given email.
// POST /api/v1/auth/send-verify-email  { "email": "...", "purpose": "register"|"change_email" }
func (h *AuthHandler) SendVerifyEmail(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required,email"`
		Purpose string `json:"purpose" binding:"required,oneof=register change_email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": friendlyValidationError(err)})
		return
	}

	if err := h.emailVerifySvc.SendCode(req.Email, req.Purpose); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "验证码发送失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "验证码已发送"})
}

// VerifyEmail verifies a code and marks the user's email as verified.
// POST /api/v1/auth/verify-email  { "email": "...", "code": "...", "purpose": "register"|"change_email" }
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Email   string `json:"email" binding:"required,email"`
		Code    string `json:"code" binding:"required"`
		Purpose string `json:"purpose" binding:"required,oneof=register change_email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": friendlyValidationError(err)})
		return
	}

	if !h.emailVerifySvc.VerifyCode(req.Email, req.Code, req.Purpose) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "验证码无效或已过期"})
		return
	}

	// For change_email, update the authenticated user's email
	if req.Purpose == "change_email" {
		userID := c.GetUint64("user_id")
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "请先登录"})
			return
		}
		if err := h.db.Model(&model.User{}).Where("id = ?", userID).
			Updates(map[string]interface{}{"email": req.Email, "email_verified": true}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新邮箱失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "邮箱已验证并更新"})
		return
	}

	// For register, mark email as verified for the user with this email
	h.db.Model(&model.User{}).Where("email = ?", req.Email).Update("email_verified", true)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "邮箱验证成功"})
}
