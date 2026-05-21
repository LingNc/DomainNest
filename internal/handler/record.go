package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecordHandler struct {
	recordService *service.RecordService
	db            *gorm.DB
}

func NewRecordHandler(recordService *service.RecordService, db *gorm.DB) *RecordHandler {
	return &RecordHandler{recordService: recordService, db: db}
}

func (h *RecordHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var enabled *bool
	if v := c.Query("enabled"); v != "" {
		b := v == "true" || v == "1"
		enabled = &b
	}

	q := service.RecordQuery{
		Host:       c.Query("host"),
		RecordType: c.Query("record_type"),
		Value:      c.Query("value"),
		Enabled:    enabled,
		SyncStatus: c.Query("sync_status"),
		Page:       page,
		PageSize:   pageSize,
	}

	result, err := h.recordService.ListRecords(nodeID, userID, q)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

func (h *RecordHandler) Create(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var req struct {
		Host       string `json:"host" binding:"required"`
		RecordType string `json:"record_type" binding:"required"`
		Value      string `json:"value" binding:"required"`
		TTL        int    `json:"ttl"`
		Priority   *int   `json:"priority"`
		Line       string `json:"line"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.CreateRecord(nodeID, userID, req.Host, req.RecordType, req.Value, req.TTL, req.Priority, req.Line)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "create_record", "dns_record", &record.ID,
		map[string]interface{}{"host": record.Host, "type": record.RecordType, "value": record.Value}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) Update(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	var req struct {
		Value    string `json:"value"`
		TTL      *int   `json:"ttl"`
		Priority *int   `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.UpdateRecord(recordID, userID, req.Value, req.TTL, req.Priority)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "update_record", "dns_record", &recordID,
		map[string]interface{}{"value": record.Value}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	if err := h.recordService.DeleteRecord(recordID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "delete_record", "dns_record", &recordID,
		nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *RecordHandler) Toggle(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的记录ID"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	record, err := h.recordService.ToggleRecord(recordID, userID, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	action := "enable_record"
	if !req.Enabled {
		action = "disable_record"
	}
	middleware.LogOperation(h.db, userID, action, "dns_record", &recordID,
		map[string]interface{}{"enabled": req.Enabled}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": record})
}

func (h *RecordHandler) BatchDelete(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.recordService.BatchDelete(req.IDs, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "batch_delete_records", "dns_record", nil,
		map[string]interface{}{"ids": req.IDs}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *RecordHandler) BatchToggle(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		IDs     []uint64 `json:"ids" binding:"required"`
		Enabled bool     `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.recordService.BatchToggle(req.IDs, userID, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	action := "batch_enable_records"
	if !req.Enabled {
		action = "batch_disable_records"
	}
	middleware.LogOperation(h.db, userID, action, "dns_record", nil,
		map[string]interface{}{"ids": req.IDs, "enabled": req.Enabled}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *RecordHandler) TransferByHost(c *gin.Context) {
	userID := c.GetUint64("user_id")
	parentID, err := strconv.ParseUint(c.Param("parentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	var req struct {
		Hosts        []string `json:"hosts" binding:"required,min=1"`
		TargetUserID uint64   `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	results := h.recordService.TransferRecordsByHost(parentID, userID, req.TargetUserID, req.Hosts)

	middleware.LogOperation(h.db, userID, "transfer_records_by_host", "domain_node", &parentID,
		map[string]interface{}{"hosts": req.Hosts, "target_user_id": req.TargetUserID}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"results": results}})
}

func (h *RecordHandler) Export(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	format := c.DefaultQuery("format", "json")

	records, err := h.recordService.ExportRecords(nodeID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=records.csv")
		w := csv.NewWriter(c.Writer)
		w.Write([]string{"host", "record_type", "value", "ttl", "priority", "line", "enabled"})
		for _, r := range records {
			priority := ""
			if r.Priority != nil {
				priority = strconv.Itoa(*r.Priority)
			}
			w.Write([]string{r.Host, r.RecordType, r.Value, strconv.Itoa(r.TTL), priority, r.Line, strconv.FormatBool(r.Enabled)})
		}
		w.Flush()
	default:
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": records})
	}
}

func (h *RecordHandler) Import(c *gin.Context) {
	userID := c.GetUint64("user_id")
	nodeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的节点ID"})
		return
	}

	format := c.DefaultQuery("format", "json")

	var records []service.ExportRecord

	switch format {
	case "csv":
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请上传CSV文件"})
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		rows, err := reader.ReadAll()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "读取CSV文件失败: " + err.Error()})
			return
		}
		if len(rows) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "CSV文件必须包含表头和至少一行数据"})
			return
		}

		for _, row := range rows[1:] {
			if len(row) < 4 {
				continue
			}
			ttl, _ := strconv.Atoi(row[3])
			var priority *int
			if len(row) > 4 && row[4] != "" {
				p, err := strconv.Atoi(row[4])
				if err == nil {
					priority = &p
				}
			}
			line := ""
			if len(row) > 5 {
				line = row[5]
			}
			enabled := true
			if len(row) > 6 && (row[6] == "false" || row[6] == "0") {
				enabled = false
			}
			records = append(records, service.ExportRecord{
				Host:       row[0],
				RecordType: row[1],
				Value:      row[2],
				TTL:        ttl,
				Priority:   priority,
				Line:       line,
				Enabled:    enabled,
			})
		}
	default:
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "读取请求体失败"})
			return
		}
		if err := json.Unmarshal(body, &records); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "JSON格式无效: " + err.Error()})
			return
		}
	}

	if len(records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有要导入的记录"})
		return
	}

	result, err := h.recordService.ImportRecords(nodeID, userID, records)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	middleware.LogOperation(h.db, userID, "import_records", "dns_record", nil,
		map[string]interface{}{"created": result.Created, "skipped": result.Skipped}, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}
