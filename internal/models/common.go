package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleAdmin       = "admin"
	RoleAdvanced    = "advanced"
	DefaultPageSize = "10"
	QueryParamID    = "id"
)

// DBModel represent generic database columns
type DBModel struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
