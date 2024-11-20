package model

import (
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

type UserDTO struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Username  string     `json:"username" gorm:"unique;not null"`
	Email     string     `json:"email" gorm:"unique;not null"`
	Password  string     `json:"-" gorm:"not null"` // 不在 JSON 中显示密码
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"` // 用于逻辑删除
}

// TableName specifies the actual table name for UserDTO
func (*UserDTO) TableName() string {
	return "users" // 设置表名为 "users"
}

// ConvertToEntity 将 UserDTO 转换为领域实体 User
func (dto *UserDTO) ConvertToEntity() *entity.User {
	return &entity.User{
		ID:        dto.ID,
		Username:  dto.Username,
		Email:     dto.Email,
		Password:  dto.Password,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
		DeletedAt: dto.DeletedAt,
	}
}

// ConvertFromEntity 从领域实体 User 转换为 UserDTO
func (dto *UserDTO) ConvertFromEntity(u *entity.User) {
	dto.ID = u.ID
	dto.Username = u.Username
	dto.Email = u.Email
	dto.Password = u.Password
	dto.CreatedAt = u.CreatedAt
	dto.UpdatedAt = u.UpdatedAt
	dto.DeletedAt = u.DeletedAt
}
