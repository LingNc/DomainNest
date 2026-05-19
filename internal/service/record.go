package service

import (
	"errors"
	"fmt"
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
	db  *gorm.DB
	perm *PermissionService
}

func NewRecordService(db *gorm.DB, perm *PermissionService) *RecordService {
	return &RecordService{db: db, perm: perm}
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
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	var records []model.DNSRecord
	query.Order("id ASC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&records)

	return &RecordListResult{Items: records, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (s *RecordService) CreateRecord(nodeID, userID uint64, host, recordType, value string, ttl int, priority *int, line string) (*model.DNSRecord, error) {
	if !IsValidRecordType(recordType) {
		return nil, fmt.Errorf("unsupported record type: %s", recordType)
	}

	if err := s.perm.RequireLevel(userID, nodeID, 2); err != nil {
		return nil, err
	}

	if !s.perm.CanUseRecordType(userID, nodeID, recordType) {
		return nil, fmt.Errorf("you are not allowed to create %s records on this domain", recordType)
	}

	if err := s.perm.ValidateIPValue(userID, nodeID, recordType, value); err != nil {
		return nil, err
	}

	if err := s.perm.ValidateHostPrefix(userID, nodeID, host); err != nil {
		return nil, err
	}

	if err := s.perm.ValidateDepth(userID, nodeID, host); err != nil {
		return nil, err
	}

	if err := validateRecordValue(recordType, value, priority); err != nil {
		return nil, err
	}

	if ttl == 0 {
		ttl = 600
	}
	if line == "" {
		line = "default"
	}

	record := &model.DNSRecord{
		NodeID:     nodeID,
		Host:       host,
		RecordType: recordType,
		Value:      value,
		TTL:        ttl,
		Priority:   priority,
		Line:       line,
		Enabled:    true,
		SyncStatus: "pending",
		CreatedBy:  userID,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RecordService) UpdateRecord(recordID, userID uint64, value string, ttl *int, priority *int) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("record not found")
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
		return errors.New("record not found")
	}

	if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
		return err
	}

	return s.db.Delete(&record).Error
}

func (s *RecordService) ToggleRecord(recordID, userID uint64, enabled bool) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("record not found")
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
			return fmt.Errorf("record %d not found", id)
		}
		if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
			return fmt.Errorf("access denied for record %d", id)
		}
	}

	return s.db.Delete(&model.DNSRecord{}, "id IN ?", recordIDs).Error
}

func (s *RecordService) BatchToggle(recordIDs []uint64, userID uint64, enabled bool) error {
	for _, id := range recordIDs {
		var record model.DNSRecord
		if err := s.db.First(&record, id).Error; err != nil {
			return fmt.Errorf("record %d not found", id)
		}
		if err := s.perm.RequireLevel(userID, record.NodeID, 2); err != nil {
			return fmt.Errorf("access denied for record %d", id)
		}
	}

	syncStatus := "pending"
	if !enabled {
		syncStatus = "disabled"
	}

	return s.db.Model(&model.DNSRecord{}).Where("id IN ?", recordIDs).
		Updates(map[string]interface{}{"enabled": enabled, "sync_status": syncStatus}).Error
}

func (s *RecordService) GetRecordByID(recordID uint64) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *RecordService) UpdateSyncStatus(recordID uint64, status, aliyunRecordID string) error {
	updates := map[string]interface{}{
		"sync_status": status,
	}
	if aliyunRecordID != "" {
		updates["aliyun_record_id"] = aliyunRecordID
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

func validateRecordValue(recordType, value string, priority *int) error {
	switch recordType {
	case "MX":
		if priority == nil {
			return errors.New("MX records require a priority value")
		}
	case "SRV":
		parts := strings.Fields(value)
		if len(parts) != 4 {
			return errors.New("SRV value must be 'priority weight port target'")
		}
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return errors.New("SRV priority must be a number")
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return errors.New("SRV weight must be a number")
		}
		if _, err := strconv.Atoi(parts[2]); err != nil {
			return errors.New("SRV port must be a number")
		}
	case "CAA":
		parts := strings.SplitN(value, " ", 3)
		if len(parts) != 3 {
			return errors.New("CAA value must be 'flag tag value'")
		}
	}
	return nil
}
