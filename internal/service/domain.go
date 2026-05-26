package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type DomainService struct {
	db             *gorm.DB
	perm           *PermissionService
	recordSvc      *RecordService
	providerSvc    *ProviderService
}

func NewDomainService(db *gorm.DB, perm *PermissionService) *DomainService {
	return &DomainService{db: db, perm: perm}
}

func (s *DomainService) SetRecordService(recordSvc *RecordService) {
	s.recordSvc = recordSvc
}

func (s *DomainService) SetProviderService(providerSvc *ProviderService) {
	s.providerSvc = providerSvc
}

// DomainConflictError carries the existing node's status when a domain already exists.
type DomainConflictError struct {
	Status string
	Msg    string
}

func (e *DomainConflictError) Error() string {
	return e.Msg
}

func (s *DomainService) CreateNode(parentID uint64, host string, ownerID uint64) (*model.DomainNode, error) {
	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return nil, errors.New("父节点不存在")
	}

	if err := s.perm.RequireLevel(ownerID, parentID, 2); err != nil {
		return nil, err
	}

	fullDomain := host + "." + parent.FullDomain

	var existing model.DomainNode
	if err := s.db.Preload("Owner").Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		return nil, &DomainConflictError{Status: existing.Status, Msg: fmt.Sprintf("域名 %s 已存在，当前归属于用户 %s", existing.FullDomain, existing.Owner.Username)}
	}

	node := &model.DomainNode{
		Host:       host,
		FullDomain: fullDomain,
		ParentID:   &parentID,
		OwnerID:    ownerID,
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}

	return node, nil
}

func (s *DomainService) GetUserNodes(userID uint64) ([]model.DomainNode, error) {
	// Get all accessible domain IDs (owned + delegated)
	accessibleIDs, err := s.perm.AccessibleDomainIDs(userID)
	if err != nil {
		return nil, err
	}
	if len(accessibleIDs) == 0 {
		return nil, nil
	}

	var nodes []model.DomainNode
	err = s.db.Where("id IN ?", accessibleIDs).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN ?", accessibleIDs)
		}).
		Preload("Owner").
		Preload("Claimer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id,username,nickname,avatar")
		}).
		Find(&nodes).Error

	if err != nil {
		return nil, err
	}

	// Populate RecordsCount using permission-filtered record count
	// Also count for preloaded children (they are not in nodes array)
	for i := range nodes {
		cnt, _ := s.recordSvc.CountAccessibleRecords(nodes[i].ID, userID)
		nodes[i].RecordsCount = cnt
		// Count for children
		for j := range nodes[i].Children {
			childCnt, _ := s.recordSvc.CountAccessibleRecords(nodes[i].Children[j].ID, userID)
			nodes[i].Children[j].RecordsCount = childCnt
		}
	}

	var roots []model.DomainNode
	for _, n := range nodes {
		if n.ParentID == nil {
			roots = append(roots, n)
		} else {
			isRoot := true
			for _, m := range nodes {
				if m.ID == *n.ParentID {
					isRoot = false
					break
				}
			}
			if isRoot {
				roots = append(roots, n)
			}
		}
	}

	return roots, err
}

func (s *DomainService) GetNode(nodeID, userID uint64) (*model.DomainNode, error) {
	if err := s.perm.RequireLevel(userID, nodeID, 1); err != nil {
		return nil, err
	}

	var node model.DomainNode
	if err := s.db.Preload("Children").Preload("Records").
		Preload("Owner", func(db *gorm.DB) *gorm.DB { return db.Select("id,username,nickname,avatar") }).
		Preload("Claimer", func(db *gorm.DB) *gorm.DB { return db.Select("id,username,nickname,avatar") }).
		First(&node, nodeID).Error; err != nil {
		return nil, errors.New("域名节点不存在")
	}
	return &node, nil
}

func (s *DomainService) FindNodeByDomain(domain string, userID uint64) (*model.DomainNode, string, error) {
	accessibleIDs, err := s.perm.AccessibleDomainIDs(userID)
	if err != nil {
		return nil, "", err
	}
	if len(accessibleIDs) == 0 {
		return nil, "", errors.New("域名不存在或无访问权限")
	}

	var node model.DomainNode
	err = s.db.Where("id IN ? AND (full_domain = ? OR ? LIKE CONCAT('%.', full_domain))",
		accessibleIDs, domain, domain).
		Order("LENGTH(full_domain) DESC").
		First(&node).Error

	if err != nil {
		return nil, "", errors.New("域名不存在或无访问权限")
	}

	var rr string
	if node.FullDomain == domain {
		rr = "@"
	} else {
		suffix := "." + node.FullDomain
		rr = domain[:len(domain)-len(suffix)]
	}

	return &node, rr, nil
}

// TransferNodeResult holds the outcome of a domain transfer, including info
// about existing delegations the new owner should be notified about.
type TransferNodeResult struct {
	DelegationCount int
	DelegatedDomains []string
}

