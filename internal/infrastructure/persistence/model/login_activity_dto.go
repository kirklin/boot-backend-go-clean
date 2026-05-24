package model

import (
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// LoginActivityDTO is the database representation of a login activity record.
type LoginActivityDTO struct {
	BaseModel
	UserID    int64     `gorm:"index;not null"`                               // 用户 ID
	LoginAt   time.Time `gorm:"type:TIMESTAMP with time zone;not null;index"` // 登录时间
	IPAddress string    `gorm:"size:45"`                                      // IPv4/IPv6 地址
	UserAgent string    `gorm:"type:text"`                                    // 客户端 User-Agent

	// GeoIP 解析字段
	Country  string `gorm:"size:64"`       // 国家
	Province string `gorm:"size:64;index"` // 省份（建索引，方便按省分布统计）
	City     string `gorm:"size:64;index"` // 城市（建索引，方便按城市分布统计）
	ISP      string `gorm:"size:64"`       // 运营商
}

// TableName sets the database table name.
func (*LoginActivityDTO) TableName() string {
	return "login_activities"
}

// ConvertToEntity converts the DTO to a domain entity.
func (dto *LoginActivityDTO) ConvertToEntity() *entity.LoginActivity {
	return &entity.LoginActivity{
		ID:        dto.ID,
		UserID:    dto.UserID,
		LoginAt:   dto.LoginAt,
		IPAddress: dto.IPAddress,
		UserAgent: dto.UserAgent,
		Country:   dto.Country,
		Province:  dto.Province,
		City:      dto.City,
		ISP:       dto.ISP,
		CreatedAt: dto.CreatedAt,
	}
}

// ConvertFromEntity populates the DTO from a domain entity.
func (dto *LoginActivityDTO) ConvertFromEntity(a *entity.LoginActivity) {
	dto.ID = a.ID
	dto.UserID = a.UserID
	dto.LoginAt = a.LoginAt
	dto.IPAddress = a.IPAddress
	dto.UserAgent = a.UserAgent
	dto.Country = a.Country
	dto.Province = a.Province
	dto.City = a.City
	dto.ISP = a.ISP
	dto.CreatedAt = a.CreatedAt
}
