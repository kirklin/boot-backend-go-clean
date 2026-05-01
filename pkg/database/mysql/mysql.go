package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

type MySQLDB struct {
	db *gorm.DB
}

func NewMySQLDB() database.Database {
	return &MySQLDB{}
}

func (m *MySQLDB) Connect(config *database.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	var err error
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance for pooling config: %w", err)
	}

	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetimeMinutes) * time.Minute)

	return nil
}

func (m *MySQLDB) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

func (m *MySQLDB) DB() *gorm.DB {
	return m.db
}