func (s *DomainService) TransferNode(nodeID, ownerID, targetUserID uint64) (*TransferNodeResult, error) {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return nil, errors.New("节点不存在")
	}

	if err := s.perm.RequireLevel(ownerID, nodeID, 4); err != nil {
		return nil, err
	}

	var result TransferNodeResult

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var nodeIDs []uint64
		err := tx.Raw(`
			WITH RECURSIVE subtree AS (
				SELECT id FROM domain_nodes WHERE id = ? AND deleted_at IS NULL
				UNION ALL
				SELECT dn.id FROM domain_nodes dn JOIN subtree s ON dn.parent_id = s.id WHERE dn.deleted_at IS NULL
			)
			SELECT id FROM subtree
		`, nodeID).Scan(&nodeIDs).Error
		if err != nil {
			return fmt.Errorf("failed to find subtree: %w", err)
		}

		if err := tx.Model(&model.DomainNode{}).Where("id IN ? AND owner_id = ?", nodeIDs, ownerID).Update("owner_id", targetUserID).Error; err != nil {
			return fmt.Errorf("failed to transfer nodes: %w", err)
		}

		// Log transfers
		for _, nid := range nodeIDs {
			tx.Create(&model.DomainTransferLog{
				NodeID:     nid,
				FromUserID: node.OwnerID,
				ToUserID:   targetUserID,
			})
		}

		// Collect delegation info for the new owner
		var permCount int64
		tx.Model(&model.DomainPermission{}).Where("domain_node_id IN ?", nodeIDs).Count(&permCount)
		result.DelegationCount = int(permCount)

		if permCount > 0 {
			tx.Model(&model.DomainNode{}).Where("id IN ?", nodeIDs).
				Pluck("full_domain", &result.DelegatedDomains)
		}

		return nil
	})

	return &result, err
}

