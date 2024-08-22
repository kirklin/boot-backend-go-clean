package postgres

import (
	"fmt"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB struct {
	db *gorm.DB
}

func NewPostgresDB() database.Database {
	return &PostgresDB{}
}

func (p *PostgresDB) Connect(config *database.Config) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		config.Host, config.User, config.Password, config.DBName, config.Port, config.SSLMode)

	var err error
	p.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	return nil
}

func (p *PostgresDB) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

func (p *PostgresDB) DB() *gorm.DB {
	return p.db
}
