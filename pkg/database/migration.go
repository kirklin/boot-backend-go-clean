package database

import (
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.User{},
		// 添加其他实体模型
	)
}
