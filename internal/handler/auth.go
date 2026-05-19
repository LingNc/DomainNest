package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"time"

	"domainnest/internal/config"
	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

type AuthHandler struct {
	authService *service.AuthService
	emailSvc    *service.EmailService
	db          *gorm.DB
	jwtSecret   string
	jwtExpire   int
}

func NewAuthHandler(authService *service.AuthService, emailSvc *service.EmailService, db *gorm.DB, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		emailSvc:    emailSvc,
		db:          db,
		jwtSecret:   cfg.Secret,
		jwtExpire:   cfg.ExpireHours,
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
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
		"message": "success",
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate token"})
		return
	}

	middleware.LogOperation(h.db, user.ID, "login", "user", &user.ID,
		map[string]interface{}{"username": user.Username}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"avatar":   user.Avatar,
				"role":     user.Role,
			},
		},
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "user not found"})
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

	if req.Username != "" {
		if err := h.authService.UpdateUsername(userID, req.Username); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
	}

	if err := h.authService.UpdateProfile(userID, req.Nickname, req.Phone, req.Email, req.Avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "profile updated"})
}

func (h *AuthHandler) ResetToken(c *gin.Context) {
	userID := c.GetUint64("user_id")

	newToken, err := h.authService.ResetToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to reset token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "token reset successfully",
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "password changed successfully"})
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
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "if the email exists, a code has been sent"})
		return
	}

	code, err := service.GenerateVerifyCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate code"})
		return
	}

	reset := &model.PasswordReset{
		UserID:    user.ID,
		Token:     code,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := h.db.Create(reset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to create reset code"})
		return
	}

	go h.emailSvc.SendPasswordReset(user.Email, code)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "if the email exists, a code has been sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var reset model.PasswordReset
	if err := h.db.Where("token = ? AND used = false AND expires_at > ?", req.Token, time.Now()).First(&reset).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid or expired token"})
		return
	}

	if err := h.authService.AdminResetPassword(reset.UserID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to reset password"})
		return
	}

	h.db.Model(&reset).Update("used", true)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "password reset successfully"})
}

func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetUint64("user_id")

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "missing file"})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "only JPEG and PNG are supported"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to encode image"})
		return
	}

	// Convert to base64 data URI
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	dataURI := fmt.Sprintf("data:image/jpeg;base64,%s", b64)

	if err := h.db.Model(&model.User{}).Where("id = ?", userID).Update("avatar", dataURI).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to save avatar"})
		return
	}

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

	query := h.db.Model(&model.OperationLog{}).Where("user_id = ?", userID)

	// Filter by action type
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	// Filter by target type
	if targetType := c.Query("target_type"); targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	// Filter by date range
	if startTime := c.Query("start_time"); startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		query = query.Where("created_at <= ?", endTime)
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

	middleware.LogOperation(h.db, userID, "grant_invite", "user", &req.TargetUserID,
		map[string]interface{}{"amount": req.Amount}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "invite quota granted"})
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

	middleware.LogOperation(h.db, userID, "revoke_invite", "user", &req.TargetUserID,
		map[string]interface{}{"amount": req.Amount}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "invite quota revoked"})
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
		Preload("Inviter").Preload("Invitee").
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
