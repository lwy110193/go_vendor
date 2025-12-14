package database

import (
	"time"
)

type BaseModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id;comment:id"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime;index;autoCreateTime;not null;comment:创建时间"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime;index;autoCreateTime;not null;comment:更新时间"`
	DeletedAt time.Time `gorm:"column:deleted_at;type:datetime;default:null;comment:删除时间"`
}
