package model

import (
	"time"

	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/pkg/utils/snowflake"
)

type BaseModel struct {
	ID int64 `gorm:"primaryKey;autoIncrement:false" json:"id,string"`

	CreatedAt time.Time      `gorm:"type:TIMESTAMP with time zone;not null" json:"created_at"`
	UpdatedAt *time.Time     `gorm:"type:TIMESTAMP with time zone;null" json:"updated_at,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"type:TIMESTAMP with time zone;index" json:"-"`
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == 0 {
		m.ID = snowflake.NextID()
	}
	m.CreatedAt = time.Now().UTC()
	return
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	now := time.Now().UTC()
	m.UpdatedAt = &now
	return
}
