package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AcmeDNSCompatHandler struct {
	svc          *service.AliyunCompatService
	ramTokenSvc  *service.RAMTokenService
	db           *gorm.DB
}

func NewAcmeDNSCompatHandler(svc *service.AliyunCompatService, ramTokenSvc *service.RAMTokenService, db *gorm.DB) *AcmeDNSCompatHandler {
	return &AcmeDNSCompatHandler{svc: svc, ramTokenSvc: ramTokenSvc, db: db}
}

func (h *AcmeDNSCompatHandler) Register(c *gin.Context) {
	userID := c.GetUint64("user_id") // may be 0 if no auth on register

	// Generate credentials
	subdomain := uuid.New().String()

	usernameBytes := make([]byte, 16)
	rand.Read(usernameBytes)
	username := hex.EncodeToString(usernameBytes)

	passwordBytes := make([]byte, 24)
	rand.Read(passwordBytes)
	password := hex.EncodeToString(passwordBytes)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Parse optional fqdn from body
	var req struct {
		FQDN string `json:"fqdn"`
	}
	c.ShouldBindJSON(&req)

	nodeID := uint64(0)
	if req.FQDN != "" {
		fqdn := strings.TrimSuffix(req.FQDN, ".")
		node, _, err := h.svc.ResolveDomain(fqdn, userID)
		if err == nil {
			nodeID = node.ID
			if userID == 0 {
				userID = node.OwnerID
			}
		}
	}

	account := &model.AcmeDNSAccount{
		UserID:     userID,
		Username:   username,
		Password:   string(passwordHash),
		Subdomain:  subdomain,
		FullDomain: req.FQDN,
		NodeID:     nodeID,
	}

	if err := h.db.Create(account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"username":   username,
		"password":   password,
		"fulldomain": subdomain + ".acmedns." + c.Request.Host,
		"subdomain":  subdomain,
		"allowfrom":  []string{"0.0.0.0/0"},
	})
}

func (h *AcmeDNSCompatHandler) Update(c *gin.Context) {
	accountVal, exists := c.Get("acmedns_account")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	account := accountVal.(*model.AcmeDNSAccount)

	var req struct {
		Subdomain string `json:"subdomain"`
		TXT       string `json:"txt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Subdomain != account.Subdomain {
		c.JSON(http.StatusForbidden, gin.H{"error": "subdomain mismatch"})
		return
	}

	// Resolve the domain from the account
	if account.NodeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no domain associated with this account"})
		return
	}

	node, _, err := h.svc.ResolveDomain(account.FullDomain, account.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	// Compute rr
	rr := "@"
	fqdn := strings.TrimSuffix(account.FullDomain, ".")
	if fqdn != node.FullDomain {
		suffix := "." + node.FullDomain
		if strings.HasSuffix(fqdn, suffix) {
			rr = strings.TrimSuffix(fqdn, suffix)
		}
	}

	// Check if record exists
	existing, _, _ := h.svc.DescribeDomainRecords(account.UserID, node.FullDomain, rr, "TXT", "", 1, 100)
	if existing != nil && existing.Total > 0 {
		// Update first matching record
		for _, r := range existing.Items {
			if r.Host == rr {
				h.svc.UpdateDomainRecord(account.UserID,
					strconv.FormatUint(r.ID, 10), rr, "TXT", req.TXT, 60, nil)
				break
			}
		}
	} else {
		h.svc.AddDomainRecord(account.UserID, node.FullDomain, rr, "TXT", req.TXT, 60, nil, "default")
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}