// deleteProviderRecords deletes platform-managed DNS records from their providers
// and marks them as trashed. Returns a map of providerID -> error for any failures.
func (s *DomainService) deleteProviderRecords(nodeIDs []uint64) error {
	if s.providerSvc == nil {
		return nil
	}

	// Collect all platform records with provider_record_id across all nodes
	var records []model.DNSRecord
	if err := s.db.Unscoped().
		Where("node_id IN ? AND deleted_at IS NULL AND provider_record_id != ''", nodeIDs).
		Find(&records).Error; err != nil {
		return fmt.Errorf("查询DNS记录失败: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	// Group records by provider_id (node's ProviderID)
	type providerKey struct {
		providerID uint64
		recordIDs  []uint64
		prIDs      []string
	}
	byProvider := make(map[uint64]*providerKey)

	for _, rec := range records {
		var node model.DomainNode
		if err := s.db.Unscoped().First(&node, rec.NodeID).Error; err != nil {
			continue
		}
		var pid uint64
		if node.ProviderID != nil {
			pid = *node.ProviderID
		} else if node.ArchivedProviderID != nil {
			pid = *node.ArchivedProviderID
		}
		if pid == 0 {
			continue
		}

		if _, ok := byProvider[pid]; !ok {
			byProvider[pid] = &providerKey{providerID: pid}
		}
		byProvider[pid].recordIDs = append(byProvider[pid].recordIDs, rec.ID)
		byProvider[pid].prIDs = append(byProvider[pid].prIDs, rec.ProviderRecordID)
	}

	// Delete from each provider
	for pid, pk := range byProvider {
		client, err := s.providerSvc.GetDNSProvider(pid)
		if err != nil {
			// Try archived provider lookup
			client, err = s.providerSvc.GetDNSProviderArchived(pid)
		}
		if err != nil {
			continue // skip, record stays as-is
		}
		for _, prID := range pk.prIDs {
			client.DeleteRecord(prID)
		}
	}

	// Mark all as trashed
	now := time.Now()
	return s.db.Unscoped().Model(&model.DNSRecord{}).
		Where("id IN ?", func() []uint64 {
			ids := make([]uint64, 0)
			for _, pk := range byProvider {
				ids = append(ids, pk.recordIDs...)
			}
			return ids
		}()).
		Updates(map[string]interface{}{
			"trashed_at":         now,
			"deleted_at":         now,
			"provider_record_id": "",
			"sync_status":        "trashed",
		}).Error
}

// getSubtreeNodeIDs returns all node IDs in the subtree rooted at nodeID (including nodeID itself).
func (s *DomainService) getSubtreeNodeIDs(nodeID uint64) ([]uint64, error) {
	var nodeIDs []uint64
	err := s.db.Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM domain_nodes WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT dn.id FROM domain_nodes dn
			JOIN subtree s ON dn.parent_id = s.id
			WHERE dn.deleted_at IS NULL
		) SELECT id FROM subtree
	`, nodeID).Scan(&nodeIDs).Error
	return nodeIDs, err
}

func (s *DomainService) DeleteNode(nodeID, userID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	if err := s.perm.RequireLevel(userID, nodeID, 4); err != nil {
		return err
	}

	now := time.Now()

	// Root domain: non-claimer returns to claimer; claimer proceeds with deletion
	if node.ParentID == nil {
		if node.Status != "archived" {
			return errors.New("根域名请先归档后再删除")
		}
		claimerID := node.OwnerID
		if node.ClaimerID != nil {
			claimerID = *node.ClaimerID
		}
		if userID != claimerID {
			// Non-claimer deleting: return ownership to claimer, un-archive
			return s.db.Transaction(func(tx *gorm.DB) error {
				// Platform records: trigger sync to delete from provider, then move to trash
				if err := tx.Model(&model.DNSRecord{}).
					Where("node_id = ? AND deleted_at IS NULL AND provider_record_id != ''", nodeID).
					Updates(map[string]interface{}{
						"sync_status": "pending",
						"enabled":     false,
					}).Error; err != nil {
					return fmt.Errorf("标记待删除失败: %w", err)
				}
				if err := tx.Model(&model.DNSRecord{}).
					Where("node_id = ? AND deleted_at IS NULL AND provider_record_id = ''", nodeID).
					Updates(map[string]interface{}{
						"trashed_at":  now,
						"deleted_at":  now,
						"sync_status": "disabled",
					}).Error; err != nil {
					return fmt.Errorf("移入回收站失败: %w", err)
				}
				// Provider records: left untouched (external to the platform)
				// Return domain to claimer, restore to active
				claimerID := node.OwnerID
				if node.ClaimerID != nil {
					claimerID = *node.ClaimerID
				}
				return tx.Model(&node).Updates(map[string]interface{}{
					"owner_id": claimerID,
					"status":   "active",
				}).Error
			})
		}
		// Claimer deleting: proceed to soft-delete below
	} // Subdomains: delete directly without archiving requirement

	// Block if has active children only (archived children don't block deletion)
	var childCount int64
	s.db.Model(&model.DomainNode{}).Where("parent_id = ? AND deleted_at IS NULL AND status = 'active'", nodeID).Count(&childCount)
	if childCount > 0 {
		return errors.New("无法删除含有子节点的节点，请先删除所有子域名")
	}

	// Delete platform records from provider before soft-delete (sync worker won't process deleted nodes)
	if err := s.deleteProviderRecords([]uint64{nodeID}); err != nil {
		return fmt.Errorf("删除DNS记录失败: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Mark all records as trashed (provider already deleted above)
		if err := tx.Model(&model.DNSRecord{}).
			Where("node_id = ? AND deleted_at IS NULL", nodeID).
			Updates(map[string]interface{}{
				"trashed_at":   now,
				"deleted_at":   now,
				"sync_status":  "trashed",
				"provider_record_id": "",
			}).Error; err != nil {
			return fmt.Errorf("移入回收站失败: %w", err)
		}

		// Soft-delete permission records
		if err := tx.Model(&model.DomainPermission{}).
			Where("domain_node_id = ?", nodeID).
			Update("status", "frozen").Error; err != nil {
			return fmt.Errorf("冻结权限失败: %w", err)
		}

		// Soft-delete the node itself
		return tx.Delete(&node).Error
	})
}

// MaterializeOrRestore attempts to materialize a subdomain node. If an archived node
// exists for the same full_domain:
//   - If targetUserID matches the archived node's owner → restore it and return
//   - If targetUserID differs and no active node exists → hard-delete archived and proceed
//   - If targetUserID differs but an active node exists → error "该子域名已被使用"
func (s *DomainService) MaterializeOrRestore(parentID uint64, host string, triggeredBy uint64, targetUserID uint64) (*model.DomainNode, error) {
	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return nil, errors.New("父节点不存在")
	}

	fullDomain := host + "." + parent.FullDomain

	// Check for existing node of any status
	var existing model.DomainNode
	if err := s.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		if existing.Status == "archived" {
			// If target is the same owner → restore
			if targetUserID == existing.OwnerID {
				if err := s.RestoreArchivedChild(existing.ID, triggeredBy); err != nil {
					return nil, err
				}
				// Reload the restored node
				s.db.First(&existing, existing.ID)
				return &existing, nil
			}
			// Different owner: check if any active node already exists
			var activeCount int64
			s.db.Model(&model.DomainNode{}).Where("full_domain = ? AND status = 'active' AND deleted_at IS NULL", fullDomain).Count(&activeCount)
			if activeCount > 0 {
				return nil, errors.New("该子域名已被使用")
			}
			// No active node → hard-delete the archived and proceed to create new
			s.db.Unscoped().Delete(&model.DNSRecord{}, "node_id = ?", existing.ID)
			if err := s.db.Unscoped().Delete(&existing).Error; err != nil {
				return nil, fmt.Errorf("删除归档节点失败: %w", err)
			}
		} else {
			// Active node already exists
			return &existing, nil
		}
	}

	// Verify records with this host exist under the parent
	var count int64
	s.db.Model(&model.DNSRecord{}).Where("node_id = ? AND host = ? AND deleted_at IS NULL", parentID, host).Count(&count)
	if count == 0 {
		return nil, errors.New("该主机名下没有DNS记录，无法转换为节点")
	}

	node := &model.DomainNode{
		Host:           host,
		FullDomain:     fullDomain,
		ParentID:       &parentID,
		OwnerID:        targetUserID,
		IsMaterialized: true,
		ProviderID:     parent.ProviderID,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(node).Error; err != nil {
			return err
		}

		// Move records to the new node and set host to "@"
		if err := tx.Model(&model.DNSRecord{}).
			Where("node_id = ? AND host = ? AND deleted_at IS NULL", parentID, host).
			Updates(map[string]interface{}{
				"node_id":     node.ID,
				"host":        "@",
				"own_node_id": node.ID,
			}).Error; err != nil {
			return err
		}

		// Transfer all sub-domain records (host ending with .<host>)
		// e.g., for host="zzuli", matches "image.zzuli", "1.image.zzuli", etc.
		suffix := "." + host
		var subRecords []model.DNSRecord
		if err := tx.Where("node_id = ? AND host LIKE ? AND host != ? AND deleted_at IS NULL", parentID, "%"+suffix, host).
			Find(&subRecords).Error; err != nil {
			return err
		}

		for _, rec := range subRecords {
			// Strip the .zzuli suffix to get relative host: "image.zzuli" → "image"
			newHost := strings.TrimSuffix(rec.Host, suffix)
			if err := tx.Model(&model.DNSRecord{}).Where("id = ?", rec.ID).Updates(map[string]interface{}{
				"node_id":     node.ID,
				"host":        newHost,
				"own_node_id": node.ID,
				"sync_status": "pending",
			}).Error; err != nil {
				return err
			}
		}

		// Set materialized_from to first record ID
		var firstRecord model.DNSRecord
		if err := tx.Where("node_id = ? AND host = ? AND deleted_at IS NULL", node.ID, "@").
			Order("id ASC").First(&firstRecord).Error; err == nil {
			node.MaterializedFrom = &firstRecord.ID
			tx.Model(node).Update("materialized_from", firstRecord.ID)
		}

		// Collect affected record IDs for logging
		var recordIDs []uint64
		tx.Model(&model.DNSRecord{}).
			Where("node_id = ? AND host = ? AND deleted_at IS NULL", node.ID, "@").
			Pluck("id", &recordIDs)

		recordIDsJSON := "[]"
		if len(recordIDs) > 0 {
			parts := make([]string, len(recordIDs))
			for i, id := range recordIDs {
				parts[i] = fmt.Sprintf("%d", id)
			}
			recordIDsJSON = "[" + strings.Join(parts, ",") + "]"
		}

		log := &model.NodeConversionLog{
			DomainNodeID: node.ID,
			Action:       "materialize",
			TriggeredBy:  triggeredBy,
			RecordIDs:    recordIDsJSON,
			Detail:       fmt.Sprintf("Made '%s' independent from %s", host, parent.FullDomain),
		}
		return tx.Create(log).Error
	})

	return node, err
}

// MaterializeNode converts an implicit host under a parent node into an explicit DomainNode.
// It is idempotent: if the node already exists, it returns the existing one.
func (s *DomainService) MaterializeNode(parentID uint64, host string, triggeredBy uint64) (*model.DomainNode, error) {
	return s.MaterializeOrRestore(parentID, host, triggeredBy, triggeredBy)
}

// DemoteNode converts an explicit materialized DomainNode back to an implicit subdomain.
// Prerequisites: no children, no permissions, no provider binding.
func (s *DomainService) DemoteNode(nodeID uint64, triggeredBy uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	if !node.IsMaterialized {
		return errors.New("该节点不是由记录转换而来，无法降级")
	}

	if node.ParentID == nil {
		return errors.New("根节点无法降级")
	}

	parentID := *node.ParentID

	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return errors.New("父节点不存在")
	}

	// Check no children
	var childCount int64
	s.db.Model(&model.DomainNode{}).Where("parent_id = ? AND deleted_at IS NULL", nodeID).Count(&childCount)
	if childCount > 0 {
		return errors.New("无法降级含有子节点的节点")
	}

	// Check no delegated permissions
	var permCount int64
	s.db.Model(&model.DomainPermission{}).Where("domain_node_id = ?", nodeID).Count(&permCount)
	if permCount > 0 {
		return errors.New("无法降级含有权限委托的节点")
	}

	// Provider binding: allow demotion if provider was inherited from parent (materialized node).
	// In that case the provider stays with the parent and we just clear it here.
	if node.ProviderID != nil {
		if node.ParentID == nil {
			return errors.New("无法降级绑定了DNS提供商的节点")
		}
		// Will be cleared during demotion — parent absorbs the provider relationship
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Re-parent records: move from this node back to parent
		var records []model.DNSRecord
		if err := tx.Where("node_id = ? AND deleted_at IS NULL", nodeID).Find(&records).Error; err != nil {
			return err
		}

		recordIDs := make([]uint64, 0, len(records))
		for _, rec := range records {
			// Restore host: "@" becomes node.Host (the subdomain name);
			// sub-records get the subdomain suffix restored as prefix.
			// e.g. node.Host="zzuli": host "@" → "zzuli", host "image" → "image.zzuli"
			var newHost string
			if rec.Host == "@" {
				newHost = node.Host
			} else {
				newHost = rec.Host + "." + node.Host
			}
			updates := map[string]interface{}{
				"node_id":     parentID,
				"host":        newHost,
				"own_node_id": nil,
				"sync_status": "pending",
			}
			if err := tx.Model(&model.DNSRecord{}).Where("id = ?", rec.ID).Updates(updates).Error; err != nil {
				return err
			}
			recordIDs = append(recordIDs, rec.ID)
		}

		// Soft-delete the node
		if err := tx.Delete(&node).Error; err != nil {
			return err
		}

		// Log conversion
		recordIDsJSON := "[]"
		if len(recordIDs) > 0 {
			parts := make([]string, len(recordIDs))
			for i, id := range recordIDs {
				parts[i] = fmt.Sprintf("%d", id)
			}
			recordIDsJSON = "[" + strings.Join(parts, ",") + "]"
		}

		log := &model.NodeConversionLog{
			DomainNodeID: nodeID,
			Action:       "dematerialize",
			TriggeredBy:  triggeredBy,
			RecordIDs:    recordIDsJSON,
			Detail:       fmt.Sprintf("Demoted '%s' back to subdomain records under %s", node.Host, parent.FullDomain),
		}
		return tx.Create(log).Error
	})
}

// ForceReclaim transfers ownership of a domain node from its current owner to the provider owner,
// and removes all delegated permissions. The caller must be the provider owner and cannot reclaim their own node.
func (s *DomainService) ForceReclaim(nodeID, providerOwnerID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	if node.ProviderID == nil {
		return errors.New("该节点未绑定DNS服务商")
	}

	var provider model.DNSProvider
	if err := s.db.First(&provider, *node.ProviderID).Error; err != nil {
		return errors.New("DNS服务商不存在")
	}

	if provider.UserID != providerOwnerID {
		return errors.New("无权操作此服务商")
	}

	if providerOwnerID == node.OwnerID {
		return errors.New("不能回收自己拥有的节点")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&node).Update("owner_id", providerOwnerID).Error; err != nil {
			return fmt.Errorf("转移所有权失败: %w", err)
		}

		if err := tx.Where("domain_node_id = ?", nodeID).Delete(&model.DomainPermission{}).Error; err != nil {
			return fmt.Errorf("删除权限记录失败: %w", err)
		}

		log := &model.OperationLog{
			UserID:     providerOwnerID,
			Action:     "force_reclaim",
			TargetType: "domain_node",
			TargetID:   &nodeID,
			Detail:     fmt.Sprintf("强制回收节点 %s，原所有者 %d", node.FullDomain, node.OwnerID),
		}
		return tx.Create(log).Error
	})
}

// ArchiveNode marks a domain node as archived, saving its provider reference and unbinding it.
func (s *DomainService) ArchiveNode(nodeID uint64, reason string) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	return s.db.Model(&node).Updates(map[string]interface{}{
		"status":               "archived",
		"archived_provider_id": node.ProviderID,
		"provider_id":          nil,
	}).Error
}

// ReactivateNode restores an archived node to active status and rebinds it to a provider.
func (s *DomainService) ReactivateNode(nodeID, providerID, userID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	if node.Status != "archived" {
		return errors.New("节点未处于归档状态")
	}

	var provider model.DNSProvider
	if err := s.db.Where("id = ? AND user_id = ?", providerID, userID).First(&provider).Error; err != nil {
		return errors.New("DNS服务商不存在或无权操作")
	}

	return s.db.Model(&node).Updates(map[string]interface{}{
		"status":               "active",
		"provider_id":          providerID,
		"archived_provider_id": nil,
	}).Error
}

// ListProviderDomains returns all domain nodes linked to a provider (including archived).
func (s *DomainService) ListProviderDomains(providerID, userID uint64) ([]model.DomainNode, error) {
	var provider model.DNSProvider
	if err := s.db.Where("id = ? AND user_id = ?", providerID, userID).First(&provider).Error; err != nil {
		return nil, errors.New("DNS服务商不存在或无权操作")
	}

	var nodes []model.DomainNode
	err := s.db.Unscoped().
		Where("provider_id = ? OR archived_provider_id = ?", providerID, providerID).
		Preload("Owner").
		Find(&nodes).Error
	return nodes, err
}

// GetConversionLogs returns the conversion history for a given node.
func (s *DomainService) GetConversionLogs(nodeID uint64) ([]model.NodeConversionLog, error) {
	var logs []model.NodeConversionLog
	err := s.db.Where("domain_node_id = ?", nodeID).
		Preload("Trigger").
		Order("id DESC").
		Find(&logs).Error
	return logs, err
}

// AdminTransferNode is like TransferNode but skips the permission check.
// Used by admin handlers where the caller is already verified as admin.
func (s *DomainService) AdminTransferNode(nodeID, targetUserID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var nodeIDs []uint64
		err := tx.Raw(`
			WITH RECURSIVE subtree AS (
				SELECT id FROM domain_nodes WHERE id = ? AND deleted_at IS NULL
				UNION ALL
				SELECT dn.id FROM domain_nodes dn JOIN subtree s ON dn.parent_id = s.id WHERE dn.deleted_at IS NULL
			)
			SELECT id FROM subtree
		`, nodeID).Scan(&nodeIDs).Error
		if err != nil {
			return fmt.Errorf("failed to find subtree: %w", err)
		}

		if err := tx.Model(&model.DomainNode{}).Where("id IN ?", nodeIDs).Update("owner_id", targetUserID).Error; err != nil {
			return fmt.Errorf("failed to transfer nodes: %w", err)
		}

		for _, nid := range nodeIDs {
			tx.Create(&model.DomainTransferLog{
				NodeID:     nid,
				FromUserID: node.OwnerID,
				ToUserID:   targetUserID,
			})
		}

		return nil
	})
}

// AdminBatchDeleteNodes deletes multiple leaf nodes in one call.
// Nodes that have children, DNS records, or do not exist are silently skipped.
func (s *DomainService) AdminBatchDeleteNodes(nodeIDs []uint64) (deleted int, skipped int, err error) {
	for _, nodeID := range nodeIDs {
		var node model.DomainNode
		if err := s.db.First(&node, nodeID).Error; err != nil {
			skipped++
			continue
		}

		var childCount int64
		s.db.Model(&model.DomainNode{}).Where("parent_id = ? AND deleted_at IS NULL", nodeID).Count(&childCount)
		if childCount > 0 {
			skipped++
			continue
		}

		var recordCount int64
		s.db.Model(&model.DNSRecord{}).Where("node_id = ? AND deleted_at IS NULL", nodeID).Count(&recordCount)
		if recordCount > 0 {
			skipped++
			continue
		}

		txErr := s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("domain_node_id = ?", nodeID).Delete(&model.DomainPermission{}).Error; err != nil {
				return err
			}
			return tx.Delete(&node).Error
		})
		if txErr == nil {
			deleted++
		} else {
			skipped++
		}
	}
	return
}

// BatchDeleteNodes deletes multiple leaf nodes owned by the given user.
// Nodes that are not owned, have children, DNS records, or do not exist are skipped.
func (s *DomainService) BatchDeleteNodes(nodeIDs []uint64, userID uint64) (deleted int, skipped int, err error) {
	for _, nodeID := range nodeIDs {
		var node model.DomainNode
		if err := s.db.First(&node, nodeID).Error; err != nil {
			skipped++
			continue
		}

		if node.OwnerID != userID {
			skipped++
			continue
		}

		var childCount int64
		s.db.Model(&model.DomainNode{}).Where("parent_id = ? AND deleted_at IS NULL", nodeID).Count(&childCount)
		if childCount > 0 {
			skipped++
			continue
		}

		var recordCount int64
		s.db.Model(&model.DNSRecord{}).Where("node_id = ? AND deleted_at IS NULL", nodeID).Count(&recordCount)
		if recordCount > 0 {
			skipped++
			continue
		}

		txErr := s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("domain_node_id = ?", nodeID).Delete(&model.DomainPermission{}).Error; err != nil {
				return err
			}
			return tx.Delete(&node).Error
		})
		if txErr == nil {
			deleted++
		} else {
			skipped++
		}
	}
	return
}

func (s *DomainService) GetTransferredAwayNodes(userID uint64) ([]model.DomainTransferLog, error) {
	var logs []model.DomainTransferLog

	// Subquery: latest transfer-out per node_id for this user
	latestSubQuery := s.db.Model(&model.DomainTransferLog{}).
		Select("node_id, MAX(id) as max_id").
		Where("from_user_id = ?", userID).
		Group("node_id")

	err := s.db.
		Joins("JOIN domain_nodes ON domain_nodes.id = domain_transfer_logs.node_id AND domain_nodes.deleted_at IS NULL").
		Joins("JOIN (?) AS latest ON latest.max_id = domain_transfer_logs.id", latestSubQuery).
		Where("domain_nodes.owner_id != ?", userID).
		Preload("Node").
		Preload("ToUser", func(db *gorm.DB) *gorm.DB { return db.Select("id,username,nickname,avatar") }).
		Order("domain_transfer_logs.created_at DESC").
		Find(&logs).Error
	return logs, err
}

func (s *DomainService) GetTransferHistory(nodeID uint64) ([]model.DomainTransferLog, error) {
	var logs []model.DomainTransferLog
	err := s.db.
		Where("node_id = ?", nodeID).
		Preload("FromUser", func(db *gorm.DB) *gorm.DB { return db.Select("id,username,nickname,avatar") }).
		Preload("ToUser", func(db *gorm.DB) *gorm.DB { return db.Select("id,username,nickname,avatar") }).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// ArchiveDomainTree archives a domain node and all its subtree owned by the user.
func (s *DomainService) ArchiveDomainTree(nodeID, userID uint64) error {
	if err := s.perm.RequireLevel(userID, nodeID, 3); err != nil {
		return err
	}

	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}
	if node.Status == "archived" {
		return errors.New("域名已归档")
	}

	// Collect ALL subtree node IDs (recursive CTE, regardless of owner)
	var nodeIDs []uint64
	s.db.Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM domain_nodes WHERE id = ? AND deleted_at IS NULL
			UNION ALL
			SELECT dn.id FROM domain_nodes dn
			JOIN subtree s ON dn.parent_id = s.id
			WHERE dn.deleted_at IS NULL
		) SELECT id FROM subtree
	`, nodeID).Scan(&nodeIDs)

	// Delete provider records before archiving (uses archived_provider_id for lookup)
	if err := s.deleteProviderRecords(nodeIDs); err != nil {
		return fmt.Errorf("删除DNS记录失败: %w", err)
	}

	now := time.Now()

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Mark all platform records in subtree as trashed
		if err := tx.Model(&model.DNSRecord{}).
			Where("node_id IN ? AND deleted_at IS NULL AND source = 'platform'", nodeIDs).
			Updates(map[string]interface{}{
				"trashed_at":         now,
				"deleted_at":         now,
				"sync_status":        "archived",
				"provider_record_id": "",
			}).Error; err != nil {
			return fmt.Errorf("归档DNS记录失败: %w", err)
		}

		// Archive all nodes in subtree
		if err := tx.Model(&model.DomainNode{}).
			Where("id IN ?", nodeIDs).
			Updates(map[string]interface{}{
				"status":               "archived",
				"archived_by":          userID,
				"archived_at":          now,
				"archived_provider_id": gorm.Expr("provider_id"),
				"provider_id":          nil,
			}).Error; err != nil {
			return err
		}

		// Freeze permissions on these nodes
		if err := tx.Model(&model.DomainPermission{}).
			Where("domain_node_id IN ?", nodeIDs).
			Update("status", "frozen").Error; err != nil {
			return err
		}

		return nil
	})
}

