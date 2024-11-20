package entity

import (
	"errors"
	"regexp"
	"time"
)

type User struct {
	ID        uint       `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Password  string     `json:"-"` // 不在 JSON 中显示密码
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"` // 用于逻辑删除
}

// Validate 验证用户实体
func (u *User) Validate() error {
	if u.Username == "" {
		return errors.New("username cannot be empty")
	}
	if u.Email == "" {
		return errors.New("email cannot be empty")
	}
	if !isValidEmail(u.Email) {
		return errors.New("invalid email format")
	}
	if len(u.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return emailRegex.MatchString(email)
}
