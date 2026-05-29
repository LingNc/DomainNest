package handler

import (
	"net/http"
	"strconv"
	"strings"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type HTTPReqCompatHandler struct {
	svc *service.AliyunCompatService
}

func NewHTTPReqCompatHandler(svc *service.AliyunCompatService) *HTTPReqCompatHandler {
	return &HTTPReqCompatHandler{svc: svc}
}

func (h *HTTPReqCompatHandler) Present(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		FQDN  string `json:"fqdn"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Strip trailing dot
	fqdn := strings.TrimSuffix(req.FQDN, ".")

	// Resolve FQDN to node
	node, _, err := h.svc.ResolveDomain(fqdn, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	// Compute rr from FQDN
	rr := "@"
	if fqdn != node.FullDomain {
		suffix := "." + node.FullDomain
		if strings.HasSuffix(fqdn, suffix) {
			rr = strings.TrimSuffix(fqdn, suffix)
		}
	}

	// Check if record already exists, update or create
	existing, _, err := h.svc.DescribeDomainRecords(userID, node.FullDomain, rr, "TXT", "", 1, 100)
	if err == nil && existing != nil && existing.Total > 0 {
		// Update existing
		for _, r := range existing.Items {
			if r.Host == rr && r.RecordType == "TXT" {
				recordIDStr := strconv.FormatUint(r.ID, 10)
				h.svc.UpdateDomainRecord(userID, recordIDStr, rr, "TXT", req.Value, 60, nil)
				break
			}
		}
	} else {
		// Create new
		h.svc.AddDomainRecord(userID, node.FullDomain, rr, "TXT", req.Value, 60, nil, "default")
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *HTTPReqCompatHandler) Cleanup(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		FQDN  string `json:"fqdn"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true}) // idempotent
		return
	}

	fqdn := strings.TrimSuffix(req.FQDN, ".")

	node, _, err := h.svc.ResolveDomain(fqdn, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": true}) // domain not found, nothing to clean
		return
	}

	rr := "@"
	if fqdn != node.FullDomain {
		suffix := "." + node.FullDomain
		if strings.HasSuffix(fqdn, suffix) {
			rr = strings.TrimSuffix(fqdn, suffix)
		}
	}

	// Find and delete matching TXT records
	existing, _, err := h.svc.DescribeDomainRecords(userID, node.FullDomain, rr, "TXT", req.Value, 1, 100)
	if err == nil && existing != nil {
		for _, r := range existing.Items {
			if r.Host == rr && r.RecordType == "TXT" && r.Value == req.Value {
				recordIDStr := strconv.FormatUint(r.ID, 10)
				h.svc.DeleteDomainRecord(userID, recordIDStr)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}