package repository

import (
	"all2wei/internal/config"
	"all2wei/internal/model"

	"gorm.io/gorm"
	"github.com/glebarez/sqlite" // 纯 Go SQLite，无需 CGO
)

func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移
	err = db.AutoMigrate(&model.User{}, &model.Material{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
