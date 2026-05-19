package service

import (
	"errors"
	"fmt"

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
	s.db.Model(&model.DomainNode{}).Where("parent_id = ?", nodeID).Count(&childCount)
	if childCount > 0 {
		return errors.New("无法删除含有子节点的节点")
	}

	var recordCount int64
	s.db.Model(&model.DNSRecord{}).Where("node_id = ?", nodeID).Count(&recordCount)
	if recordCount > 0 {
		return errors.New("无法删除含有DNS记录的节点")
	}

	return s.db.Delete(&node).Error
}
