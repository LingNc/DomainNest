package model

import (
	"fmt"
	"log"
	"time"

	"domainnest/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(3 * time.Minute)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database auto-migration...")
	if err := db.AutoMigrate(
		&User{},
		&DNSProvider{},
		&DomainNode{},
		&DNSRecord{},
		&OperationLog{},
		&PasswordReset{},
		&SystemSetting{},
		&DomainPermission{},
		&RAMToken{},
		&FriendRequest{},
		&Friendship{},
		&Message{},
		&InviteLog{},
		&EmailVerification{},
		&NodeConversionLog{},
		&SyncLog{},
		&DomainTransferLog{},
		&FilterPreset{},
		&NotificationSetting{},
		&InviteCode{},
	); err != nil {
		return err
	}

	if err := migrateDomainNodesUniqueIndex(db); err != nil {
		return fmt.Errorf("migrateDomainNodesUniqueIndex: %w", err)
	}

	return nil
}

// migrateDomainNodesUniqueIndex replaces the unique index on full_domain alone
// with a composite approach using a generated stored column. This allows soft-deleted
// rows (deleted_at IS NOT NULL) to coexist with active rows (deleted_at IS NULL)
// without violating the unique constraint.
//
// The generated column full_domain_key is:
//   - full_domain when deleted_at IS NULL (active row)
//   - CONCAT(id, ':', full_domain) when deleted_at IS NOT NULL (soft-deleted)
//
// Since id is unique per row, soft-deleted rows never conflict. Active rows
// still enforce uniqueness on full_domain alone.
func migrateDomainNodesUniqueIndex(db *gorm.DB) error {
	// 1. Drop the old unique index on full_domain if it exists
	var indexExists int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'domain_nodes'
		AND index_name = 'idx_domain_nodes_full_domain'`).Scan(&indexExists)
	if indexExists > 0 {
		log.Println("[Migration] Dropping old unique index idx_domain_nodes_full_domain...")
		if err := db.Exec("DROP INDEX idx_domain_nodes_full_domain ON domain_nodes").Error; err != nil {
			return fmt.Errorf("drop old index: %w", err)
		}
	}

	// 2. Add the generated stored column if it doesn't exist
	var columnExists int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'domain_nodes'
		AND column_name = 'full_domain_key'`).Scan(&columnExists)
	if columnExists == 0 {
		log.Println("[Migration] Adding generated column full_domain_key...")
		if err := db.Exec(`
			ALTER TABLE domain_nodes
			ADD COLUMN full_domain_key VARCHAR(512) GENERATED ALWAYS AS (
				CASE WHEN deleted_at IS NULL THEN full_domain
				ELSE CONCAT(CAST(id AS CHAR), ':', full_domain)
				END
			) STORED NOT NULL
		`).Error; err != nil {
			return fmt.Errorf("add generated column: %w", err)
		}
	}

	// 3. Add the new unique index on full_domain_key if it doesn't exist
	var newIndexExists int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'domain_nodes'
		AND index_name = 'idx_domain_nodes_full_domain_uniq'`).Scan(&newIndexExists)
	if newIndexExists == 0 {
		log.Println("[Migration] Creating new unique index idx_domain_nodes_full_domain_uniq...")
		if err := db.Exec("CREATE UNIQUE INDEX idx_domain_nodes_full_domain_uniq ON domain_nodes (full_domain_key)").Error; err != nil {
			return fmt.Errorf("create new unique index: %w", err)
		}
	}

	return nil
}
