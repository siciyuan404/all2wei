package repository

import (
	"strings"

	"all2wei/internal/config"
	"all2wei/internal/model"

	"gorm.io/gorm"
	"github.com/glebarez/sqlite"
)

func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&model.User{}, &model.Material{})
	if err != nil {
		return nil, err
	}

	migrateFolderField(db)

	return db, nil
}

func migrateFolderField(db *gorm.DB) {
	var count int64
	db.Model(&model.Material{}).Where("folder = '' OR folder IS NULL").Count(&count)
	if count == 0 {
		return
	}

	var materials []model.Material
	db.Where("folder = '' OR folder IS NULL").Find(&materials)

	for _, m := range materials {
		parts := strings.Split(m.VideoKey, "/")
		folder := ""
		if len(parts) >= 3 {
			folder = strings.Join(parts[:len(parts)-1], "/")
		} else if len(parts) == 2 {
			folder = parts[0]
		}
		if folder != "" {
			db.Model(&m).Update("folder", folder)
		}
	}
}
