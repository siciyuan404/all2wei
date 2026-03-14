package repository

import (
	"all2wei/internal/model"

	"gorm.io/gorm"
)

type MaterialRepository struct {
	db *gorm.DB
}

func NewMaterialRepository(db *gorm.DB) *MaterialRepository {
	return &MaterialRepository{db: db}
}

func (r *MaterialRepository) Create(material *model.Material) error {
	return r.db.Create(material).Error
}

func (r *MaterialRepository) GetByID(id uint) (*model.Material, error) {
	var material model.Material
	err := r.db.First(&material, id).Error
	if err != nil {
		return nil, err
	}
	return &material, nil
}

func (r *MaterialRepository) GetByUserID(userID uint) ([]model.Material, error) {
	var materials []model.Material
	err := r.db.Where("user_id = ? AND status = ?", userID, "active").
		Order("created_at DESC").
		Find(&materials).Error
	return materials, err
}

func (r *MaterialRepository) Update(material *model.Material) error {
	return r.db.Save(material).Error
}

func (r *MaterialRepository) Delete(id uint) error {
	return r.db.Model(&model.Material{}).Where("id = ?", id).Update("status", "deleted").Error
}
