package model

import (
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/pkg/utils/timeutil"
)

type UserDTO struct {
	BaseModel
	Username  string  `json:"username" gorm:"unique;not null"`
	Email     string  `json:"email" gorm:"unique;not null"`
	Password  string  `json:"-" gorm:"not null"` // 不在 JSON 中显示密码
	AvatarURL *string `json:"avatar_url,omitempty"`
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
		AvatarURL: dto.AvatarURL,
		CreatedAt: dto.CreatedAt,
		UpdatedAt: dto.UpdatedAt,
		DeletedAt: timeutil.ToSqlNullTime(dto.DeletedAt),
	}
}

// ConvertFromEntity 从领域实体 User 转换为 UserDTO
func (dto *UserDTO) ConvertFromEntity(u *entity.User) {
	dto.ID = u.ID
	dto.Username = u.Username
	dto.Email = u.Email
	dto.Password = u.Password
	dto.AvatarURL = u.AvatarURL
	dto.CreatedAt = u.CreatedAt
	dto.UpdatedAt = u.UpdatedAt
	dto.DeletedAt = timeutil.ToGormDeletedAt(u.DeletedAt)
}
