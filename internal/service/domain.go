package service

import (
	"errors"
	"fmt"
	"strings"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type DomainService struct {
	db   *gorm.DB
	perm *PermissionService
}

func NewDomainService(db *gorm.DB, perm *PermissionService) *DomainService {
	return &DomainService{db: db, perm: perm}
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
	if err := s.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		return nil, errors.New("域名已存在")
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
		Preload("Records").
		Preload("Owner").
		Find(&nodes).Error

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
	if err := s.db.Preload("Children").Preload("Records").First(&node, nodeID).Error; err != nil {
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

func (s *DomainService) TransferNode(nodeID, ownerID, targetUserID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	if err := s.perm.RequireLevel(ownerID, nodeID, 4); err != nil {
		return err
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

		return nil
	})
}

func (s *DomainService) DeleteNode(nodeID, userID uint64) error {
	var node model.DomainNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return errors.New("节点不存在")
	}

	if err := s.perm.RequireLevel(userID, nodeID, 4); err != nil {
		return err
	}

	var childCount int64
	s.db.Model(&model.DomainNode{}).Where("parent_id = ? AND deleted_at IS NULL", nodeID).Count(&childCount)
	if childCount > 0 {
		return errors.New("无法删除含有子节点的节点，请先删除所有子域名")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Cascade-delete DNS records belonging to this node
		if err := tx.Where("node_id = ?", nodeID).Delete(&model.DNSRecord{}).Error; err != nil {
			return fmt.Errorf("删除DNS记录失败: %w", err)
		}

		// Delete permission records for this node
		if err := tx.Where("domain_node_id = ?", nodeID).Delete(&model.DomainPermission{}).Error; err != nil {
			return fmt.Errorf("删除权限记录失败: %w", err)
		}

		// Delete the node itself
		return tx.Delete(&node).Error
	})
}

// MaterializeNode converts an implicit host under a parent node into an explicit DomainNode.
// It is idempotent: if the node already exists, it returns the existing one.
func (s *DomainService) MaterializeNode(parentID uint64, host string, triggeredBy uint64) (*model.DomainNode, error) {
	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return nil, errors.New("父节点不存在")
	}

	fullDomain := host + "." + parent.FullDomain

	// Idempotent: return existing if already materialized
	var existing model.DomainNode
	if err := s.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		return &existing, nil
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
		OwnerID:        parent.OwnerID,
		IsMaterialized: true,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(node).Error; err != nil {
			return err
		}

		// Update exact host match records: set own_node_id
		if err := tx.Model(&model.DNSRecord{}).
			Where("node_id = ? AND host = ? AND deleted_at IS NULL", parentID, host).
			Update("own_node_id", node.ID).Error; err != nil {
			return err
		}

		// Set materialized_from to first record ID
		var firstRecord model.DNSRecord
		if err := tx.Where("node_id = ? AND host = ? AND deleted_at IS NULL", parentID, host).
			Order("id ASC").First(&firstRecord).Error; err == nil {
			node.MaterializedFrom = &firstRecord.ID
			tx.Model(node).Update("materialized_from", firstRecord.ID)
		}

		// Collect affected record IDs for logging
		var recordIDs []uint64
		tx.Model(&model.DNSRecord{}).
			Where("node_id = ? AND host = ? AND deleted_at IS NULL", parentID, host).
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
			Detail:       fmt.Sprintf("Materialized '%s' from records under %s", host, parent.FullDomain),
		}
		return tx.Create(log).Error
	})

	return node, err
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

	// Check no provider binding
	if node.ProviderID != nil {
		return errors.New("无法降级绑定了DNS提供商的节点")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Re-parent records: move from this node back to parent
		var records []model.DNSRecord
		if err := tx.Where("node_id = ? AND deleted_at IS NULL", nodeID).Find(&records).Error; err != nil {
			return err
		}

		recordIDs := make([]uint64, 0, len(records))
		for _, rec := range records {
			newHost := node.Host
			if rec.Host != "@" {
				newHost = node.Host + "." + rec.Host
			}

			updates := map[string]interface{}{
				"node_id":     parentID,
				"host":        newHost,
				"own_node_id": nil,
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
			Detail:       fmt.Sprintf("Demoted '%s' back to implicit under %s", node.Host, node.FullDomain),
		}
		return tx.Create(log).Error
	})
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