// RestoreDomainTree restores an archived domain tree back to active status.
func (s *DomainService) RestoreDomainTree(nodeID, userID uint64) error {
	if err := s.perm.RequireLevel(userID, nodeID, 3); err != nil {
		return err
	}

	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}
	if node.Status != "archived" {
		return errors.New("域名未归档")
	}

	// Prevent restore if the provider was deleted (no fallback available)
	if node.ArchivedProviderID != nil {
		var provider model.DNSProvider
		if err := s.db.First(&provider, *node.ArchivedProviderID).Error; err != nil {
			return errors.New("该域名的DNS提供商已被删除，无法恢复，请重新添加服务商后再试")
		}
	}

	// Check if an active node already exists for the same full_domain
	var conflict model.DomainNode
	if err := s.db.Where("full_domain = ? AND status = 'active' AND deleted_at IS NULL AND id != ?", node.FullDomain, nodeID).First(&conflict).Error; err == nil {
		return fmt.Errorf("域名 %s 已存在活跃节点 (ID=%d)，无法恢复", node.FullDomain, conflict.ID)
	}

	// Collect ALL archived subtree nodes (regardless of owner)
	var nodeIDs []uint64
	s.db.Raw(`
		WITH RECURSIVE subtree AS (
			SELECT id FROM domain_nodes WHERE id = ? AND deleted_at IS NULL AND status = 'archived'
			UNION ALL
			SELECT dn.id FROM domain_nodes dn
			JOIN subtree s ON dn.parent_id = s.id
			WHERE dn.deleted_at IS NULL AND dn.status = 'archived'
		) SELECT id FROM subtree
	`, nodeID).Scan(&nodeIDs)

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Restore nodes
		if err := tx.Model(&model.DomainNode{}).
			Where("id IN ?", nodeIDs).
			Updates(map[string]interface{}{
				"status":               "active",
				"archived_by":          0,
				"archived_at":          nil,
				"provider_id":          gorm.Expr("archived_provider_id"),
				"archived_provider_id": nil,
			}).Error; err != nil {
			return err
		}

		// Unfreeze permissions
		if err := tx.Model(&model.DomainPermission{}).
			Where("domain_node_id IN ? AND status = 'frozen'", nodeIDs).
			Update("status", "active").Error; err != nil {
			return err
		}

		return nil
	})
}

