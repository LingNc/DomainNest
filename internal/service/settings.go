package service

import (
	"encoding/json"
	"sync"

	"domainnest/internal/config"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type SettingsService struct {
	db    *gorm.DB
	cache sync.Map
}

func NewSettingsService(db *gorm.DB) *SettingsService {
	return &SettingsService{db: db}
}

func (s *SettingsService) Get(category string) (string, error) {
	if cached, ok := s.cache.Load(category); ok {
		return cached.(string), nil
	}
	var setting model.SystemSetting
	if err := s.db.Where("key = ?", category).First(&setting).Error; err != nil {
		return "", nil
	}
	s.cache.Store(category, setting.Value)
	return setting.Value, nil
}

func (s *SettingsService) Set(category, value string) error {
	setting := model.SystemSetting{Key: category, Value: value}
	err := s.db.Where("key = ?", category).Assign(model.SystemSetting{Value: value}).FirstOrCreate(&setting).Error
	if err == nil {
		s.cache.Store(category, value)
	}
	return err
}

func (s *SettingsService) GetSMTPConfig() *config.SMTPConfig {
	raw, err := s.Get("smtp")
	if err != nil || raw == "" {
		return nil
	}
	var cfg config.SMTPConfig
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return nil
	}
	if cfg.Host == "" {
		return nil
	}
	return &cfg
}
