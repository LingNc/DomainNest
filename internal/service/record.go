package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

var allowedRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "ALIAS": true,
	"MX": true, "TXT": true, "CAA": true, "NS": true, "SRV": true,
}

func IsValidRecordType(t string) bool {
	return allowedRecordTypes[t]
}

type RecordService struct {
	db   *gorm.DB
	perm *PermissionService
	dom  *DomainService
}

func NewRecordService(db *gorm.DB, perm *PermissionService, dom *DomainService) *RecordService {
	return &RecordService{db: db, perm: perm, dom: dom}
}

// CheckPermission verifies the user has at least the given permission level on the node.
func (s *RecordService) CheckPermission(userID, nodeID uint64, level int) error {
	return s.perm.RequireLevel(userID, nodeID, level)
}

type RecordQuery struct {
	Host       string
	RecordType string
	Value      string
	Enabled    *bool
	SyncStatus string
	Page       int
	PageSize   int
}

type RecordListResult struct {
	Items    []model.DNSRecord `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

func (s *RecordService) ListRecords(nodeID, userID uint64, q RecordQuery) (*RecordListResult, error) {
	if err := s.perm.RequireLevel(userID, nodeID, 1); err != nil {
		return nil, err
	}

	query := s.db.Model(&model.DNSRecord{}).Where("node_id = ?", nodeID)

	// Apply permission-based filters for non-owner users
	var perm model.DomainPermission
	isOwner := false
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err == nil && node.OwnerID == userID {
		isOwner = true
	}
	if !isOwner {
		if err := s.db.Where("user_id = ? AND domain_node_id = ? AND status = 'active'", userID, nodeID).First(&perm).Error; err == nil {
			// Filter by allowed types
			if perm.AllowedTypes != "" && perm.AllowedTypes != "[]" {
				var types []string
				if jsonErr := jsonUnmarshal(perm.AllowedTypes, &types); jsonErr == nil && len(types) > 0 {
					query = query.Where("record_type IN ?", types)
				}
			}
			// Filter by host prefix (backward compat)
			if perm.HostPrefix != "" {
				query = query.Where("host LIKE ?", perm.HostPrefix+"%")
			}
		}
	}

	if q.Host != "" {
		query = query.Where("host LIKE ?", "%"+q.Host+"%")
	}
	if q.RecordType != "" {
		query = query.Where("record_type = ?", q.RecordType)
	}
	if q.Value != "" {
		query = query.Where("value LIKE ?", "%"+q.Value+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if q.SyncStatus != "" {
		query = query.Where("sync_status = ?", q.SyncStatus)
	}

	var total int64
	query.Count(&total)

	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 10000 {
		q.PageSize = 20
	}

	var records []model.DNSRecord
	query.Order("id ASC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&records)

	return &RecordListResult{Items: records, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// jsonUnmarshal is a helper to unmarshal JSON strings.
func jsonUnmarshal(s string, v interface{}) error {
	b := []byte(s)
	return json.Unmarshal(b, v)
}

func (s *RecordService) CreateRecord(nodeID, userID uint64, host, recordType, value string, ttl int, priority *int, line string, extraArgs ...interface{}) (*model.DNSRecord, error) {
	if !IsValidRecordType(recordType) {
		return nil, fmt.Errorf("不支持的记录类型: %s", recordType)
	}

	if err := s.perm.RequireLevel(userID, nodeID, 2); err != nil {
		return nil, err
	}

	if !s.perm.CanUseRecordType(userID, nodeID, recordType) {
		return nil, fmt.Errorf("您无权在该域名上创建 %s 记录", recordType)
	}

	if err := s.perm.ValidateIPValue(userID, nodeID, recordType, value); err != nil {
		return nil, err
	}

	if err := s.perm.ValidateHostRules(userID, nodeID, host); err != nil {
		return nil, err
	}

	if err := s.perm.ValidateDepth(userID, nodeID, host); err != nil {
		return nil, err
	}

	if err := validateRecordValue(recordType, value, priority); err != nil {
		return nil, err
	}

	// Host collision check: reject if host matches a materialized child node
	if host != "@" {
		var childNode model.DomainNode
		if err := s.db.Where("parent_id = ? AND host = ? AND deleted_at IS NULL", nodeID, host).
			First(&childNode).Error; err == nil {
			return nil, fmt.Errorf("主机名 '%s' 已被子节点 %s 占用，请在该节点下创建记录", host, childNode.FullDomain)
		}
	}

	if ttl == 0 {
		ttl = 600
	}
	if line == "" {
		line = "default"
	}

	// Extract optional providerRecordID from extraArgs
	var providerRecordID string
	if len(extraArgs) > 0 {
		if prid, ok := extraArgs[0].(string); ok {
			providerRecordID = prid
		}
	}

	syncStatus := "pending"
	if providerRecordID != "" {
		syncStatus = "synced"
	}

	record := &model.DNSRecord{
		NodeID:           nodeID,
		Host:             host,
		RecordType:       recordType,
		Value:            value,
		TTL:              ttl,
		Priority:         priority,
		Line:             line,
		Enabled:          true,
		SyncStatus:       syncStatus,
		ProviderRecordID: providerRecordID,
		CreatedBy:        userID,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RecordService) UpdateRecord(recordID, userID uint64, value string, ttl *int, priority *int) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("记录不存在")
	}

	if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
		return nil, err
	}

	if value != "" {
		if err := validateRecordValue(record.RecordType, value, priority); err != nil {
			return nil, err
		}
	}

	updates := map[string]interface{}{
		"sync_status": "pending",
	}
	if value != "" {
		updates["value"] = value
	}
	if ttl != nil {
		updates["ttl"] = *ttl
	}
	if priority != nil {
		updates["priority"] = *priority
	}

	if err := s.db.Model(&record).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.First(&record, recordID)
	return &record, nil
}

func (s *RecordService) DeleteRecord(recordID, userID uint64) error {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return errors.New("记录不存在")
	}

	if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
		return err
	}

	return s.db.Delete(&record).Error
}

func (s *RecordService) ToggleRecord(recordID, userID uint64, enabled bool) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("记录不存在")
	}

	if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
		return nil, err
	}

	syncStatus := "pending"
	if !enabled {
		syncStatus = "disabled"
	}

	updates := map[string]interface{}{
		"enabled":     enabled,
		"sync_status": syncStatus,
	}
	if err := s.db.Model(&record).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.First(&record, recordID)
	return &record, nil
}

func (s *RecordService) BatchDelete(recordIDs []uint64, userID uint64) error {
	for _, id := range recordIDs {
		var record model.DNSRecord
		if err := s.db.First(&record, id).Error; err != nil {
			return fmt.Errorf("记录 %d 不存在", id)
		}
		if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
			return fmt.Errorf("无权访问记录 %d", id)
		}
	}

	return s.db.Delete(&model.DNSRecord{}, "id IN ?", recordIDs).Error
}

func (s *RecordService) BatchToggle(recordIDs []uint64, userID uint64, enabled bool) error {
	for _, id := range recordIDs {
		var record model.DNSRecord
		if err := s.db.First(&record, id).Error; err != nil {
			return fmt.Errorf("记录 %d 不存在", id)
		}
		if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
			return fmt.Errorf("无权访问记录 %d", id)
		}
	}

	syncStatus := "pending"
	if !enabled {
		syncStatus = "disabled"
	}

	return s.db.Model(&model.DNSRecord{}).Where("id IN ?", recordIDs).
		Updates(map[string]interface{}{"enabled": enabled, "sync_status": syncStatus}).Error
}

func (s *RecordService) BatchUpdateGroupTag(recordIDs []uint64, userID uint64, groupTag string) error {
	for _, id := range recordIDs {
		var record model.DNSRecord
		if err := s.db.First(&record, id).Error; err != nil {
			return fmt.Errorf("记录 %d 不存在", id)
		}
		if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
			return fmt.Errorf("无权访问记录 %d", id)
		}
	}

	return s.db.Model(&model.DNSRecord{}).Where("id IN ?", recordIDs).
		Update("group_tag", groupTag).Error
}

func (s *RecordService) GetRecordByID(recordID uint64) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *RecordService) UpdateSyncStatus(recordID uint64, status, providerRecordID string) error {
	updates := map[string]interface{}{
		"sync_status": status,
	}
	if providerRecordID != "" {
		updates["provider_record_id"] = providerRecordID
	}
	return s.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).Updates(updates).Error
}

func (s *RecordService) FindRecordByNodeAndHost(nodeID uint64, host, recordType string) (*model.DNSRecord, error) {
	var record model.DNSRecord
	err := s.db.Where("node_id = ? AND host = ? AND record_type = ?", nodeID, host, recordType).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

type ExportRecord struct {
	Host       string `json:"host" csv:"host"`
	RecordType string `json:"record_type" csv:"record_type"`
	Value      string `json:"value" csv:"value"`
	TTL        int    `json:"ttl" csv:"ttl"`
	Priority   *int   `json:"priority,omitempty" csv:"priority"`
	Line       string `json:"line" csv:"line"`
	Enabled    bool   `json:"enabled" csv:"enabled"`
}

func (s *RecordService) ExportRecords(nodeID, userID uint64) ([]ExportRecord, error) {
	if err := s.perm.RequireLevel(userID, nodeID, 1); err != nil {
		return nil, err
	}

	var records []model.DNSRecord
	if err := s.db.Where("node_id = ?", nodeID).Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}

	exports := make([]ExportRecord, len(records))
	for i, r := range records {
		exports[i] = ExportRecord{
			Host:       r.Host,
			RecordType: r.RecordType,
			Value:      r.Value,
			TTL:        r.TTL,
			Priority:   r.Priority,
			Line:       r.Line,
			Enabled:    r.Enabled,
		}
	}
	return exports, nil
}

type ImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

func (s *RecordService) ImportRecords(nodeID, userID uint64, records []ExportRecord) (*ImportResult, error) {
	if err := s.perm.RequireLevel(userID, nodeID, 2); err != nil {
		return nil, err
	}

	result := &ImportResult{}
	for i, r := range records {
		if !IsValidRecordType(r.RecordType) {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: unsupported record type %s", i+1, r.RecordType))
			result.Skipped++
			continue
		}
		if err := validateRecordValue(r.RecordType, r.Value, r.Priority); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", i+1, err))
			result.Skipped++
			continue
		}

		ttl := r.TTL
		if ttl == 0 {
			ttl = 600
		}
		line := r.Line
		if line == "" {
			line = "default"
		}

		record := &model.DNSRecord{
			NodeID:     nodeID,
			Host:       r.Host,
			RecordType: r.RecordType,
			Value:      r.Value,
			TTL:        ttl,
			Priority:   r.Priority,
			Line:       line,
			Enabled:    r.Enabled,
			SyncStatus: "pending",
			CreatedBy:  userID,
		}
		if !r.Enabled {
			record.Enabled = false
			record.SyncStatus = "disabled"
		}

		if err := s.db.Create(record).Error; err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", i+1, err))
			result.Skipped++
		} else {
			result.Created++
		}
	}
	return result, nil
}

// TransferResult represents the outcome of transferring a single host's subdomain.
type TransferResult struct {
	Host        string `json:"host"`
	Status      string `json:"status"`                 // "transferred" | "error"
	NodeID      uint64 `json:"node_id,omitempty"`
	FullDomain  string `json:"full_domain,omitempty"`
	Action      string `json:"action,omitempty"`       // "materialized_and_transferred" | "transferred"
	RecordCount int    `json:"record_count,omitempty"`
	Error       string `json:"error,omitempty"`
}

// TransferRecordsByHost materializes implicit subdomains (if needed) and transfers
// ownership to targetUserID for each given host under the parent node.
func (s *RecordService) TransferRecordsByHost(parentID, currentUserID, targetUserID uint64, hosts []string) []TransferResult {
	results := make([]TransferResult, 0, len(hosts))

	// Validate parent node ownership (level 4)
	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		for _, host := range hosts {
			results = append(results, TransferResult{Host: host, Status: "error", Error: "父节点不存在"})
		}
		return results
	}
	if err := s.perm.RequireLevel(currentUserID, parentID, 4); err != nil {
		for _, host := range hosts {
			results = append(results, TransferResult{Host: host, Status: "error", Error: err.Error()})
		}
		return results
	}

	// Validate target user exists
	var targetUser model.User
	if err := s.db.First(&targetUser, targetUserID).Error; err != nil {
		for _, host := range hosts {
			results = append(results, TransferResult{Host: host, Status: "error", Error: "目标用户不存在"})
		}
		return results
	}

	for _, host := range hosts {
		r := s.transferSingleHost(parent, currentUserID, targetUserID, host)
		results = append(results, r)
	}

	return results
}

func (s *RecordService) transferSingleHost(parent model.DomainNode, currentUserID, targetUserID uint64, host string) TransferResult {
	fullDomain := host + "." + parent.FullDomain

	// Skip root host
	if host == "@" {
		return TransferResult{Host: host, Status: "error", Error: "无法转让根记录，请使用「转让域名」功能"}
	}

	// Count records for this host
	var recordCount int64
	s.db.Model(&model.DNSRecord{}).Where("node_id = ? AND host = ? AND deleted_at IS NULL", parent.ID, host).Count(&recordCount)

	// Look up existing materialized node
	var node model.DomainNode
	err := s.db.Where("full_domain = ?", fullDomain).First(&node).Error

	action := "transferred"
	if err != nil {
		// Node does not exist - materialize first
		if recordCount == 0 {
			return TransferResult{Host: host, Status: "error", Error: "该主机名下没有DNS记录，无法转换为节点"}
		}
		newNode, matErr := s.dom.MaterializeNode(parent.ID, host, currentUserID)
		if matErr != nil {
			return TransferResult{Host: host, Status: "error", Error: matErr.Error()}
		}
		node = *newNode
		action = "materialized_and_transferred"
	} else {
		// Node exists - check ownership
		if node.OwnerID == targetUserID {
			return TransferResult{Host: host, Status: "error", Error: "目标用户已拥有该子域名", NodeID: node.ID}
		}
		if node.OwnerID != currentUserID {
			return TransferResult{Host: host, Status: "error", Error: "您不拥有该子域名", NodeID: node.ID}
		}
	}

	// Transfer
	if _, transErr := s.dom.TransferNode(node.ID, currentUserID, targetUserID); transErr != nil {
		return TransferResult{Host: host, Status: "error", Error: transErr.Error(), NodeID: node.ID}
	}

	return TransferResult{
		Host:        host,
		Status:      "transferred",
		NodeID:      node.ID,
		FullDomain:  fullDomain,
		Action:      action,
		RecordCount: int(recordCount),
	}
}

// RenameGroupTag renames all records' group_tag from oldTag to newTag for a given node.
func (s *RecordService) RenameGroupTag(nodeID, userID uint64, oldTag, newTag string) (int64, error) {
	if err := s.perm.RequireLevel(userID, nodeID, 2); err != nil {
		return 0, err
	}
	if oldTag == "" || newTag == "" {
		return 0, fmt.Errorf("分组名称不能为空")
	}
	if oldTag == newTag {
		return 0, fmt.Errorf("新旧分组名称相同")
	}
	var count int64
	s.db.Model(&model.DNSRecord{}).Where("node_id = ? AND group_tag = ?", nodeID, newTag).Count(&count)
	if count > 0 {
		return 0, fmt.Errorf("分组 '%s' 已存在", newTag)
	}
	result := s.db.Model(&model.DNSRecord{}).
		Where("node_id = ? AND group_tag = ?", nodeID, oldTag).
		Update("group_tag", newTag)
	return result.RowsAffected, result.Error
}

// DeleteGroupTag removes the group_tag from all records in the specified group.
func (s *RecordService) DeleteGroupTag(nodeID, userID uint64, tag string) (int64, error) {
	if err := s.perm.RequireLevel(userID, nodeID, 2); err != nil {
		return 0, err
	}
	if tag == "" {
		return 0, fmt.Errorf("分组名称不能为空")
	}
	result := s.db.Model(&model.DNSRecord{}).
		Where("node_id = ? AND group_tag = ?", nodeID, tag).
		Update("group_tag", "")
	return result.RowsAffected, result.Error
}

func validateRecordValue(recordType, value string, priority *int) error {
	switch recordType {
	case "A":
		if net.ParseIP(value) == nil || !strings.Contains(value, ".") {
			return errors.New("A记录值必须是合法的IPv4地址")
		}
	case "AAAA":
		if net.ParseIP(value) == nil || !strings.Contains(value, ":") {
			return errors.New("AAAA记录值必须是合法的IPv6地址")
		}
	case "CNAME", "ALIAS", "NS":
		if !isValidDomainName(value) {
			return fmt.Errorf("%s记录值必须是合法的域名", recordType)
		}
	case "MX":
		if priority == nil {
			return errors.New("MX记录需要指定优先级")
		}
		if !isValidDomainName(value) {
			return errors.New("MX记录值必须是合法的域名")
		}
	case "SRV":
		parts := strings.Fields(value)
		if len(parts) != 4 {
			return errors.New("SRV记录值格式必须为 '优先级 权重 端口 目标'")
		}
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return errors.New("SRV优先级必须为数字")
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return errors.New("SRV权重必须为数字")
		}
		port, err := strconv.Atoi(parts[2])
		if err != nil {
			return errors.New("SRV端口必须为数字")
		}
		if port < 0 || port > 65535 {
			return errors.New("SRV端口范围为0-65535")
		}
		if !isValidDomainName(parts[3]) {
			return errors.New("SRV目标必须是合法的域名")
		}
	case "CAA":
		parts := strings.SplitN(value, " ", 3)
		if len(parts) != 3 {
			return errors.New("CAA记录值格式必须为 '标志 标签 值'")
		}
		flag, err := strconv.Atoi(parts[0])
		if err != nil || flag < 0 || flag > 255 {
			return errors.New("CAA标志必须为0-255的数字")
		}
	case "TXT":
		// TXT records accept any string, no validation needed
	}
	return nil
}

func isValidDomainName(name string) bool {
	if name == "" || len(name) > 253 {
		return false
	}
	name = strings.TrimSuffix(name, ".")
	parts := strings.Split(name, ".")
	if len(parts) < 1 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		if !isAlphaNumeric(part[0]) || !isAlphaNumeric(part[len(part)-1]) {
			return false
		}
	}
	return true
}

func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
