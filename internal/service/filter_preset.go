package service

import (
	"domainnest/internal/errs"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type FilterPresetService struct {
	db *gorm.DB
}

func NewFilterPresetService(db *gorm.DB) *FilterPresetService {
	return &FilterPresetService{db: db}
}

func (s *FilterPresetService) ListPresets(userID uint64) ([]model.FilterPreset, error) {
	var presets []model.FilterPreset
	err := s.db.Where("user_id = ?", userID).Order("id ASC").Find(&presets).Error
	return presets, err
}

func (s *FilterPresetService) SavePreset(userID uint64, name string, filters string) (*model.FilterPreset, error) {
	if name == "" {
		return nil, errs.New(errs.PresetNameRequired, "预设名称不能为空")
	}
	preset := &model.FilterPreset{
		UserID:  userID,
		Name:    name,
		Filters: filters,
	}
	if err := s.db.Create(preset).Error; err != nil {
		return nil, err
	}
	return preset, nil
}

func (s *FilterPresetService) DeletePreset(presetID, userID uint64) error {
	result := s.db.Where("id = ? AND user_id = ?", presetID, userID).Delete(&model.FilterPreset{})
	if result.RowsAffected == 0 {
		return errs.New(errs.PresetNotFound, "预设不存在")
	}
	return result.Error
}
