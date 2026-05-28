package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type TencentCompatHandler struct {
	svc        *service.AliyunCompatService
	ramTokenSvc *service.RAMTokenService
}

func NewTencentCompatHandler(svc *service.AliyunCompatService, ramTokenSvc *service.RAMTokenService) *TencentCompatHandler {
	return &TencentCompatHandler{svc: svc, ramTokenSvc: ramTokenSvc}
}

func (h *TencentCompatHandler) checkRAMAccess(c *gin.Context, nodeID uint64, recordType string) error {
	tokenVal, exists := c.Get("ram_token")
	if !exists {
		return nil
	}
	token, ok := tokenVal.(*model.RAMToken)
	if !ok || token == nil {
		return nil
	}
	if err := h.ramTokenSvc.CheckDomainAccess(token, nodeID); err != nil {
		return err
	}
	if recordType != "" {
		if err := h.ramTokenSvc.CheckRecordType(token, recordType); err != nil {
			return err
		}
	}
	return nil
}

func (h *TencentCompatHandler) Dispatch(c *gin.Context) {
	action := c.GetHeader("X-TC-Action")
	if action == "" {
		h.writeError(c, "InvalidParameter", "Missing X-TC-Action header")
		return
	}

	switch action {
	case "DescribeDomainList":
		h.DescribeDomainList(c)
	case "DescribeRecordList":
		h.DescribeRecordList(c)
	case "CreateRecord":
		h.CreateRecord(c)
	case "ModifyRecord":
		h.ModifyRecord(c)
	case "DeleteRecord":
		h.DeleteRecord(c)
	default:
		h.writeError(c, "InvalidAction.NotFound", "The specified action is not valid.")
	}
}

func (h *TencentCompatHandler) DescribeDomainList(c *gin.Context) {
	userID := c.GetUint64("user_id")
	domains, err := h.svc.DescribeDomains(userID)
	if err != nil {
		h.writeError(c, "InternalError", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"Response": gin.H{
			"RequestId":   genReqID(),
			"TotalCount":  len(domains),
			"DomainList":  domains,
		},
	})
}

func (h *TencentCompatHandler) DescribeRecordList(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var body struct {
		Domain     string `json:"Domain"`
		Subdomain  string `json:"Subdomain"`
		RecordType string `json:"RecordType"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Domain = c.Query("Domain")
		body.Subdomain = c.Query("Subdomain")
		body.RecordType = c.Query("RecordType")
	}

	if body.Domain == "" {
		h.writeError(c, "MissingParameter", "Domain is required.")
		return
	}

	// Build full domain from Subdomain + Domain
	domainName := body.Domain
	if body.Subdomain != "" && body.Subdomain != "@" {
		domainName = body.Subdomain + "." + body.Domain
	}

	result, nodeID, err := h.svc.DescribeDomainRecords(userID, domainName, "", body.RecordType, "", 1, 100)
	if err != nil {
		h.writeError(c, "InvalidDomainName.NoExist", "The specified domain name does not exist.")
		return
	}

	if err := h.checkRAMAccess(c, nodeID, body.RecordType); err != nil {
		h.writeError(c, "Forbidden", err.Error())
		return
	}

	var records []gin.H
	for _, r := range result.Items {
		status := "Enable"
		if !r.Enabled {
			status = "Disable"
		}
		records = append(records, gin.H{
			"RecordId":   strconv.FormatUint(r.ID, 10),
			"DomainName": body.Domain,
			"SubDomain":  r.Host,
			"RecordType": r.RecordType,
			"Value":      r.Value,
			"TTL":        r.TTL,
			"Priority":   r.Priority,
			"Line":       r.Line,
			"Status":     status,
		})
	}
	if records == nil {
		records = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"Response": gin.H{
			"RequestId":   genReqID(),
			"TotalCount":  result.Total,
			"RecordList":  records,
		},
	})
}

func (h *TencentCompatHandler) CreateRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var body struct {
		Domain     string `json:"Domain"`
		SubDomain  string `json:"SubDomain"`
		RecordType string `json:"RecordType"`
		Value      string `json:"Value"`
		TTL        int    `json:"TTL"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.writeError(c, "InvalidParameter", "Invalid request body")
		return
	}

	if body.Domain == "" || body.SubDomain == "" || body.RecordType == "" || body.Value == "" {
		h.writeError(c, "MissingParameter", "Required parameters are missing.")
		return
	}

	if body.TTL == 0 {
		body.TTL = 600
	}

	// Build full domain
	fqdn := body.SubDomain
	if body.SubDomain == "@" || body.SubDomain == "" {
		fqdn = body.Domain
	} else {
		fqdn = body.SubDomain + "." + body.Domain
	}

	node, nodeID, err := h.svc.ResolveDomain(fqdn, userID)
	if err != nil {
		h.writeError(c, "InvalidDomainName.NoExist", "域名不存在或无访问权限")
		return
	}
	if err := h.checkRAMAccess(c, nodeID, body.RecordType); err != nil {
		h.writeError(c, "Forbidden", err.Error())
		return
	}

	record, err := h.svc.AddDomainRecord(userID, body.Domain, body.SubDomain, body.RecordType, body.Value, body.TTL, nil, "default")
	if err != nil {
		h.writeError(c, "DomainRecordDuplicate", err.Error())
		return
	}

	_ = node

	c.JSON(http.StatusOK, gin.H{
		"Response": gin.H{
			"RequestId": genReqID(),
			"RecordId":  strconv.FormatUint(record.ID, 10),
			"SubDomain": record.Host,
			"RecordType": record.RecordType,
			"Value":     record.Value,
			"TTL":       record.TTL,
		},
	})
}

