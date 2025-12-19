package models

import (
	"time"

	"gorm.io/gorm"
)

type Teacher struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string         `json:"name" gorm:"not null;size:100"`
	Surname   string         `json:"surname" gorm:"not null;size:100"`
	Email     string         `json:"email" gorm:"unique;not null;size:255"`
	Phone     string         `json:"phone,omitempty" gorm:"size:20"`
	UserID    *uint          `json:"user_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Teacher) TableName() string {
	return "teachers"
}