// ListArchivedChildren returns all archived children under a root domain node.
func (s *DomainService) ListArchivedChildren(rootNodeID, callerID uint64) ([]model.DomainNode, error) {
	var root model.DomainNode
	if err := s.db.First(&root, rootNodeID).Error; err != nil {
		return nil, errors.New("节点不存在")
	}

	if err := s.perm.RequireLevel(callerID, rootNodeID, 4); err != nil {
		return nil, err
	}

	var nodes []model.DomainNode
	pattern := "%." + root.FullDomain
	err := s.db.Where("status = 'archived' AND deleted_at IS NULL AND full_domain LIKE ?", pattern).
		Preload("Owner").
		Order("full_domain ASC").
		Find(&nodes).Error
	return nodes, err
}

// RestoreArchivedChild restores an archived child node to active status under a new root.
func (s *DomainService) RestoreArchivedChild(nodeID, callerID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}
	if node.Status != "archived" {
		return errors.New("节点未处于归档状态")
	}

	// Check if an active node already exists for the same full_domain
	var conflict model.DomainNode
	if err := s.db.Where("full_domain = ? AND status = 'active' AND deleted_at IS NULL", node.FullDomain).First(&conflict).Error; err == nil {
		return fmt.Errorf("该子域名已存在活跃节点 (ID=%d)，请使用现有节点或先删除后再恢复", conflict.ID)
	}

	// Find root domain (same full_domain suffix, not archived)
	var root model.DomainNode
	err := s.db.Where("full_domain = ? AND status != 'archived' AND deleted_at IS NULL",
		node.FullDomain[strings.Index(node.FullDomain, ".")+1:]).First(&root).Error
	if err != nil {
		return errors.New("未找到可用的根域名")
	}

	if err := s.perm.RequireLevel(callerID, root.ID, 4); err != nil {
		return err
	}

	if node.ArchivedProviderID != nil {
		var provider model.DNSProvider
		if err := s.db.Unscoped().First(&provider, *node.ArchivedProviderID).Error; err != nil {
			return errors.New("该域名的DNS提供商已被删除，无法恢复，请重新添加服务商后再试")
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":               "active",
			"parent_id":            &root.ID,
			"archived_by":          0,
			"archived_at":          nil,
			"archived_provider_id": nil,
		}
		if node.ArchivedProviderID != nil {
			updates["provider_id"] = *node.ArchivedProviderID
		}
		tx.Where("node_id = ? AND host = ? AND deleted_at IS NULL", root.ID, node.Host).
			Delete(&model.DNSRecord{})
		if err := tx.Model(&model.DomainPermission{}).
			Where("domain_node_id = ? AND status = 'frozen'", nodeID).
			Update("status", "active").Error; err != nil {
			return err
		}
		return tx.Model(&node).Updates(updates).Error
	})
}

