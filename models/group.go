// models/group.go или models/common.go
package models

import (
	"time"

	"gorm.io/gorm"
)

type Group struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string         `json:"name" gorm:"not null;size:100"`
	Code      string         `json:"code" gorm:"unique;not null;size:20"`
	Students  []Student      `json:"students,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
