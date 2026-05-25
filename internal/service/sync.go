package service

import (
	"fmt"
	"log"
	"math"
	"time"

	"domainnest/internal/config"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type SyncService struct {
	db          *gorm.DB
	ddnsService *DDNSService
	cfg         *config.SyncConfig
	stopCh      chan struct{}
}

func NewSyncService(db *gorm.DB, ddnsService *DDNSService, cfg *config.SyncConfig) *SyncService {
	s := &SyncService{
		db:          db,
		ddnsService: ddnsService,
		cfg:         cfg,
		stopCh:      make(chan struct{}),
	}
	s.applyDefaults()
	return s
}

func (s *SyncService) applyDefaults() {
	s.cfg.Enabled = true
	if s.cfg.Interval <= 0 {
		s.cfg.Interval = 60
	}
	if s.cfg.BatchSize <= 0 {
		s.cfg.BatchSize = 10
	}
	if s.cfg.MaxRetries <= 0 {
		s.cfg.MaxRetries = 5
	}
	if s.cfg.BaseBackoff <= 0 {
		s.cfg.BaseBackoff = 30
	}
	if s.cfg.MaxBackoff <= 0 {
		s.cfg.MaxBackoff = 3600
	}
}

func (s *SyncService) Start() {
	if !s.cfg.Enabled {
		log.Println("DNS sync worker disabled")
		return
	}
	log.Printf("DNS sync worker started (interval=%ds, batchSize=%d)", s.cfg.Interval, s.cfg.BatchSize)
	go s.loop()
}

func (s *SyncService) Stop() {
	close(s.stopCh)
}

func (s *SyncService) loop() {
	ticker := time.NewTicker(time.Duration(s.cfg.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			log.Println("DNS sync worker stopped")
			return
		case <-ticker.C:
			s.processBatch()
		}
	}
}

func (s *SyncService) processBatch() {
	var records []model.DNSRecord
	now := time.Now()
	s.db.Joins("JOIN domain_nodes ON domain_nodes.id = dns_records.node_id AND domain_nodes.deleted_at IS NULL").
		Where("domain_nodes.status = ?", "active").
		Where("(dns_records.sync_status = ? OR dns_records.sync_status = ?) AND (dns_records.next_sync_at IS NULL OR dns_records.next_sync_at <= ?)",
			"pending", "failed", now).
		Order("dns_records.sync_attempts ASC").
		Limit(s.cfg.BatchSize).
		Find(&records)

	for i := range records {
		recordID := records[i].ID
		err := s.ddnsService.SyncRecord(recordID)
		if err != nil {
			s.handleSyncFailure(&records[i], err)
		} else {
			s.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).Update("sync_status", "synced")
			s.db.Create(&model.SyncLog{
				RecordID:   recordID,
				Action:     "sync",
				Status:     "success",
				ProviderID: records[i].ProviderRecordID,
			})
		}
	}
}

func (s *SyncService) handleSyncFailure(record *model.DNSRecord, syncErr error) {
	attempts := record.SyncAttempts + 1
	backoff := int(math.Min(
		float64(s.cfg.BaseBackoff)*math.Pow(2, float64(record.SyncAttempts)),
		float64(s.cfg.MaxBackoff),
	))

	updates := map[string]interface{}{
		"sync_attempts":   attempts,
		"last_sync_error": syncErr.Error(),
	}
	if attempts < s.cfg.MaxRetries {
		nextSync := time.Now().Add(time.Duration(backoff) * time.Second)
		updates["next_sync_at"] = nextSync
	} else {
		updates["next_sync_at"] = nil
		updates["sync_status"] = "failed"
	}

	s.db.Model(&model.DNSRecord{}).Where("id = ?", record.ID).Updates(updates)

	s.db.Create(&model.SyncLog{
		RecordID:   record.ID,
		Action:     "sync",
		Status:     "failed",
		Error:      syncErr.Error(),
		ProviderID: record.ProviderRecordID,
	})
}

func (s *SyncService) ManualSync(recordIDs []uint64) (synced int, failed int) {
	s.db.Model(&model.DNSRecord{}).Where("id IN ?", recordIDs).
		Updates(map[string]interface{}{
			"sync_status":     "pending",
			"sync_attempts":   0,
			"next_sync_at":    nil,
			"last_sync_error": "",
		})

	var records []model.DNSRecord
	s.db.Where("id IN ?", recordIDs).Find(&records)

	for i := range records {
		if err := s.ddnsService.SyncRecord(records[i].ID); err != nil {
			failed++
			s.handleSyncFailure(&records[i], err)
		} else {
			synced++
			s.db.Model(&model.DNSRecord{}).Where("id = ?", records[i].ID).Update("sync_status", "synced")
			s.db.Create(&model.SyncLog{
				RecordID:   records[i].ID,
				Action:     "sync",
				Status:     "success",
				ProviderID: records[i].ProviderRecordID,
			})
		}
	}
	return
}

func (s *SyncService) GetSyncLogs(nodeID uint64, page, pageSize int) ([]model.SyncLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	query := s.db.Model(&model.SyncLog{}).
		Joins("JOIN dns_records ON dns_records.id = sync_logs.record_id").
		Where("dns_records.node_id = ?", nodeID)
	query.Count(&total)

	var logs []model.SyncLog
	err := query.Order("sync_logs.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("获取同步日志失败: %w", err)
	}

	return logs, total, nil
}