// GetArchivedDomains returns all archived domain nodes owned by the user.
func (s *DomainService) GetArchivedDomains(userID uint64) ([]model.DomainNode, error) {
	var nodes []model.DomainNode
	err := s.db.Where("owner_id = ? AND status = 'archived' AND deleted_at IS NULL", userID).
		Order("archived_at DESC").Find(&nodes).Error
	return nodes, err
}

// SyncFromProvider fetches records from the DNS provider and reconciles them with local records.
// It ONLY syncs records for domains that are ALREADY managed locally - it does NOT create new domain nodes.
func (s *DomainService) SyncFromProvider(domainID, userID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, domainID).Error; err != nil {
		return errors.New("域名节点不存在")
	}

	// Ensure the domain exists and is actively managed (not deleted/archived)
	if node.DeletedAt.Valid {
		return errors.New("域名已被删除，无法同步")
	}
	if node.Status == "archived" {
		return errors.New("域名已归档，请先恢复后再同步")
	}

	if err := s.perm.RequireLevel(userID, domainID, 2); err != nil {
		return err
	}

	if node.ProviderID == nil {
		return errors.New("该域名未绑定DNS服务商")
	}

	p, err := s.providerSvc.GetDNSProvider(*node.ProviderID)
	if err != nil {
		return fmt.Errorf("获取DNS服务商失败: %w", err)
	}

	providerRecords, err := p.ListRecords(node.FullDomain)
	if err != nil {
		return fmt.Errorf("获取服务商记录失败: %w", err)
	}

	// Get all local records (including platform-created ones that have provider_record_id)
	var localRecords []model.DNSRecord
	if err := s.db.Where("node_id = ? AND deleted_at IS NULL", domainID).Find(&localRecords).Error; err != nil {
		return fmt.Errorf("查询本地记录失败: %w", err)
	}

	// Build map by provider_record_id and by host+type (for fallback matching)
	localByPRID := make(map[string]*model.DNSRecord)
	localByHostType := make(map[string]*model.DNSRecord)
	for i := range localRecords {
		rec := &localRecords[i]
		if rec.ProviderRecordID != "" {
			localByPRID[rec.ProviderRecordID] = rec
		}
		key := rec.Host + "|" + rec.RecordType
		localByHostType[key] = rec
	}

	// Tracks which local PRIDs were found on provider (to detect deletions)
	foundPRIDs := make(map[string]bool)

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, pr := range providerRecords {
			// Map RR to host: "@" or domainName itself maps to node.Host (root domain)
			host := pr.Host
			if pr.Host == "@" || pr.Host == node.FullDomain {
				host = node.Host
			} else {
				suffix := "." + node.FullDomain
				if strings.HasSuffix(pr.Host, suffix) {
					host = strings.TrimSuffix(pr.Host, suffix)
				}
			}

			priority := convertPriority(pr.Priority)

			if existing, ok := localByPRID[pr.RecordID]; ok {
				// Update if different
				needsUpdate := existing.Host != host ||
					existing.Value != pr.Value ||
					existing.TTL != int(pr.TTL) ||
					(existing.Priority == nil && priority != nil) ||
					(existing.Priority != nil && *existing.Priority != *priority)

				if needsUpdate {
					updates := map[string]interface{}{
						"host":               host,
						"value":              pr.Value,
						"ttl":                int(pr.TTL),
						"priority":           priority,
						"sync_status":        "synced",
						"provider_record_id": pr.RecordID,
					}
					if err := tx.Model(&model.DNSRecord{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
						return fmt.Errorf("更新记录失败: %w", err)
					}
				}
				foundPRIDs[pr.RecordID] = true
			} else if existing, ok := localByHostType[host+"|"+pr.Type]; ok {
				// Fallback: match by host+type, update provider_record_id and values
				needsUpdate := existing.Host != host ||
					existing.Value != pr.Value ||
					existing.TTL != int(pr.TTL) ||
					(existing.Priority == nil && priority != nil) ||
					(existing.Priority != nil && *existing.Priority != *priority)

				updates := map[string]interface{}{
					"provider_record_id": pr.RecordID,
					"sync_status":        "synced",
				}
				if needsUpdate {
					updates["host"] = host
					updates["value"] = pr.Value
					updates["ttl"] = int(pr.TTL)
					updates["priority"] = priority
				}
				if err := tx.Model(&model.DNSRecord{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
					return fmt.Errorf("更新记录ID失败: %w", err)
				}
				foundPRIDs[pr.RecordID] = true
			} else {
				// Create new local record
				newRec := &model.DNSRecord{
					NodeID:           domainID,
					Host:             host,
					RecordType:       pr.Type,
					Value:            pr.Value,
					TTL:              int(pr.TTL),
					Priority:         priority,
					Line:             pr.Line,
					Enabled:          true,
					ProviderRecordID: pr.RecordID,
					SyncStatus:       "synced",
					Source:           "provider",
				}
				if err := tx.Create(newRec).Error; err != nil {
					return fmt.Errorf("创建记录失败: %w", err)
				}
				foundPRIDs[pr.RecordID] = true
			}
		}

		// Hard-delete local provider records that exist in our DB but were NOT found on provider
		// Platform-created records (source='platform') are kept even if not found on provider
		for prID, rec := range localByPRID {
			if !foundPRIDs[prID] && rec.Source == "provider" {
				if err := tx.Unscoped().Delete(&model.DNSRecord{}, "id = ?", rec.ID).Error; err != nil {
					return fmt.Errorf("删除记录失败: %w", err)
				}
			}
		}

		return nil
	})
}

