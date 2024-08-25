package database

import (
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

func AutoMigrate(db Database) error {
	return db.DB().AutoMigrate(
		&entity.User{},
		// 添加其他实体模型
	)
}
