package timeutil

import (
	"time"

	"gorm.io/gorm"
)

// ToGormDeletedAt 将 *time.Time 转换为 gorm.DeletedAt
func ToGormDeletedAt(t *time.Time) gorm.DeletedAt {
	if t == nil {
		return gorm.DeletedAt{}
	}
	return gorm.DeletedAt{
		Time:  *t,
		Valid: true,
	}
}

// ToTimePointer 将 gorm.DeletedAt 转换为 *time.Time
func ToTimePointer(t gorm.DeletedAt) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