// ReturnSubdomainToClaimer returns a subdomain to its parent owner and trashes the user's DNS records.
func (s *DomainService) ReturnSubdomainToClaimer(nodeID, userID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}
	if node.OwnerID != userID {
		return errors.New("你不是该子域名的所有者")
	}
	if node.ParentID == nil || *node.ParentID == 0 {
		return errors.New("根域名不能归还，请使用删除功能")
	}

	// Find parent to get claimer
	var parent model.DomainNode
	if err := s.db.First(&parent, *node.ParentID).Error; err != nil {
		return errors.New("父节点不存在")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Transfer ownership back to parent's owner
		if err := tx.Model(&model.DomainNode{}).Where("id = ?", nodeID).Update("owner_id", parent.OwnerID).Error; err != nil {
			return err
		}

		// Move user's records on this node to trash
		now := time.Now()
		if err := tx.Model(&model.DNSRecord{}).
			Where("node_id = ? AND created_by = ? AND deleted_at IS NULL", nodeID, userID).
			Updates(map[string]interface{}{
				"trashed_at":  now,
				"deleted_at":  now,
				"sync_status": "disabled",
			}).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetDomainNodesWithProvider returns all active nodes bound to a DNS provider
func (s *DomainService) GetDomainNodesWithProvider() ([]model.DomainNode, error) {
	var nodes []model.DomainNode
	err := s.db.Where("provider_id IS NOT NULL AND status = 'active' AND deleted_at IS NULL").
		Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
