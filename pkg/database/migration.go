package database

import (
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence/model"
)

func AutoMigrate(db Database) error {
	return db.DB().AutoMigrate(
		&model.UserDTO{},
		// 添加其他实体模型
	)
}
