package mysql

import (
	"fmt"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLDB struct {
	db *gorm.DB
}

func NewMySQLDB() database.Database {
	return &MySQLDB{}
}

func (m *MySQLDB) Connect(config *database.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	var err error
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

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
