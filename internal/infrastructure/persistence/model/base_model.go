package model

import (
	"database/sql"
	"time"

	"github.com/kirklin/boot-backend-go-clean/pkg/utils/snowflake"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID int64 `gorm:"primaryKey;autoIncrement:false"`

	CreatedAt time.Time      `gorm:"type:TIMESTAMP with time zone;not null"`
	UpdatedAt sql.NullTime   `gorm:"type:TIMESTAMP with time zone;null"`
	DeletedAt gorm.DeletedAt `gorm:"type:TIMESTAMP with time zone;index"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == 0 {
		m.ID = snowflake.NextID()
	}
	m.CreatedAt = time.Now().UTC()
	return
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	return
}
