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

// DomainSyncer defines the interface for pulling from providers
type DomainSyncer interface {
	SyncFromProvider(nodeID, userID uint64) error
	GetDomainNodesWithProvider() ([]model.DomainNode, error)
}

type SyncService struct {
	db          *gorm.DB
	ddnsService *DDNSService
	domainSvc   DomainSyncer
	cfg         *config.SyncConfig
	stopCh      chan struct{}
	pullCounter int
	verifyCounter int
}

func NewSyncService(db *gorm.DB, ddnsService *DDNSService, domainSvc DomainSyncer, cfg *config.SyncConfig) *SyncService {
	s := &SyncService{
		db:          db,
		ddnsService: ddnsService,
		domainSvc:   domainSvc,
		cfg:         cfg,
		stopCh:      make(chan struct{}),
		pullCounter: 0,
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
			s.pullCounter++
			if s.pullCounter >= 10 {
				s.pullCounter = 0
				s.pullFromProviders()
			}
			s.verifyCounter++
			if s.verifyCounter >= 3 {
				s.verifyCounter = 0
				s.verifySyncedRecords()
			}
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

func (s *SyncService) pullFromProviders() {
	if s.domainSvc == nil {
		return
	}
	nodes, err := s.domainSvc.GetDomainNodesWithProvider()
	if err != nil {
		log.Printf("获取绑定服务商的域名节点失败: %v", err)
		return
	}
	for _, node := range nodes {
		if err := s.domainSvc.SyncFromProvider(node.ID, node.OwnerID); err != nil {
			log.Printf("同步域名 %s 从服务商失败: %v", node.FullDomain, err)
		}
	}
}

func (s *SyncService) verifySyncedRecords() {
	if s.domainSvc == nil {
		return
	}
	nodes, err := s.domainSvc.GetDomainNodesWithProvider()
	if err != nil {
		log.Printf("verifySyncedRecords: 获取域名节点失败: %v", err)
		return
	}

	var verified, cleared int
	for _, node := range nodes {
		if node.ProviderID == nil {
			continue
		}
		p, err := s.ddnsService.GetProviderForNode(node.ID)
		if err != nil {
			continue
		}
		providerRecords, err := p.ListRecords(node.FullDomain)
		if err != nil {
			continue
		}

		providerIDs := make(map[string]bool, len(providerRecords))
		for _, pr := range providerRecords {
			providerIDs[pr.RecordID] = true
		}

		var syncedRecords []model.DNSRecord
		s.db.Where("node_id = ? AND sync_status = ? AND provider_record_id != '' AND deleted_at IS NULL",
			node.ID, "synced").Find(&syncedRecords)

		for _, rec := range syncedRecords {
			verified++
			if !providerIDs[rec.ProviderRecordID] {
				cleared++
				s.db.Model(&model.DNSRecord{}).Where("id = ?", rec.ID).Updates(map[string]interface{}{
					"sync_status":     "pending",
					"provider_record_id": "",
					"sync_attempts":   0,
					"next_sync_at":    nil,
					"last_sync_error": "record missing on provider",
				})
			}
		}
	}
	if cleared > 0 {
		log.Printf("verifySyncedRecords: verified %d records, marked %d for re-sync", verified, cleared)
	}
}
