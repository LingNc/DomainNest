package service

import (
	"log"
	"time"

	"domainnest/internal/errs"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type TrashService struct {
	db              *gorm.DB
	providerService *ProviderService
}

func NewTrashService(db *gorm.DB, providerService *ProviderService) *TrashService {
	return &TrashService{db: db, providerService: providerService}
}

type TrashQuery struct {
	Host          string `form:"host"`
	RecordType    string `form:"record_type"`
	DomainID      uint64 `form:"domain_id"`
	TrashedAfter  string `form:"trashed_after"`
	TrashedBefore string `form:"trashed_before"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
}

type TrashListResult struct {
	Items    []TrashItem `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type TrashItem struct {
	ID         uint64     `json:"id"`
	NodeID     uint64     `json:"node_id"`
	Host       string     `json:"host"`
	RecordType string     `json:"record_type"`
	Value      string     `json:"value"`
	TTL        int        `json:"ttl"`
	TrashedAt  *time.Time `json:"trashed_at"`
	FullDomain string     `json:"full_domain" gorm:"->"`
	Source     string     `json:"source"`
}

func (s *TrashService) ListTrash(userID uint64, q TrashQuery) (*TrashListResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	// Base query: trashed records belonging to domains the user owns
	base := s.db.Unscoped().Model(&model.DNSRecord{}).
		Joins("JOIN domain_nodes ON domain_nodes.id = dns_records.node_id").
		Where("dns_records.trashed_at IS NOT NULL").
		Where("domain_nodes.owner_id = ?", userID)

	if q.Host != "" {
		base = base.Where("dns_records.host LIKE ?", "%"+q.Host+"%")
	}
	if q.RecordType != "" {
		base = base.Where("dns_records.record_type = ?", q.RecordType)
	}
	if q.DomainID > 0 {
		base = base.Where("dns_records.node_id = ?", q.DomainID)
	}
	if q.TrashedAfter != "" {
		base = base.Where("dns_records.trashed_at >= ?", q.TrashedAfter)
	}
	if q.TrashedBefore != "" {
		base = base.Where("dns_records.trashed_at <= ?", q.TrashedBefore)
	}

	var total int64
	base.Count(&total)

	var items []TrashItem
	err := base.Select("dns_records.*, domain_nodes.full_domain").
		Order("dns_records.trashed_at DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&items).Error
	if err != nil {
		return nil, errs.Wrap(errs.InternalError, err)
	}

	return &TrashListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

// TrashRecord moves a record to the trash. Provider-side deletion happens here.
func (s *TrashService) TrashRecord(recordID, userID uint64) error {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return errs.New(errs.RecordNotFound, "记录不存在")
	}

	// Verify ownership through node
	var node model.DomainNode
	if err := s.db.Unscoped().First(&node, record.NodeID).Error; err != nil {
		return errs.New(errs.DomainNotFound, "域名不存在")
	}
	if node.OwnerID != userID {
		return errs.New(errs.NoPermission, "无权操作此记录")
	}

	// Delete from provider first (if it has a provider_record_id)
	if record.ProviderRecordID != "" && node.ProviderID != nil {
		client, err := s.providerService.GetDNSProvider(*node.ProviderID)
		if err == nil {
			client.DeleteRecord(record.ProviderRecordID)
			// Ignore provider error — record may already be gone
		}
	}

	now := time.Now()
	return s.db.Unscoped().Model(&record).Updates(map[string]interface{}{
		"trashed_at":         now,
		"deleted_at":         now,
		"provider_record_id": "",
		"sync_status":        "trashed",
	}).Error
}

// RestoreRecord restores a trashed record and sets it to pending sync.
func (s *TrashService) RestoreRecord(recordID, userID uint64) error {
	var record model.DNSRecord
	if err := s.db.Unscoped().First(&record, recordID).Error; err != nil {
		return errs.New(errs.RecordNotFound, "记录不存在")
	}
	if record.TrashedAt == nil {
		return errs.New(errs.RecordNotInTrash, "记录不在回收站中")
	}

	// Verify ownership
	var node model.DomainNode
	if err := s.db.Unscoped().First(&node, record.NodeID).Error; err != nil {
		return errs.New(errs.DomainNotFound, "域名不存在")
	}
	if node.OwnerID != userID {
		return errs.New(errs.NoPermission, "无权操作此记录")
	}

	return s.db.Unscoped().Model(&record).Updates(map[string]interface{}{
		"trashed_at":         nil,
		"deleted_at":         nil,
		"provider_record_id": "",
		"sync_status":        "pending",
	}).Error
}

// PermanentDelete hard-deletes a trashed record.
func (s *TrashService) PermanentDelete(recordID, userID uint64) error {
	var record model.DNSRecord
	if err := s.db.Unscoped().First(&record, recordID).Error; err != nil {
		return errs.New(errs.RecordNotFound, "记录不存在")
	}
	if record.TrashedAt == nil {
		return errs.New(errs.RecordNotInTrash, "记录不在回收站中")
	}

	// Verify ownership
	var node model.DomainNode
	if err := s.db.Unscoped().First(&node, record.NodeID).Error; err != nil {
		return errs.New(errs.DomainNotFound, "域名不存在")
	}
	if node.OwnerID != userID {
		return errs.New(errs.NoPermission, "无权操作此记录")
	}

	return s.db.Unscoped().Delete(&record).Error
}

// EmptyTrash permanently deletes all trashed records for a user.
func (s *TrashService) EmptyTrash(userID uint64) (int64, error) {
	result := s.db.Unscoped().
		Where("trashed_at IS NOT NULL").
		Where("node_id IN (SELECT id FROM domain_nodes WHERE owner_id = ?)", userID).
		Delete(&model.DNSRecord{})
	return result.RowsAffected, result.Error
}

// BatchTrash moves multiple records to trash.
func (s *TrashService) BatchTrash(recordIDs []uint64, userID uint64) (trashed int, failed int) {
	for _, id := range recordIDs {
		if err := s.TrashRecord(id, userID); err != nil {
			failed++
		} else {
			trashed++
		}
	}
	return
}

// BatchRestore restores multiple trashed records.
func (s *TrashService) BatchRestore(recordIDs []uint64, userID uint64) (restored int, failed int) {
	for _, id := range recordIDs {
		if err := s.RestoreRecord(id, userID); err != nil {
			failed++
		} else {
			restored++
		}
	}
	return
}

// PurgeExpired permanently deletes records trashed more than 30 days ago.
func (s *TrashService) PurgeExpired() int64 {
	cutoff := time.Now().AddDate(0, 0, -30)
	result := s.db.Unscoped().
		Where("trashed_at IS NOT NULL AND trashed_at < ?", cutoff).
		Delete(&model.DNSRecord{})
	if result.RowsAffected > 0 {
		log.Printf("[trash] Purged %d expired records", result.RowsAffected)
	}
	return result.RowsAffected
}
