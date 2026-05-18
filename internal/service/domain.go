package service

import (
	"errors"
	"fmt"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type DomainService struct {
	db *gorm.DB
}

func NewDomainService(db *gorm.DB) *DomainService {
	return &DomainService{db: db}
}

func (s *DomainService) CreateNode(parentID uint64, host string, ownerID uint64) (*model.DomainNode, error) {
	var parent model.DomainNode
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return nil, errors.New("parent node not found")
	}

	if parent.OwnerID != ownerID {
		return nil, errors.New("you do not own the parent domain")
	}

	fullDomain := host + "." + parent.FullDomain

	var existing model.DomainNode
	if err := s.db.Where("full_domain = ?", fullDomain).First(&existing).Error; err == nil {
		return nil, errors.New("domain already exists")
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
	var nodes []model.DomainNode
	err := s.db.Where("owner_id = ?", userID).
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("owner_id = ?", userID)
		}).
		Preload("Records").
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
	var node model.DomainNode
	err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).
		Preload("Children").
		Preload("Records").
		First(&node).Error
	if err != nil {
		return nil, errors.New("domain node not found or access denied")
	}
	return &node, nil
}

func (s *DomainService) FindNodeByDomain(domain string, userID uint64) (*model.DomainNode, string, error) {
	var node model.DomainNode
	err := s.db.Where("owner_id = ? AND (full_domain = ? OR ? LIKE CONCAT('%.', full_domain))",
		userID, domain, domain).
		Order("LENGTH(full_domain) DESC").
		First(&node).Error

	if err != nil {
		return nil, "", errors.New("domain not found or access denied")
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
		return errors.New("node not found")
	}
	if node.OwnerID != ownerID {
		return errors.New("you do not own this domain")
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
		return errors.New("node not found")
	}
	if node.OwnerID != userID {
		return errors.New("you do not own this domain")
	}

	var childCount int64
	s.db.Model(&model.DomainNode{}).Where("parent_id = ?", nodeID).Count(&childCount)
	if childCount > 0 {
		return errors.New("cannot delete node with children")
	}

	var recordCount int64
	s.db.Model(&model.DNSRecord{}).Where("node_id = ?", nodeID).Count(&recordCount)
	if recordCount > 0 {
		return errors.New("cannot delete node with DNS records")
	}

	return s.db.Delete(&node).Error
}
