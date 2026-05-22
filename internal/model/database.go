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

// migrateDomainNodesUniqueIndex replaces the unique index on full_domain with
// a regular index. This allows soft-deleted rows to coexist with active rows
// without violating a unique constraint. Application code ensures uniqueness
// for active rows via SELECT ... FOR UPDATE before inserts.
func migrateDomainNodesUniqueIndex(db *gorm.DB) error {
	// Drop the old unique index on full_domain if it exists
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

	// Ensure a regular (non-unique) index exists on full_domain
	var regularIndexExists int64
	db.Raw(`SELECT COUNT(*) FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = 'domain_nodes'
		AND index_name = 'idx_domain_nodes_full_domain'`).Scan(&regularIndexExists)
	if regularIndexExists == 0 {
		log.Println("[Migration] Creating regular index on full_domain...")
		if err := db.Exec("CREATE INDEX idx_domain_nodes_full_domain ON domain_nodes (full_domain)").Error; err != nil {
			return fmt.Errorf("create regular index: %w", err)
		}
	}

	return nil
}
