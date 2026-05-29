package handler

import (
	"net/http"
	"strconv"
	"strings"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type TechnitiumCompatHandler struct {
	svc *service.AliyunCompatService
}

func NewTechnitiumCompatHandler(svc *service.AliyunCompatService) *TechnitiumCompatHandler {
	return &TechnitiumCompatHandler{svc: svc}
}

func (h *TechnitiumCompatHandler) AddRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")

	domain := c.PostForm("domain")
	zone := c.PostForm("zone")
	recordType := c.DefaultPostForm("type", "TXT")
	text := c.PostForm("text")
	ttlStr := c.DefaultPostForm("ttl", "600")

	if domain == "" || zone == "" || text == "" {
		c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": "missing required parameters"})
		return
	}

	ttl, _ := strconv.Atoi(ttlStr)
	if ttl == 0 {
		ttl = 600
	}

	rr := "@"
	if domain != zone {
		suffix := "." + zone
		if strings.HasSuffix(domain, suffix) {
			rr = strings.TrimSuffix(domain, suffix)
		}
	}

	_, err := h.svc.AddDomainRecord(userID, zone, rr, recordType, text, ttl, nil, "default")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *TechnitiumCompatHandler) DeleteRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")

	domain := c.PostForm("domain")
	zone := c.PostForm("zone")
	text := c.PostForm("text")

	if domain == "" || zone == "" {
		c.JSON(http.StatusOK, gin.H{"status": "error", "errorMessage": "missing required parameters"})
		return
	}

	rr := "@"
	if domain != zone {
		suffix := "." + zone
		if strings.HasSuffix(domain, suffix) {
			rr = strings.TrimSuffix(domain, suffix)
		}
	}

	result, _, err := h.svc.DescribeDomainRecords(userID, zone, rr, "TXT", text, 1, 100)
	if err != nil || result == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	for _, rec := range result.Items {
		if rec.Host == rr && rec.RecordType == "TXT" {
			if text == "" || rec.Value == text {
				h.svc.DeleteDomainRecord(userID, strconv.FormatUint(rec.ID, 10))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}