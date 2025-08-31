package timeutil

import (
	"database/sql"

	"gorm.io/gorm"
)

// ToGormDeletedAt 将 sql.NullTime 转换为 gorm.DeletedAt
func ToGormDeletedAt(t sql.NullTime) gorm.DeletedAt {
	if !t.Valid {
		return gorm.DeletedAt{}
	}
	return gorm.DeletedAt{
		Time:  t.Time,
		Valid: true,
	}
}

// ToSqlNullTime 将 gorm.DeletedAt 转换为 sql.NullTime
func ToSqlNullTime(t gorm.DeletedAt) sql.NullTime {
	if !t.Valid {
		return sql.NullTime{}
	}
	return sql.NullTime{
		Time:  t.Time,
		Valid: true,
	}
}
