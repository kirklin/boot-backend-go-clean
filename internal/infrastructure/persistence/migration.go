package persistence

import (
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence/model"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

// AutoMigrate runs GORM AutoMigrate for all registered domain models.
// Add new model structs here as they are created.
//
// NOTE: AutoMigrate is convenient for development but should NOT be used
// in production. Use a proper migration tool (e.g. golang-migrate) instead.
func AutoMigrate(db database.Database) error {
	return db.DB().AutoMigrate(
		&model.UserDTO{},
		// Add new models here:
		// &model.PostDTO{},
	)
}
