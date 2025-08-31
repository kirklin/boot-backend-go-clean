package model

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID int64 `gorm:"primaryKey"`

	CreatedAt time.Time      `gorm:"type:TIMESTAMP with time zone;not null"`
	UpdatedAt sql.NullTime   `gorm:"type:TIMESTAMP with time zone;null"`
	DeletedAt gorm.DeletedAt `gorm:"type:TIMESTAMP with time zone;index"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = time.Now().UTC()
	return
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	return
}