func (h *TencentCompatHandler) ModifyRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var body struct {
		Domain     string `json:"Domain"`
		RecordId   string `json:"RecordId"`
		SubDomain  string `json:"SubDomain"`
		RecordType string `json:"RecordType"`
		Value      string `json:"Value"`
		TTL        int    `json:"TTL"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.writeError(c, "InvalidParameter", "Invalid request body")
		return
	}

	if body.RecordId == "" || body.SubDomain == "" || body.RecordType == "" || body.Value == "" {
		h.writeError(c, "MissingParameter", "Required parameters are missing.")
		return
	}

	recordIDNum, err := strconv.ParseUint(body.RecordId, 10, 64)
	if err != nil {
		h.writeError(c, "InvalidParameter", "无效的记录ID")
		return
	}
	record, err := h.svc.GetRecord(recordIDNum)
	if err != nil {
		h.writeError(c, "InvalidRecordID.NotExist", "记录不存在")
		return
	}
	if err := h.checkRAMAccess(c, record.NodeID, body.RecordType); err != nil {
		h.writeError(c, "Forbidden", err.Error())
		return
	}

	if body.TTL == 0 {
		body.TTL = 600
	}

	record, err = h.svc.UpdateDomainRecord(userID, body.RecordId, body.SubDomain, body.RecordType, body.Value, body.TTL, nil)
	if err != nil {
		h.writeError(c, "DomainRecordNotBelongToUser", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"Response": gin.H{
			"RequestId":  genReqID(),
			"RecordId":   strconv.FormatUint(record.ID, 10),
			"SubDomain":  record.Host,
			"RecordType": record.RecordType,
			"Value":      record.Value,
			"TTL":        record.TTL,
		},
	})
}

func (h *TencentCompatHandler) DeleteRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var body struct {
		Domain   string `json:"Domain"`
		RecordId string `json:"RecordId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.writeError(c, "InvalidParameter", "Invalid request body")
		return
	}

	if body.RecordId == "" {
		h.writeError(c, "MissingParameter", "RecordId is required.")
		return
	}

	recordIDNum, err := strconv.ParseUint(body.RecordId, 10, 64)
	if err != nil {
		h.writeError(c, "InvalidParameter", "无效的记录ID")
		return
	}
	record, err := h.svc.GetRecord(recordIDNum)
	if err != nil {
		h.writeError(c, "InvalidRecordID.NotExist", "记录不存在")
		return
	}
	if err := h.checkRAMAccess(c, record.NodeID, ""); err != nil {
		h.writeError(c, "Forbidden", err.Error())
		return
	}

	err = h.svc.DeleteDomainRecord(userID, body.RecordId)
	if err != nil {
		h.writeError(c, "DomainRecordNotBelongToUser", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"Response": gin.H{
			"RequestId": genReqID(),
			"RecordId":  body.RecordId,
		},
	})
}

func (h *TencentCompatHandler) writeError(c *gin.Context, code, message string) {
	c.JSON(http.StatusOK, gin.H{
		"Response": gin.H{
			"Error": gin.H{
				"Code":    code,
				"Message": message,
			},
			"RequestId": genReqID(),
		},
	})
}